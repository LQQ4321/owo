package db

import (
	"strconv"
	"strings"

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

// 提交到judger之前
func (u *Users) UpdateStatusPre(problemId, status, submitTime string) {
	if u.Status == "" { //特判没有题目提交的情况
		u.Status = strings.Join([]string{problemId, "1", status, submitTime}, "|")
		return
	}
	list := strings.Split(u.Status, "#")
	editId := -1
	submitCount := "1"
	for i, v := range list {
		if strings.Split(v, "|")[0] == problemId {
			editId = i
			count, err := strconv.Atoi(strings.Split(v, "|")[1])
			if err != nil {
				logger.Sugar().Errorln(err)
				count = 1
			} else {
				count++
			}
			submitCount = strconv.Itoa(count)
			break
		}
	}
	newStatus := strings.Join([]string{problemId, submitCount, status, submitTime}, "|")
	if editId == -1 {
		list = append(list, newStatus)
	} else {
		list[editId] = newStatus
	}
	u.Status = strings.Join(list, "#")
}

// judger处理完成，返回之后
func (u *Users) UpdateStatusSuf(problemId, status string) {
	list := strings.Split(u.Status, "#")
	editId := -1
	for i, v := range list {
		if strings.Split(v, "|")[0] == problemId {
			editId = i
			break
		}
	}
	if editId != -1 {
		temp := strings.Split(list[editId], "|")
		temp[2] = status
		list[editId] = strings.Join(temp, "|")
		u.Status = strings.Join(list, "#")
	} else {
		logger.Sugar().Errorln("not found match option")
	}
}

func (u *Users) IsAccepted(problemId string) bool {
	list := strings.Split(u.Status, "#")
	for _, v := range list {
		statusList := strings.Split(v, "|")
		if statusList[0] == problemId &&
			(statusList[2] == config.ACCEPTED || statusList[2] == config.FIRST_AC) {
			return true
		}
	}
	return false
}

type Submits struct {
	ID            int    `gorm:"primaryKey"`
	StudentNumber string //唯一，可以关联查找，然后得到StudentName和SchoolName
	SubmitTime    string //提交时间
	ProblemId     string //题目名称
	Language      string //语言
	Status        string //状态
	RunTime       string //单位ms
	RunMemory     string //单位MB
	FileSize      string //包含单位
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

var (
	units = []string{" byte", " KB"}
)

func UnitConversion(fileSize int64) string {
	for _, v := range units {
		if fileSize >= 1024 {
			fileSize >>= 10
		} else {
			return strconv.FormatInt(fileSize, 10) + v
		}
	}
	return strconv.FormatInt(fileSize, 10) + " MB"
}
