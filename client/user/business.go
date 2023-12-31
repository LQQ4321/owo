package user

import (
	"net/http"

	"github.com/LQQ4321/owo/db"
	"github.com/LQQ4321/owo/judger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	DB     *gorm.DB
	logger *zap.SugaredLogger
	worker judger.Worker
)

var jsonFuncMap map[string]jsonFunc
var formFuncMap map[string]formFunc

func UserInit(loggerInstance *zap.SugaredLogger, w judger.Worker) {
	DB = db.DB
	logger = loggerInstance
	worker = w
	// 跟map[string]*jsonFunc应该没有区别吧
	jsonFuncMap = make(map[string]jsonFunc)
	jsonFuncMap = map[string]jsonFunc{
		"login":               login,
		"sendNews":            sendNews,
		"requestProblemsInfo": requestProblemsInfo,
		"requestUsersInfo":    requestUsersInfo,
		"requestSubmitsInfo":  requestSubmitsInfo,
		"requestNewsInfo":     requestNewsInfo,
		"downloadExampleFile": downloadExampleFile,
		"downloadPdfFile":     downloadPdfFile,
	}

	formFuncMap = make(map[string]formFunc)
	formFuncMap = map[string]formFunc{
		"submitCode": submitCode,
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
