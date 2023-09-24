package client

import (
	"github.com/LQQ4321/owo/client/manager"
	"github.com/LQQ4321/owo/client/user"
	"github.com/LQQ4321/owo/config"
	"github.com/gin-gonic/gin"
)

var (
	Router *gin.Engine
)

func init() {
	Router = gin.Default()
	Router.MaxMultipartMemory = 8 << 20 //这里是8MB

	Router.POST("/managerJson", manager.JsonRequest)
	Router.POST("/managerForm", manager.FormRequest)

	Router.POST("/studentJson", user.JsonRequest)
	Router.POST("/studentForm", user.FormRequest)
	// Router.StaticFile("/downloadProblemsZip", config.FILES_PATH+"Problems.zip")
	go Router.Run(config.URL_PORT)
}
