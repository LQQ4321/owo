package config

// 容器间网络配置
const (
	URL_PORT    = ":5051"
	JUDGER_PORT = ":5050"
	MYSQL_PORT  = ":3306"
	JUDGER_PATH = "judger"
	MYSQL_PATH  = "mysql"
	JUDGER_DSN  = "http://" + JUDGER_PATH + JUDGER_PORT + "/run"
	MYSQL_DSN   = "root:3515063609563648226@tcp(" +
		MYSQL_PATH + MYSQL_PORT +
		")/?charset=utf8mb4&parseTime=True&loc=Local"
)

// 文件路径配置
const (
	SHARE_JUDGER = "files/share_judger/"
	ALL_CONTEST  = "files/allContest/"
)

// 响应前端的状态
const (
	FAIL    = "fail"
	SUCCEED = "succeed"
)
