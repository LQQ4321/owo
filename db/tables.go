package db

import (
	"strconv"
	"time"

	"github.com/LQQ4321/owo/config"
)

// 结构唯一
type Contests struct {
	ID          int    `gorm:"primaryKey"` //这里应该不算用到gorm(所以不用导入)，可以通过反射得到字符串
	ContestName string //比赛名称
	CreatorName string //创造该场比赛的管理员
	CreateTime  string //比赛创建时间
	StartTime   string //比赛开始时间
	EndTime     string //比赛结束时间
}

type Managers struct {
	ID          int    `gorm:"primaryKey"`
	ManagerName string //管理员名称
	Password    string //密码
}

// 结构相同，表名不同
type Problems struct {
	ID int `gorm:"primaryKey"`
	//很少会改变的数据，前端只请求一次，如果管理员改变了它，那么选手可以重启程序来获取更新的数据
	ProblemName string //题目名称
	Pdf         bool   //题目是否上传了pdf文件
	// 下面的字段前端虽然只需要用到一次，但是后端会经常用到，所以也可以缓存
	TimeLimit    int64  //运行时间限制,ms
	MemoryLimit  int64  //运行内存限制,MB
	MaxFileLimit int64  //选手提交文件大小限制,KB
	TestFiles    string //题目的测试文件信息，各文件信息使用"|"分隔，例：编号|输入文件路径|输出文件路径|sha256
	ExampleFiles string //题目的样例文件信息，各文件信息使用"|"分隔,例：编号|输入文件路径|输出文件路径
	//比赛期间经常会改变的数据，需要缓存
	SubmitTotal int64 //提交该题目的人数
	SubmitAc    int64 //通过该题目的人数

}

type Users struct {
	ID            int    `gorm:"primaryKey"`
	StudentNumber string //唯一
	StudentName   string //不唯一
	SchoolName    string
	Password      string
	LoginTime     string
	Status        string //该场比赛的题目状态,分割符使用"|",因为里面要包含时间，时间有分割符":"
}

type Submits struct {
	ID            int    `gorm:"primaryKey"`
	StudentNumber string //唯一，可以关联查找，然后得到StudentName和SchoolName
	SubmitTime    string //提交时间
	ProblemName   string //题目名称
	Language      string //语言
	Status        string //状态
	RunTime       string //单位ms
	RunMemory     string //单位MB
	FileSize      string //单位KB
}

type News struct {
	ID         int    `gorm:"primaryKey"`
	IsManager  bool   //发送该条消息的人员类型，管理者或者选手,默认是false，也就是默认选手
	Identifier string //发送该条消息的人员标识，managerName或者studentNumber
	Text       string //发送的文本信息
	SendTime   string //发送时间
}

// 第一个参数接收int和string类型的值
func GetTableName(id interface{}, tableSuffix string) string {
	if num, ok := id.(int); ok {
		return config.TABLE_PREFIX + strconv.Itoa(num) + tableSuffix
	}
	return config.TABLE_PREFIX + id.(string) + tableSuffix
}

// 这里有必要缓存的数据是会经常改变而且有大量请求需求的数据
var CacheMap map[string]ContestInfo = make(map[string]ContestInfo) //string(contestId) -> ContestInfo

// 下面的成员别忘了初始化(所以上面的map的value到底要不要用指针*ContestInfo呀,应该可以不用)
// 为了方便，下面所有数据的查询都在一个协程中完成
type ContestInfo struct {
	// 因为LatestReqTime也涉及读写操作，所以判断"有必要查询时间"的时候要将其锁起来,
	// 因为可能要等待获取令牌，所以判断每场比赛的"有必要查询时间"时都可以安排一个协程
	LatestReqTime time.Time //最近一次请求的时间
	//因为这三者也有可能在比赛期间改变，为了及时得到更新数据，所以也需要缓存
	ContestName string
	StartTime   string
	EndTime     string
	// 下面两者的使用率达到百分百了
	ProblemMap    map[string]ProblemInfo //题目名->数据
	UserInfoSlice []UserInfo             //所有选手的数据
	// 所有的提交记录，感觉这个字段没有必要缓存
	// 1.占用的内存太大	2.使用率低(管理员+选手自己)，就使用了两次
	// SubmitInfoSlice []SubmitInfo
	// 管理员的信息使用率高，选手的信息使用率低，感觉可以针对性的缓存
	// NewInfoSlice []NewInfo
	// 理想：因为读的情况远远比写的情况要多，而且并行的读操作是互不影响的，
	// 所以理想情况下应该是有一把读锁和一把写锁
	// 读锁锁上的时候只能进行读操作(大量的读)，写锁锁上的时候只能进行写操作(极少的写)
	// 现实：现在只有一把锁，读的时候不能写，但是读的操作不是并行的，这样就会降低效率
	// RToken chan struct{} //读锁的缓存通道
	// 更新一个值需要两个协程，一个发送数据协程，一个接收数据协程，发送数据协程可以不受读写锁的约束
	// SetCh chan dataType //将更新的数据发送到管道中，然后在更新协程中给变量设置新值
	// mu    sync.RWMutex
	// 尝试的改进：创建一个具有缓存的读管道和一个缓存大小为一的写管道
	// 读操作：
	// 尝试进行读操作前，要先获取写令牌，获取不到就阻塞；获取到写令牌后马上释放写令牌，
	// 从而让其他想要进行读操作，但正阻塞在获取写令牌的协程可以得到令牌。然后
	// 获取令牌
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

// type SubmitInfo struct { //有些数据user端不应该解析出来给选手看或者可以多处理一点，直接不传过去
// 	StudentNumber string //唯一，可以关联查找，然后得到StudentName和SchoolName
// 	SubmitTime    string //提交时间
// 	ProblemName   string //题目名称
// 	Language      string //语言
// 	Status        string //状态
// 	RunTime       string //单位ms
// 	RunMemory     string //单位MB
// 	FileSize      string //单位KB
// }

// type NewInfo struct { //有些数据user端不应该解析出来,如其他选手的消息
// 	IsManager  bool   //发送该条消息的人员类型，管理者或者选手,默认是false，也就是默认选手
// 	Identifier string //发送该条消息的人员标识，managerName或者studentNumber
// 	Text       string //发送的文本信息
// 	SendTime   string //发送时间
// }

// 关于数据缓存：
// 因为不想学redis，所以直接使用内存来构造一个缓存

// 呼吸时间：数据是否过期的时间
// 有必要查询时间：最近的一次请求的时间到当前的时间，超过一定值，就没有必要查询了

// 		后端：
// 比赛期间可能会修改比赛名和比赛时间，所以user端定位到一个比赛应该用contestId,而不是contestName
// 我们可以给每场比赛都构建一个结构体，从而通过将数据保存在结构体中来达到缓存数据的目的，
// 我们并不需要每隔一段时间就查询一次数据库，我们可以反过来，
// 向数据库发起查询的情况：
// 1.当前不存在该数据(未创建或者已被删除)
// 2.当前数据过期(当前时间减去最近一次查询的时间得到的时间间隔超过了一次"呼吸的时间")
// 释放缓存的情况：
// 每隔一段时间检查一下最近的一次查询距离当前时间的间隔，大于一个时间段，我们就可以删除它，从而释放内存。
//

//		前端：
// 防抖和节流
// 前端一次请求可以把所有的缓存数据一次性请求完成，也就是每点击一次相关(跟请求缓存数据有关的)按钮，
// 都应该调用请求缓存数据的方法，该方法请求成功，会返回所有的缓存数据，
// 这样就不用设置定时器来定期获取缓存数据了，当然，别忘了防抖和节流
