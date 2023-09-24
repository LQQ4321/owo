package user

import (
	"net/http"
	"strconv"

	"github.com/LQQ4321/owo/config"
	"github.com/LQQ4321/owo/db"
	"github.com/gin-gonic/gin"
)

// {"contestId","studentNumber","password"}
func login(info []string, c *gin.Context) {
	var response struct {
		Status      string `json:"status"`
		StudentName string `json:"studentName"`
		SchoolName  string `json:"schoolName"`
		ContestName string `json:"contestName"`
		StartTime   string `json:"startTime"`
		EndTime     string `json:"endTime"`
	}
	response.Status = config.FAIL
	// 看看能不能转成数字，防止sql攻击
	contestId, err := strconv.Atoi(info[0])
	if err != nil {
		logger.Errorln(err)
	} else {
		contest := db.Contests{ID: contestId}
		result := DB.Model(&db.Contests{}).First(&contest)
		if result.Error != nil { //比赛不存在
			logger.Errorln(result.Error)
		} else {
			var user db.Users
			result = DB.Table(db.GetTableName(info[0], config.USER_TABLE_SUFFIX)).
				Where(&db.Users{StudentNumber: info[1], Password: info[2]}).
				First(&user)
			if result.Error != nil { //选手信息不正确，用户不存在或密码不正确
				logger.Errorln(result.Error)
			} else {
				response.ContestName = contest.ContestName
				response.StartTime = contest.StartTime
				response.EndTime = contest.EndTime
				response.StudentName = user.StudentName
				response.SchoolName = user.SchoolName
				response.Status = config.SUCCEED
			}
		}
	}
	c.JSON(http.StatusOK, response)
}

// {"contestId"}
func requestProblemsInfo(info []string, c *gin.Context) {
	var response struct {
		Status   string        `json:"status"`
		Problems []db.Problems `json:"problems"`
	}
	response.Status = config.FAIL
	result := DB.Table(db.GetTableName(info[0], config.PROBLEM_TABLE_SUFFIX)).
		Find(&response.Problems)
	if result.Error != nil {
		logger.Errorln(result.Error)
	} else {
		for i, _ := range response.Problems {
			response.Problems[i].TestFiles = ""
		}
		response.Status = config.SUCCEED
	}
	c.JSON(http.StatusOK, response)
}

// {"contestId"}
func requestUsersInfo(info []string, c *gin.Context) {
	var response struct {
		Status string     `json:"status"`
		Users  []db.Users `json:"users"`
	}
	response.Status = config.FAIL
	result := DB.Table(db.GetTableName(info[0], config.USER_TABLE_SUFFIX)).
		Find(&response.Users)
	if result.Error != nil {
		logger.Errorln(result.Error)
	} else {
		for i, _ := range response.Users {
			response.Users[i].Password = ""
		}
		response.Status = config.SUCCEED
	}
	c.JSON(http.StatusOK, response)
}

// {"contestId","studentNumber"}//因为只是请求自己的提交数据，其实数据量也不是太大
func requestSubmitsInfo(info []string, c *gin.Context) {
	var response struct {
		Status  string       `json:"status"`
		Submits []db.Submits `json:"submits"`
	}
	response.Status = config.FAIL
	result := DB.Table(db.GetTableName(info[0], config.SUBMIT_TABLE_SUFFIX)).
		Where(&db.Submits{StudentNumber: info[1]}).
		Find(&response.Submits)
	if result.Error != nil {
		logger.Error(result.Error)
	} else {
		response.Status = config.SUCCEED
	}
	c.JSON(http.StatusOK, response)
}

// {"contestId","studentNumber"}//因为只是请求自己和管理员的发送消息，其实数据量也不是太大
func requestNewsInfo(info []string, c *gin.Context) {
	var response struct {
		Status string    `json:"status"`
		News   []db.News `json:"news"`
	}
	response.Status = config.FAIL
	result := DB.Table(db.GetTableName(info[0], config.NEW_TABLE_SUFFIX)).
		Where(&db.News{IsManager: false, Identifier: info[1]}).
		Or(&db.News{IsManager: true}).
		Find(&response.News)
	if result.Error != nil {
		logger.Error(result.Error)
	} else {
		response.Status = config.SUCCEED
	}
	c.JSON(http.StatusOK, response)
}

// {"contestId","exampleFilePath"}
func downloadExampleFile(info []string, c *gin.Context) {
	var response struct {
		Status      string `json:"status"`
		ExampleFile string `json:"exampleFile"`
	}
	response.Status = config.FAIL
	// TODO 路径已经知道了，直接将样例文件读取成字符串返回即可
	// (文件不存在最多也就是报错，不会产生什么严重的后果)
}

// {"contestId","problems.id"}//pdf文件不是简单的文本文件，要通过c.File(来获取)

func downloadPdfFile(info []string, c *gin.Context) {

}

// ======= not file up ============= file down =========================

// {"contestId","studentNumber","problems.id","language"}
