package db

import (
	"sync"
	"time"
)

const (
	// 为了防止一直存在读操作，导致写操作不能执行，所有设定一个比例，
	// 最多读maxReadWriteRatio次就一定要写一次
	maxReadWriteRatio          = 2 << 7
	expirationTimeLimitForData = time.Hour * 1
)

var (
	CacheMap = make(map[string]*SafeData) //string(contestId) -> ContestInfo
	// 保证删除数据时数据的完整性,在创建数据表的时候就添加键值对contestId -> chan，
	// 当第一次调用该比赛的更新函数的时候，就获取令牌，防止同时有多个更新函数在执行
	ChMap = make(map[string]chan struct{})
	// 等待更新完成
	WaitCh      = make(map[string]chan struct{})
	CacheDataMu sync.RWMutex
)

type SafeData struct {
	LatestReqTime time.Time   //记录最近一次请求的时间
	TimeMu        *sync.Mutex //在对LatestReqTime变量进行读写操作时保证数据一致性
	*ContestInfo
	ReadToken chan struct{}     //具有缓存的通道，用于读操作获取令牌
	DataMu    *sync.RWMutex     //读写锁，保护数据完整性
	SetCh     chan *ContestInfo //传输的数据对象太大，为了避免拷贝，应该传递指针
	sync.Once
	// FirstCh   chan struct{}//第一次更新数据
}

type ContestInfo struct {
	ContestName   string
	StartTime     string
	EndTime       string
	ProblemMap    map[string]ProblemInfo //题目名->数据
	UserInfoSlice []UserInfo
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

func cacheUpdateLoop() {
	// 更新定时器的间隔
	updateInterval := time.Minute * 3
	updateTicker := time.NewTicker(updateInterval)
	defer updateTicker.Stop()
	// 检测缓存数据是否过期的定时器的时间间隔
	// 又因为可能大部分时间CacheMap都有RLock()操作，所以获得lock()的机会较少
	// 所以时间间隔应该小一点
	cleanCacheInterval := time.Hour
	cleanTicker := time.NewTicker(cleanCacheInterval)
	defer cleanTicker.Stop()
	// // updateTicker.C是一个chan，返回值的类型使time.Time
	// // 因为我们用不到返回值，所以不必写出这样：
	// // for v := range updateTicker.C {}
	// for range updateTicker.C {
	// 	updateFunc() //还是说是 go updateFunc()???
	// }

	for {
		select {
		case <-updateTicker.C:
			iterateContest() //应该等待该函数执行完(这里应该实惠阻塞等待该函数执行完的吧)
		case <-cleanTicker.C:
			go cleanCacheData() //如果该函数是
		}
	}
}

// 为了保证更新数据(写操作)不会花费太多的时间,写操作中查询数据库的时候也能够进行读操作，
// 所以应该保证只有"赋值"的时候写锁是锁上的,所以更新的数据应该通过管道传输给一个
// 管道，然后启动一个协程去专门完成赋值的操作

func req() {
	CacheDataMu.RLock() //在读锁锁上期间，就不会删除数据了
	defer CacheDataMu.RUnlock()
	if _, ok := CacheMap["contestId"]; ok { //该场比赛初始化完成
		// 进行读操作
	} else {
		// 进行初始化操作
		UpdateFunc("contestId")
	}
}

func UpdateFunc(contestId string) {
	//
	if _, ok := ChMap[contestId]; ok { //该场比赛初始化未完成
		select {
		case ChMap[contestId] <- struct{}{}: //获取令牌，从而开始更新权限
			// ChMap[contestId]管道不应该关闭
		default:
			// 数据更新完成后关闭该通道
			<-WaitCh[contestId]
		}
		// select {
		// case ch <- struct{}{}:
		// 	//如果通道阻塞，那么会执行default
		// 	//如果通道关闭，那么会报错
		// 	// 那是不是说还得再用一个chan，然后
		// default:
		// 	<-waitUpdateDone
		// }

		// <-ChMap[contestId] //阻塞，等待更新函数执行完成
	}
	return
	// else { //自己就是那个更新函数

	// 	defer func() {
	// 		close(ChMap[contestId]) //广播，表示初始化后的首次更新函数已经执行完成
	// 	}()
	// }
	if _, ok := ChMap[contestId]; ok { //

	}
	CacheMap[contestId].DataMu.Lock()
	defer CacheMap[contestId].DataMu.Unlock()
	CacheMap[contestId].ContestInfo = <-CacheMap[contestId].SetCh
	for len(CacheMap[contestId].ReadToken) > 0 {
		<-CacheMap[contestId].ReadToken
	}
}

func iterateContest() {

}

// 因为清理缓存数据不要求有多快，所以不必每场比赛都单独开一个协程专门处理
func cleanCacheData() {
	CacheDataMu.Lock()
	defer CacheDataMu.Unlock() //等到该函数执行完再解锁也行，该函数执行花不了多少时间
	for k, v := range CacheMap {
		if time.Since(v.LatestReqTime) > expirationTimeLimitForData {
			delete(CacheMap, k) //清理缓存数据
		}
	}
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
