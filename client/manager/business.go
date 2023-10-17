package manager

import (
	"net/http"

	"github.com/LQQ4321/owo/db"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	DB     *gorm.DB
	logger *zap.SugaredLogger
)

var jsonFuncMap map[string]jsonFunc
var formFuncMap map[string]formFunc

func ManagerInit(loggerInstance *zap.SugaredLogger) {
	DB = db.DB
	logger = loggerInstance

	jsonFuncMap = make(map[string]jsonFunc)
	jsonFuncMap = map[string]jsonFunc{
		"managerOperate":          managerOperate,
		"createANewContest":       createANewContest,
		"deleteAContest":          deleteAContest,
		"createANewProblem":       createANewProblem,
		"deleteAProblem":          deleteAProblem,
		"requestContestList":      requestContestList,
		"requestProblemList":      requestProblemList,
		"changeContestConfig":     changeContestConfig,
		"changeProblemConfig":     changeProblemConfig,
		"downloadPlayerList":      downloadPlayerList,
		"downloadSubmitCode":      downloadSubmitCode,
		"sendNews":                sendNews,
		"requestUsersInfo":        requestUsersInfo,
		"requestSubmitsInfo":      requestSubmitsInfo,
		"requestNewsInfo":         requestNewsInfo,
		"createRandomContestData": createRandomContestData,
	}

	formFuncMap = make(map[string]formFunc)
	formFuncMap = map[string]formFunc{
		"addUsersFromFile": addUsersFromFile,
		"uploadPdfFile":    uploadPdfFile,
		"uploadIoFiles":    uploadIoFiles,
	}
}

type jsonFunc func([]string, *gin.Context)
type formFunc func(*gin.Context)

func JsonRequest(c *gin.Context) {
	var request struct {
		RequestType string   `json:"requestType"`
		Info        []string `json:"info"`
	}
	if err := c.BindJSON(&request); err != nil {
		logger.Error("parse request data fail :", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if v, ok := jsonFuncMap[request.RequestType]; ok {
		v(request.Info, c)
	} else {
		c.JSON(http.StatusNotFound, nil)
	}
}

// handing requests in FORM format
func FormRequest(c *gin.Context) {
	requestType := c.Request.FormValue("requestType")
	if v, ok := formFuncMap[requestType]; ok {
		v(c)
	} else {
		c.JSON(http.StatusNotFound, nil)
	}
}
