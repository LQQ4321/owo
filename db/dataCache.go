package db

import (
	"fmt"
	"sync"
	"time"
)

const (
	// 为了防止一直存在读操作，导致写操作不能执行，所有设定一个比例，
	// 最多读maxReadWriteRatio次就一定要写一次
	maxReadWriteRatio          = 2 << 7
	expirationTimeLimitForData = time.Hour * 1
)

var CacheMap map[string]*SafeData = make(map[string]*SafeData) //string(contestId) -> ContestInfo
var CacheDataMu sync.RWMutex

type SafeData struct {
	LatestReqTime time.Time //记录最近一次请求的时间
	// TimeMu        *sync.Mutex //在对LatestReqTime变量进行读写操作时保证数据一致性
	*ContestInfo
	ReadToken chan struct{}     //具有缓存的通道，用于读操作获取令牌
	DataMu    *sync.RWMutex     //读写锁，保护数据完整性
	SetCh     chan *ContestInfo //传输的数据对象太大，为了避免拷贝，应该传递指针
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

// 为了保证更新数据(写操作)不会花费太多的时间,写操作中查询数据库的时候也能够进行读操作，
// 所以应该保证只有"赋值"的时候写锁是锁上的,所以更新的数据应该通过管道传输给一个
// 管道，然后启动一个协程去专门完成赋值的操作
func updateFunc(contestId string) {
	CacheMap[contestId].DataMu.Lock()
	defer CacheMap[contestId].DataMu.Unlock()
	CacheMap[contestId].ContestInfo = <-CacheMap[contestId].SetCh
	for len(CacheMap[contestId].ReadToken) > 0 {
		<-CacheMap[contestId].ReadToken
	}
}

func cacheUpdateLoop() {
	// 更新定时器的间隔
	updateInterval := time.Minute * 1
	updateTicker := time.NewTicker(updateInterval)
	defer updateTicker.Stop()
	// 检测缓存数据是否过期的定时器的时间间隔
	cleanCacheInterval := time.Hour * 8
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
			cleanCacheData() //有可能
		}
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
func goSync() {
	mutilOpt := time.NewTicker(time.Minute * 3)
	singleOpt := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-mutilOpt.C:
			time.Sleep(time.Minute * 10)
			fmt.Println("mutil operate")
		case <-singleOpt.C:
			fmt.Println("singlr operate")
		}
	}
}
