package user

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/LQQ4321/owo/config"
	"github.com/LQQ4321/owo/db"
	"github.com/LQQ4321/owo/judger"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// {"contestId","studentNumber","password","loginTime"}
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
				user.LoginTime = info[3]
				result := DB.Table(db.GetTableName(info[0], config.USER_TABLE_SUFFIX)).
					Updates(&user)
				if result.Error != nil {
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
	}
	c.JSON(http.StatusOK, response)
}

// {"contestId","studentNumber","text",sendTime}
func sendNews(info []string, c *gin.Context) {
	var response struct {
		Status string `json:"status"`
	}
	response.Status = config.FAIL
	result := DB.Table(db.GetTableName(info[0], config.NEW_TABLE_SUFFIX)).Create(&db.News{
		IsManager:  false,
		Identifier: info[1],
		Text:       info[2],
		SendTime:   info[3],
	})
	if result.Error != nil {
		logger.Errorln(result.Error)
	} else {
		response.Status = config.SUCCEED
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
	response.Problems = make([]db.Problems, 0)
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
// 有一个bug，就是如果选手最后一个小时才登录上去，那么他既不能请求排名数据也没有之前的排名数据，
// 但是正常来说应该是可以得到最后一个小时前的排名数据的。
// 有一个解决办法，就是从submits表里面解析数据，因为users表的status字段里有的数据，submits表里面都有，
// 唯一的缺点就是submits表的数据太大了，而且还需要users表的studentNumber，studentName，schoolName字段
// 才能最终构建出排名。但是显而易见，这个弥补的方法调用的情况还是比较少的，要满足最后一小时登录的条件，
// 而且每名选手只会请求一次，所以消耗资源的情况其实还好
func requestUsersInfo(info []string, c *gin.Context) {
	var response struct {
		Status string     `json:"status"`
		Users  []db.Users `json:"users"`
	}
	response.Status = config.FAIL
	response.Users = make([]db.Users, 0)
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
	response.Submits = make([]db.Submits, 0)
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
	response.News = make([]db.News, 0)
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
	content, err := ioutil.ReadFile(info[0])
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
			newUser.UpdateStatusPre(problemId, config.SUBMIT_SUCCEED, submitTime)
			result = tx.Table(db.GetTableName(contestId, config.USER_TABLE_SUFFIX)).
				Updates(&newUser)
			if result.Error != nil {
				return result.Error
			}
		}
		submitPath = config.ALL_CONTEST + contestId +
			"/" + problemId + "/" + config.USER_SUBMIT_PATH +
			"/" + strconv.Itoa(submit.ID) +
			"." + strings.Split(file.Filename, ".")[1] //这里前端校验的时候多检查一点
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
	testList := strings.Split(problem.TestFiles, "#")
	inputPaths := make([]string, 0)
	hashValues := make([]string, 0)
	for _, v := range testList {
		li := strings.Split(v, "|")
		// 这里的输入文件是按顺序传递过去的，judger好像也是按顺序传递回来的
		inputPaths = append(inputPaths, config.JUDGER_SHARE_FILE+li[1])
		hashValues = append(hashValues, li[3])
	}
	req := &judger.Request{
		Time:           problem.TimeLimit,
		Memory:         problem.MemoryLimit,
		Language:       language,
		SourceFilePath: config.JUDGER_SHARE_FILE + submitPath,
		InputPath:      inputPaths,
		Optional:       3,
	}
	if language == config.PYTHON3 {
		req.Optional = 2
	}
	ch := worker.Submit(c.Request.Context(), req)
	// 阻塞，等待judger回传结果
	rt := <-ch
	// 出现内部错误
	if rt.Error != nil {
		logger.Errorln(rt.Error)
		setInternalError(&oldUser, &submit, contestId, problemId)
		return
	}
	parseRes := dealJudgerResult(rt.Results, hashValues)
	submit.Status = parseRes[0]
	submit.RunTime = parseRes[1]
	submit.RunMemory = parseRes[2]
	result = DB.Table(db.GetTableName(contestId, config.SUBMIT_TABLE_SUFFIX)).
		Updates(&submit)
	if result.Error != nil {
		logger.Errorln(result.Error)
	}
	if parseRes[0] == config.SUBMIT_FAIL {
		setInternalError(&oldUser, &submit, contestId, problemId)
		return
	}
	problem.SubmitTotal++
	if parseRes[0] == config.ACCEPTED {
		parseRes[0] = config.FIRST_AC
		problem.SubmitAc++
	}
	result = DB.Table(db.GetTableName(contestId, config.PROBLEM_TABLE_SUFFIX)).
		Updates(&problem)
	if result.Error != nil {
		logger.Errorln(result.Error)
	}
	if !newUser.IsAccepted(problemId) {
		newUser.UpdateStatusSuf(problemId, parseRes[0])
		result = DB.Table(db.GetTableName(contestId, config.USER_TABLE_SUFFIX)).
			Updates(&newUser)
		if result.Error != nil {
			logger.Errorln(result.Error)
		}
	}
}

// 处理judger返回的数据
func dealJudgerResult(r []judger.Result, hashValues []string) []string {
	// 使用平均值
	var averageTime, averageMemory int64
	for _, v := range r {
		if v.Status != config.ACCEPTED { //取其中一个错误返回即可
			return []string{v.Status, "0", "0"}
		}
		averageTime += v.Time
		averageMemory += v.Memory

	}
	outputFileIds := make([]string, 0)
	for _, v := range r {
		if v.OutputFileId() == "" { //正常来说状态是Accepted就应该有输出文件id
			logger.Errorln(fmt.Errorf("output file id parse error"))
			return []string{config.SUBMIT_FAIL, "0", "0"} //如果没有输出文件的id，多半是解析那里就发生错误了
		}
		outputFileIds = append(outputFileIds, v.OutputFileId())
	}
	for i, v := range outputFileIds { //应该是按顺序返回对应的文件吧
		h := judger.GenerateHashValue(config.SHARE_JUDGER + v)
		if h == "" {
			logger.Errorln(fmt.Errorf("hash value is null"))
			return []string{config.SUBMIT_FAIL, "0", "0"}
		}
		if h != hashValues[i] {
			return []string{config.WRONG_ANSWER, "0", "0"}
		}
	}
	// 有没有可能subSubmit退出了，然后该协程也会强制退出，只不过是因为该协程在subSubmit退出之前就执行完了？？？
	go func() { //删除比赛选手产生的输出文件(输出结果对应的文件一般比较大，而且用完以后用处不大，可以删除)
		for _, v := range outputFileIds {
			if err := os.Remove(config.SHARE_JUDGER + v); err != nil {
				logger.Errorln(err)
				return
			}
		}
	}()
	averageTime /= int64(len(r))
	averageMemory /= int64(len(r))
	return []string{config.ACCEPTED,
		strconv.FormatInt(averageTime>>20, 10),   //ns -> ms
		strconv.FormatInt(averageMemory>>20, 10)} //byte -> MB
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
