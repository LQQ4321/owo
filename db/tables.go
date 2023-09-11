package db

import "strconv"

var (
	TableId int //比赛的数据表对应的编号
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

func (Problems) TableName() string {
	return "lqq" + strconv.Itoa(TableId) + "_problems"
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

func (Users) TableName() string {
	return "lqq" + strconv.Itoa(TableId) + "_users"
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

func (Submits) TableName() string {
	return "lqq" + strconv.Itoa(TableId) + "_submits"
}

type News struct {
	ID         int    `gorm:"primaryKey"`
	IsManager  bool   //发送该条消息的人员类型，管理者或者选手,默认是false，也就是默认选手
	Identifier string //发送该条消息的人员标识，managerName或者studentNumber
	Text       string //发送的文本信息
	SendTime   string //发送时间
}

func (News) TableName() string {
	return "lqq" + strconv.Itoa(TableId) + "_news"
}
