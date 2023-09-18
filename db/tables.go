package db

import (
	"strconv"
	"time"

	"github.com/LQQ4321/owo/config"
)

// 结构唯一
type Contests struct {
	ID          int    `gorm:primaryKey` //这里应该不算用到gorm(所以不用导入)，可以通过反射得到字符串
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
	ID           int    `gorm:"primaryKey"`
	ProblemName  string //题目名称
	TimeLimit    int64  //运行时间限制,ms
	MemoryLimit  int64  //运行内存限制,MB
	MaxFileLimit int64  //选手提交文件大小限制,KB
	Pdf          bool   //题目是否上传了pdf文件
	SubmitTotal  int64  //提交该题目的人数
	SubmitAc     int64  //通过该题目的人数
	TestFiles    string //题目的测试文件信息，各文件信息使用"|"分隔，例：编号|输入文件路径|输出文件路径|sha256
	ExampleFiles string //题目的样例文件信息，各文件信息使用"|"分隔,例：编号|输入文件路径|输出文件路径
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

func GetTableName(id int, tableSuffix string) string {
	return config.TABLE_PREFIX + strconv.Itoa(id) + tableSuffix
}

// 这里有必要缓存的数据是会经常改变而且有大量请求需求的数据
var CacheMap map[string]ContestInfo = make(map[string]ContestInfo)

// 下面的成员别忘了初始化(所以上面的map的value到底要不要用指针*ContestInfo呀)
type ContestInfo struct {
	// 因为LatestReqTime也涉及读写操作，所以判断"有必要查询时间"的时候要将其锁起来
	// 因为可能要等待获取令牌，所以判断每场比赛的"有必要查询时间"时都可以安排一个协程
	LatestReqTime time.Time //最近一次请求的时间
	ContestId     int       //比赛名就是key,需要id来构建数据表前缀
	// 下面两者的使用率达到百分百了
	ProblemMap    map[string]ProblemInfo //题目名->数据
	UserInfoSlice []UserInfo             //所有选手的数据
	// 所有的提交记录，感觉这个字段没有必要缓存
	// 1.占用的内存太大	2.使用率低(管理员+选手自己)，就使用了两次
	// SubmitInfoSlice []SubmitInfo
	// 管理员的信息使用率高，选手的信息使用率低，感觉可以针对性的缓存
	NewInfoSlice []NewInfo
	Token        chan struct{}
}

type ProblemInfo struct { //测试和样例文件的路径(需要查询很多次，但是很少会改变,后期再优化吧)就不缓存了
	TotalSubmit string
	AcSubmit    string
}

type UserInfo struct {
	StudentNumber string
	StudentName   string
	SchoolName    string
	Status        string //状态，也许可以不用预处理，交给前端来处理
}

type SubmitInfo struct { //有些数据user端不应该解析出来
	StudentNumber string //唯一，可以关联查找，然后得到StudentName和SchoolName
	SubmitTime    string //提交时间
	ProblemName   string //题目名称
	Language      string //语言
	Status        string //状态
	RunTime       string //单位ms
	RunMemory     string //单位MB
	FileSize      string //单位KB
}

type NewInfo struct { //有些数据user端不应该解析出来,如其他选手的消息
	IsManager  bool   //发送该条消息的人员类型，管理者或者选手,默认是false，也就是默认选手
	Identifier string //发送该条消息的人员标识，managerName或者studentNumber
	Text       string //发送的文本信息
	SendTime   string //发送时间
}

// 关于数据缓存：
// 因为不想学redis，所以直接使用内存来构造一个缓存

// 呼吸时间：数据是否过期的时间
// 有必要查询时间：最近的一次请求的时间到当前的时间，超过一定值，就没有必要查询了

// 		后端：
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
