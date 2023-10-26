package config

// 容器间网络配置
const (
	URL_PORT    = ":5051"
	JUDGER_PORT = ":5050"
	MYSQL_PORT  = ":3306"
	REDIS_PORT  = ":6379"
	JUDGER_PATH = "judger"
	MYSQL_PATH  = "mysql"
	REDIS_PATH  = "owo-redis"
	JUDGER_DSN  = "http://" + JUDGER_PATH + JUDGER_PORT + "/run"
	MYSQL_DSN   = "root:3515063609563648226@tcp(" +
		MYSQL_PATH + MYSQL_PORT +
		")/?charset=utf8mb4&parseTime=True&loc=Local"
)

// 文件路径配置
const (
	JUDGER_SHARE_FILE = "/dev/shm/"
	SHARE_JUDGER      = "files/share_judger/"
	ALL_CONTEST       = "files/allContest/"
	USER_SUBMIT_PATH  = "submit" //选手提交的代码在submit目录下
	PDF_FILE_NAME     = "problem.pdf"
	TEST_FILE_NAME    = "test"
	EXAMPLE_FILE_NAME = "example"
)

// 响应前端的状态
const (
	FAIL    = "fail"
	SUCCEED = "succeed"
)

// 数据表配置
const (
	TABLE_PREFIX         = "lqq" //好像MySQL的表名不能使数字开头，所以加上一个前缀
	PROBLEM_TABLE_SUFFIX = "_problems"
	USER_TABLE_SUFFIX    = "_users"
	NEW_TABLE_SUFFIX     = "_news"
	SUBMIT_TABLE_SUFFIX  = "_submits"
)

// 提交状态
const (
	SUBMIT_FAIL    = "InternalError" //提交失败，是后端的问题，不怪选手，不记在一次失败的提交中
	SUBMIT_SUCCEED = "Pending"       //提交成功，等待判题机执行代码
	FIRST_AC       = "FirstAc"       //这道题目第一个AC的选手
	ACCEPTED       = "Accepted"      //成功通过该题
	WRONG_ANSWER   = "WrongAnswer"   //回答错误
	OTHER_ERROR    = "OtherError"    //其他错误的回答
)

// 支持语言
const (
	C_PLUS  = "c++"
	C       = "c"
	GOLANG  = "golang"
	JAVA    = "java"
	PYTHON3 = "python3"
)

// 其他常量
const (
	LAST_ID = "lastId"
)
