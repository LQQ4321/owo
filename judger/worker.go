package judger

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// 设置请求缓存的大小，最多可以接收 maxWaiting 个请求
const maxWaiting = 1 << 10

type Request struct {
	Time           int64 //提交代码的运行时间
	Memory         int64
	Language       string
	SourceFilePath string
	ExecuteFileId  string // python have only source file
	InputPath      []string
	Optional       int //(1 : complier,2 : execute,3 : complier and execute)

	OjServerTime int64 //ojserver从发送到等待的时间，或者说阻塞时间
	TickInterval int64
}

type Response struct {
	Results []Result
	Error   error
}

// worker配置
type Config struct {
	Parallelism int
}

// default tick interval 500ms
const defaultTickInterval = 500 * time.Millisecond

// 等待judger两分钟，其实应该是 ClockInterval = 所有程序运行时间 + 额外的时间开销
// 这里为了方便就暂定为两分钟
const defaultClockInterval = time.Minute * 2

type waiter struct {
	tickInterval   time.Duration //查询的间隔时间(每隔一段时间查询一下超时了没有，类似于心跳)
	clockTimeLimit time.Duration //阻塞时间
}

func (w *waiter) wait(ctx context.Context, proc *process) bool {
	start := time.Now()
	if w.tickInterval == 0 {
		w.tickInterval = defaultTickInterval
	}
	if w.clockTimeLimit == 0 {
		w.clockTimeLimit = defaultClockInterval
	}
	ticker := time.NewTicker(w.tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done(): //client
			close(proc.done)
			proc.Response = Response{
				Error: fmt.Errorf("client cancelled before execute"),
			}
			return true
		case <-proc.done:
			return true
		case <-ticker.C:
			if time.Since(start) > w.clockTimeLimit {
				close(proc.done)
				proc.Response = Response{
					Error: fmt.Errorf("judger exceeded time limit"),
				}
				return false
			}
		}
	}
}

type process struct {
	done chan struct{}
	Response
}

// 对外接口
type Worker interface {
	Start()
	Submit(context.Context, *Request) <-chan Response
	Shutdown()
}

type workRequest struct {
	*Request
	context.Context
	resultCh chan<- Response
}

type worker struct {
	parallelism int //因为并行的最大程度就是处理器的数量,设置超过处理器的数量就没有意义了
	startOnce   sync.Once
	stopOnce    sync.Once
	wg          sync.WaitGroup
	workCh      chan workRequest
	done        chan struct{}
}

// 这里是具体操作，用来处理请求
func (w *worker) workDoCmd(pc context.Context, req *Request) Response {
	// ojserver将请求发送到judger端后，不可能无限等待，所以要设置超时情况下的处理
	ctx, cancel := context.WithCancel(pc)
	// 防止程序意外退出？所以使用defer，双重保险？？？
	defer cancel()
	wait := &waiter{
		clockTimeLimit: time.Duration(req.OjServerTime),
		tickInterval:   time.Duration(req.TickInterval),
	}
	proc := &process{
		done: make(chan struct{}),
	}
	// tow close() and two cancel(),not close of closed channel
	defer func() {
		if _, ok := <-proc.done; ok {
			close(proc.done)
		}
	}()
	go judgerServer(req, proc)
	wait.wait(ctx, proc)
	// close(proc.done)
	// cancel judger execute
	cancel()
	return proc.Response
}

// 无限循环，等待请求的到来
func (w *worker) loop() {
	defer w.wg.Done()
	for {
		select {
		case req, ok := <-w.workCh:
			if !ok {
				logger.Info(fmt.Errorf("loop chan fail"))
				return
			}
			select {
			case <-req.Context.Done():
				req.resultCh <- Response{
					Error: fmt.Errorf("cancelled before execute"),
				}
			default:
				req.resultCh <- w.workDoCmd(req.Context, req.Request)
			}
		case <-w.done:
			return
		}
	}
}

// 开始服务
func (w *worker) Start() {
	w.startOnce.Do(func() {
		w.workCh = make(chan workRequest, maxWaiting)
		w.done = make(chan struct{})
		w.wg.Add(w.parallelism)
		for i := 0; i < w.parallelism; i++ {
			go w.loop()
		}
	})
}

// 将请求发送到workCh
func (w *worker) Submit(ctx context.Context, req *Request) <-chan Response {
	ch := make(chan Response, 1)
	select {
	case w.workCh <- workRequest{
		Request:  req,
		Context:  ctx,
		resultCh: ch,
	}:
	default:
		ch <- Response{
			Error: fmt.Errorf("worker queue is full"),
		}
	}
	return ch
}

// Shutdown waits all worker to finish
func (w *worker) Shutdown() {
	w.stopOnce.Do(func() {
		close(w.done)
		w.wg.Wait()
	})
}

func New(conf Config) Worker {
	return &worker{
		parallelism: conf.Parallelism,
	}
}
