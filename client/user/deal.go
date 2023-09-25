package user

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/LQQ4321/owo/config"
	"github.com/LQQ4321/owo/db"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

// {"exampleFilePath = contests.id/problems.id/xxx"}//路径已经在前端完成了，不需要再查询数据库
func downloadExampleFile(info []string, c *gin.Context) {
	var response struct {
		Status      string `json:"status"`
		ExampleFile string `json:"exampleFile"`
	}
	response.Status = config.FAIL
	// 路径已经知道了，直接将样例文件读取成字符串返回即可(文件不存在最多也就是报错，不会产生什么严重的后果)
	exampleFilePath := config.ALL_CONTEST + info[0]
	content, err := ioutil.ReadFile(exampleFilePath)
	if err != nil {
		logger.Errorln(err)
	} else {
		response.ExampleFile = string(content)
		response.Status = config.SUCCEED
	}
	c.JSON(http.StatusOK, response)
}

// {"problemFilePath = contests.id/problems.id/problem.pdf"}
// pdf文件不是简单的文本文件，要通过c.File(来获取)
func downloadPdfFile(info []string, c *gin.Context) {
	problemFilePath := config.ALL_CONTEST + info[0]
	c.File(problemFilePath)
}

// ======= not file up ============= file down =========================

// {"contestId","problemId","studentNumber","language","submitTime","file"}
// 返回的状态：提交失败，提交成功
func submitCode(c *gin.Context) {
	var response struct {
		// 因为正式比赛的时候提交的请求非常多，前端不太可能一直等待judger返回运行的结果，
		// 所以选手提交代码后，我们只需要返回是否提交成功即可，后续运行的状态可以通过查询提交来获取。
		Status string `json:"status"`
	}
	response.Status = config.FAIL
	contestId := c.Request.FormValue("contestId")
	problemId := c.Request.FormValue("problemId")
	studentNumber := c.Request.FormValue("studentNumber")
	language := c.Request.FormValue("language")
	submitTime := c.Request.FormValue("submitTime")
	if contestId == "" || problemId == "" || studentNumber == "" ||
		language == "" || submitTime == "" {
		logger.Errorln("c.Request.FormValue fail")
		c.JSON(http.StatusOK, response)
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		logger.Errorln(err)
		c.JSON(http.StatusOK, response)
		return
	}
	var newUser db.Users
	// 如果后续发生内部错误，应该回滚到提交之前的状态
	var oldUser db.Users
	result := DB.Table(db.GetTableName(contestId, config.USER_TABLE_SUFFIX)).
		Where(&db.Users{StudentNumber: studentNumber}).
		First(&newUser)
	oldUser = newUser
	if result.Error != nil {
		logger.Error(result.Error)
		c.JSON(http.StatusOK, response)
		return
	}
	submit := db.Submits{
		StudentNumber: studentNumber,
		SubmitTime:    submitTime,
		ProblemId:     problemId,
		Language:      language,
		Status:        config.SUBMIT_SUCCEED,
		FileSize:      db.UnitConversion(file.Size),
	}
	var submitPath string
	err = DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Table(db.GetTableName(contestId, config.SUBMIT_TABLE_SUFFIX)).
			Create(&submit)
		if result.Error != nil {
			return result.Error
		}
		// 倘若已经ac，可以在submits表中记录提交，但是不需要在users表中记录
		if !newUser.IsAccepted(problemId) {
			newUser.UpdateStatus(problemId, config.SUBMIT_SUCCEED, submitTime)
			result = tx.Table(db.GetTableName(contestId, config.USER_TABLE_SUFFIX)).
				Updates(&newUser)
			if result.Error != nil {
				return result.Error
			}
		}
		submitPath = config.ALL_CONTEST + contestId +
			"/" + problemId + "/" + strconv.Itoa(submit.ID) +
			"/" + strings.Split(file.Filename, ".")[1]
		if err := c.SaveUploadedFile(file, submitPath); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logger.Errorln(err)
	} else {
		response.Status = config.SUCCEED
	}
	// 因为提交的代码不一定马上能够得到运行，所以这里先返回状态
	c.JSON(http.StatusOK, response)
	problemIdNum, err := strconv.Atoi(problemId)
	if err != nil {
		logger.Errorln(err)
		setInternalError(&oldUser, &submit, contestId, problemId)
		return
	}
	problem := db.Problems{
		ID: problemIdNum,
	}
	result = DB.Table(db.GetTableName(contestId, config.PROBLEM_TABLE_SUFFIX)).
		First(&problem)
	if result.Error != nil {
		logger.Errorln(result.Error)
		setInternalError(&oldUser, &submit, contestId, problemId)
		return
	}
	// TODO 提交到judger

}

// 如果发生了内部错误，那么应该将该次提交的状态设置为内部错误状态
// 修改submits和users表
func setInternalError(oldUser *db.Users, submit *db.Submits, contestId, problemId string) {
	if !oldUser.IsAccepted(problemId) {
		// 回滚到这一次提交之前的状态
		result := DB.Table(db.GetTableName(contestId, config.USER_TABLE_SUFFIX)).
			Updates(oldUser)
		if result.Error != nil {
			logger.Errorln(result.Error)
		}
	}
	submit.Status = config.SUBMIT_FAIL
	result := DB.Table(db.GetTableName(contestId, config.SUBMIT_TABLE_SUFFIX)).
		Updates(submit)
	if result.Error != nil {
		logger.Errorln(result.Error)
	}
}
