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

	formFuncList = make([]formFunc, 0)
}

var (
	jsonRequestList = []string{
		"managerOperate",
		"requestInfoList",
		"createANewContest",
		"deleteAContest",
		"changeInfoNotFile",
		"createANewProblem",
		"deleteAProblem",
		"aboutPermission",
		"downloadFiles",
		"sendNews",
		"requestContestInfoList",
	}
	formRequestList = []string{
		"addUsersFromFile",
		"uploadProblemFiles",
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
	logger.Infoln(request.RequestType)
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
