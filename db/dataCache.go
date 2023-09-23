package db

import (
	"strconv"
	"sync"
	"time"
)

const (
	// 为了防止一直存在读操作，导致写操作不能执行，所有设定一个比例，
	// 最多读maxReadWriteRatio次就一定要写一次
	maxReadWriteRatio          = 2 << 7
	expirationTimeLimitForData = time.Minute //time.Hour * 1
)

var (
	CacheMap = make(map[string]*SafeData) //string(contestId) -> ContestInfo
	// 保证删除数据时数据的完整性,在创建数据表的时候就添加键值对contestId -> chan，
	// 当第一次调用该比赛的更新函数的时候，就获取令牌，防止同时有多个更新函数在执行
	UpdateCh = make(map[string]chan struct{})
	// 等待更新完成
	WaitCh      = make(map[string]chan struct{})
	CacheDataMu = new(sync.RWMutex)
	cleanWaitCh = make(chan struct{}, 1)
)

type SafeData struct {
	LatestReqTime time.Time   //记录最近一次请求的时间
	TimeMu        *sync.Mutex //在对LatestReqTime变量进行读写操作时保证数据一致性
	*ContestInfo
	ReadToken chan struct{}     //具有缓存的通道，用于读操作获取令牌
	DataMu    *sync.RWMutex     //读写锁，保护数据完整性
	setCh     chan *ContestInfo //传输的数据对象太大，为了避免拷贝，应该传递指针
	waitCh    chan struct{}     //保证同一时间只有一个updateFunc协程在执行
}

type ContestInfo struct {
	ContestName   string
	StartTime     string
	EndTime       string
	ProblemMap    map[string]ProblemInfo //题目名->数据
	UserInfoSlice []UserInfo
	Error         error //没有错误就是nil
}

type ProblemInfo struct { //测试和样例文件的路径(需要查询很多次，但是很少会改变,后期再优化吧)就不缓存了
	TotalSubmit  string
	AcSubmit     string
	TimeLimit    int64 //运行时间限制,ms
	MemoryLimit  int64 //运行内存限制,MB
	MaxFileLimit int64 //选手提交文件大小限制,KB
	TestFiles    []TestFile
	ExampleFile  []ExampleFile
}

type TestFile struct {
	Id                  string
	InputPath           string
	OutPutPath          string
	OutputFileHashValue string
}
type ExampleFile struct {
	Id         string
	InputPath  string
	OutPutPath string
}

type UserInfo struct {
	StudentNumber string
	StudentName   string
	SchoolName    string
	Status        string //状态，也许可以不用预处理，交给前端来处理
}

// 当CacheMap中不存在想要查询比赛的缓存数据的时候调用
func InitContestCache(contestId string) {
	select {
	// 创建该场比赛的时候就应该这里的UpdateCh[contestId]初始化好，
	// 清理缓存数据的时候应该将这里的UpdateCh[contestId]管道清空为零
	case UpdateCh[contestId] <- struct{}{}: // 这里抢到了第一次更新数据的活
		logger.Sugar().Infoln(contestId + " start init ")
		// 隐藏bug,这里对CacheMap进行写，cacheUpdateLoop进行读，可能存在竞态
		CacheMap[contestId] = &SafeData{
			LatestReqTime: time.Now(),
			ReadToken:     make(chan struct{}, maxReadWriteRatio),
			setCh:         make(chan *ContestInfo),
			TimeMu:        new(sync.Mutex),
			DataMu:        new(sync.RWMutex),
			waitCh:        make(chan struct{}, 1),
		}
		updateFunc(contestId)
		// 通知同样想要进行第一次更新但是未能获取到令牌而阻塞的InitContestCache函数结束阻塞
		// 清除数据后，应该重新给该场比赛创建一个WaitCh[contestId]
		close(WaitCh[contestId])
	// updateFunc(contestId)
	default:
		logger.Sugar().Info(contestId + " waiting init")
		<-WaitCh[contestId] //创建该场比赛的时候就应该这里的WaitCh[contestId]初始化好
	}
}

// 向数据库请求数据，向CacheMap[contestId].SetCh管道发送得到的数据
func updateFunc(contestId string) {
	select {
	case CacheMap[contestId].waitCh <- struct{}{}:
	default:
		return
	}
	// 应该保证会有数据发送过去，避免协程发生死锁
	go setValue(contestId)
	// 向数据库获取数据的过程中是不需要加锁的，这期间还可以用旧的数据来完成前端的请求(如果还有读令牌的话)
	// 因为全部都是查询操作，所以不需要用到事务
	contestInfo := &ContestInfo{
		ProblemMap:    make(map[string]ProblemInfo),
		UserInfoSlice: make([]UserInfo, 0),
	}
	contestIdNum, err := strconv.Atoi(contestId)
	if err != nil {
		contestInfo.Error = err
		logger.Sugar().Errorln(err)
	} else {
		contest := Contests{ID: contestIdNum}
		result := DB.Model(&Contests{}).First(&contest)
		if result.Error != nil {
			contestInfo.Error = result.Error
			logger.Sugar().Errorln(result.Error)
		} else {
			// 先测试一下原理，等会再继续查询更多的数据
			contestInfo.ContestName = contest.ContestName
			contestInfo.StartTime = contest.StartTime
			contestInfo.EndTime = contest.EndTime
		}
	}
	// 因为该通道没事设置缓存，所以不一定马上发送成功，要等到setValue获取到该场比赛的写锁才行
	CacheMap[contestId].setCh <- contestInfo
}

// 监听管道，进行写操作
func setValue(contestId string) {
	// 既然我能够加上锁，那就表示现在现在没有CacheMap[contestId].DataMu.RLock()
	CacheMap[contestId].DataMu.Lock()
	CacheMap[contestId].ContestInfo = <-CacheMap[contestId].setCh
	CacheMap[contestId].DataMu.Unlock()
	// 重新分配读令牌
	// 就算这里一边清空，前端请求中一边输入，这也是可行的，因为此时数据已经赋值完成
	// 经过测试，只要前端有待发送的信号，这里的len就会不断变化，从而一直清空，
	// 但是不必担心ReadToken失去作用，因为这里的清空速率总比前端发送过来的请求速率要快的多吧,
	// 所以for循环不会变成死循环
	for len(CacheMap[contestId].ReadToken) > 0 {
		<-CacheMap[contestId].ReadToken
	}
}

// 现在还有一个问题，就是更新协程和清理缓存的协程应该最多同时只能有一个
func cacheUpdateLoop() {
	// 更新定时器的间隔
	updateInterval := time.Second * 30 //time.Minute * 3
	updateTicker := time.NewTicker(updateInterval)
	defer updateTicker.Stop()
	// 检测缓存数据是否过期的定时器的时间间隔
	// 又因为可能大部分时间CacheMap都有RLock()操作，所以获得lock()的机会较少
	// 所以时间间隔应该小一点
	cleanCacheInterval := time.Minute //time.Hour
	cleanTicker := time.NewTicker(cleanCacheInterval)
	defer cleanTicker.Stop()
	for {
		select {
		case <-updateTicker.C:
			logger.Sugar().Infoln("update ticker start")
			// 好像不加锁不行？？？因为虽然执行这里的case就不会同时执行下面的case，
			// 但是下面case中是一个协程
			CacheDataMu.RLock()
			// 因为这里是启动一个协程去更新，所以实际上也不会锁太长时间,可以看成是非阻塞的
			for k, _ := range CacheMap {
				contestId := k //要特别小心协程的参数传递
				go updateFunc(contestId)
			}
			CacheDataMu.RUnlock()
		case <-cleanTicker.C:
			logger.Sugar().Infoln("clean Ticker start")
			go cleanCacheData() //如果该函数是
		}
	}
}

// 因为清理缓存数据不要求有多快，所以不必每场比赛都单独开一个协程专门处理
func cleanCacheData() {
	// 如果想要一个函数同一时间只有一个协程在执行，可以使用这种结构来保证
	select {
	// 获取令牌，执行本函数
	case cleanWaitCh <- struct{}{}:
		// 不能获取令牌，返回
	default:
		return
	}
	// 这里实际上是有点难等的，因为只有CacheMap不被任何操作依赖的时候，才能加锁
	// 所以可能出现前一个协程还没等到，后一个协程就又启动了，
	// 后面优化的时候应该确保同一时间只有一个该协程
	CacheDataMu.Lock()
	defer CacheDataMu.Unlock() //等到该函数执行完再解锁也行，该函数执行花不了多少时间
	for k, v := range CacheMap {
		if time.Since(v.LatestReqTime) > expirationTimeLimitForData {
			delete(CacheMap, k) //清理缓存数据
			for len(UpdateCh[k]) > 0 {
				<-UpdateCh[k] //释放令牌
			}
			WaitCh[k] = make(chan struct{}) //创建一个新令牌
		}
	}
	// 释放令牌(既然能来到这一步，就表示它拿到令牌了，所以释放令牌的时候不会阻塞在这里)
	<-cleanWaitCh
}

// 经过验证：
// 1.父协程结束不会使子协程的运行停止
// 2.for{select{case...}}结构中，如果一个case是耗时操作，那么整个结构都会等待它操作完成，
// 在这期间即使其他case接收到信号，也会阻塞直到耗时操作的case处理完成
// func goSync() {
// 	mutilOpt := time.NewTicker(time.Minute * 3)
// 	singleOpt := time.NewTicker(time.Second * 5)
// 	for {
// 		select {
// 		case <-mutilOpt.C:
// 			time.Sleep(time.Minute * 10)
// 			fmt.Println("mutil operate")
// 		case <-singleOpt.C:
// 			fmt.Println("singlr operate")
// 		}
// 	}
// }
// =====================================================================================
// var CacheMap[String]*SafeData = make(map[string]*SafeData)
// var CacheDataMu sync.RWMutex
// var WaitCh = make(map[string]chan struct{})//不参与更新，则等待
// var UpdateCh = make(map[string]chan struct{},1)//参与更新，则获取令牌

// type SafeData struct {
// 	LatestReqTime time.Time   //记录最近一次请求的时间
// 	TimeMu        *sync.Mutex //在对LatestReqTime变量进行读写操作时保证数据一致性
// 	ContestInfo *ContestInfo
// 	ReadToken chan struct{}     //具有缓存的通道，用于读操作获取令牌
// 	DataMu    *sync.RWMutex     //读写锁，保护数据完整性
// 	SetCh     chan *ContestInfo //传输的数据对象太大，为了避免拷贝，应该传递指针
// 	done	chan struct{}//用于结束监听该场比赛的协程
// }

// 下面来梳理一下流程：
// 前端发送获取缓存数据的请求
// 1	CacheDataMu.RLock()将读锁锁上，防止在这期间对CacheMap进行删除操作
// 2.0	if _,ok := WaitCh[contestId];判断当前请求的比赛初始化完成没有
// 2.1	如果完成，进行读操作	ok = false
// 2.1.1	CacaeMap[contestId].TimeMu.Lock()
// 2.1.2	CacaeMap[contestId].LatestReqTime = time.Now()
// 2.1.3	CacaeMap[contestId].TimeMu.Unlock()
// 2.1.4	CacaeMap[contestId].DataMu.RLock()
// 2.1.5	response.ContestInfo = *CacaeMap[contestId].ContestInfo
// 2.1.6	CacaeMap[contestId].DataMu.RUnlock()
// 2.2	如果没完成，执行数据更新操作 UpdateFunc(contestId)	ok = true
// 为了避免重复写代码，2.2这一步应该在2.1之前，这样就不用重复写2.1.x的步骤了
// 3	defer CacheDataMu.RUnlock()将锁解锁

// 数据更新操作	:	UpdateFunc(contestId)
// 1.0	select {case UpdateCh <- struct{}{}: 数据更新操作	default: <- WaitCh}
// 1.1	执行第一个case，如果获取到令牌(能够向UpdateCh中发送信号)
// 1.1.1	执行更新操作
// 1.1.2	初始化该场比赛CacheMap[contestId] = &SafeData{}
// 1.1.3	启动一个协程，用于监听CacheMap[contestId].SetCh管道 go setValue(contestId)
// 1.1.4	向数据库获取数据	databaseUpdate(contestId)
// 1.1.5	close(WaitCh)
// 1.2	执行default,不能够获取到令牌
// 1.2.1	<- WaitCh 等待数据更新完成

// 向数据库获取数据并更新到ContestInfo中	：	databaseUpdate(contestId)
// 1	获取数据库数据,将数据生成为一个&ContestInfo{}
// 2	将该指针发送到SetCh管道中SetCh <- &ContestInfo{}

// 初始化完该场比赛后就应该启动的协程，从SetCh管道中获取数据，然后赋值 setValue(contestId)
// 0	for{select{case <- CacheMap[contestId].SetCh: 数据赋值 case <- CacheMap[contestId].done: 结束该协程}}
// 1.0	如果CacheMap[contestId].SetCh接收到信号
// 1.1	CacheMap[contestId].DataMu.Lock()
// 1.2	CacheMap[contestId].ContestInfo = <- CacheMap[contestId].SetCh
// 1.3	CacheMap[contestId].DataMu.Unlock()
// 2.0	如果CacheMap[contestId].done接收到信号
// 2.1	结束该协程 return

// 还需要额外启动一个协程，用于定期更新缓存数据已经清理缓存数据	：	cacheDataLoop()
// for {select{}}

// 应该是前端调用initContestCache(contestId)
// 然后initContestCache(contestId)再调用updateFunc(contestId)
// 然后updateFunc(contestId)再调用setValue(contestId)
