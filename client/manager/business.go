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

var jsonFuncList []jsonFunc
var formFuncList []formFunc

func ManagerInit(loggerInstance *zap.SugaredLogger) {
	DB = db.DB
	logger = loggerInstance

	jsonFuncList = make([]jsonFunc, 0)
	jsonFuncList = append(jsonFuncList, managerOperate)
	jsonFuncList = append(jsonFuncList, createANewContest)
	jsonFuncList = append(jsonFuncList, deleteAContest)
	jsonFuncList = append(jsonFuncList, createANewProblem)
	jsonFuncList = append(jsonFuncList, deleteAProblem)
	jsonFuncList = append(jsonFuncList, requestContestList)
	jsonFuncList = append(jsonFuncList, requestProblemList)
	jsonFuncList = append(jsonFuncList, changeContestConfig)
	jsonFuncList = append(jsonFuncList, changeProblemConfig)
	jsonFuncList = append(jsonFuncList, downloadPlayerList)
	jsonFuncList = append(jsonFuncList, sendNews)
	jsonFuncList = append(jsonFuncList, requestContestCacheData)

	formFuncList = make([]formFunc, 0)
	formFuncList = append(formFuncList, addUsersFromFile)
	formFuncList = append(formFuncList, uploadPdfFile)
	formFuncList = append(formFuncList, uploadIoFiles)
}

var (
	jsonRequestList = []string{
		"managerOperate",
		"createANewContest",
		"deleteAContest",
		"createANewProblem",
		"deleteAProblem",
		"requestContestList",
		"requestProblemList",
		"changeContestConfig",
		"changeProblemConfig",
		"downloadPlayerList",
		"sendNews",
		"requestContestCacheData",
	}
	formRequestList = []string{
		"addUsersFromFile",
		"uploadPdfFile",
		"uploadIoFiles",
	}
)

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
	for i, f := range jsonFuncList {
		if jsonRequestList[i] == request.RequestType {
			f(request.Info, c)
			break
		}
	}
}

// handing requests in FORM format
func FormRequest(c *gin.Context) {
	requestType := c.Request.FormValue("requestType")
	for i, f := range formFuncList {
		if formRequestList[i] == requestType {
			f(c)
			break
		}
	}
}
