package manager

import (
	"errors"
	"net/http"

	"github.com/LQQ4321/owo/config"
	"github.com/LQQ4321/owo/db"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// {"login","root","root"}
// {"addManager","liqiquan","123456"}
// {"deleteManager","liqiquan"}
// {"updatePassword","liqiquan","qwe"}
// {"queryManagers"}
// return [{"liqiquan","123456","3"}]
func managerOperate(info []string, c *gin.Context) {
	var response struct {
		Status string `json:"status"`
		// ErrorInfo string `json:"errorInfo"`//或许可以给每个失败的操作一点出错信息反馈
	}
	response.Status = config.FAIL
	if info[0] == "login" {
		result := DB.Where(&db.Managers{ManagerName: info[1], Password: info[2]}).
			First(&db.Managers{})
		if result.Error != nil {
			if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				logger.Errorln(result.Error)
			}
		} else {
			response.Status = config.SUCCEED
		}
	} else if info[0] == "addManager" {
		var manager db.Managers
		result := DB.Where(&db.Managers{ManagerName: info[1]}).First(&db.Managers{})
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				manager.ManagerName = info[1]
				manager.Password = info[2]
				result = DB.Create(&manager)
				if result.Error != nil {
					logger.Errorln(result.Error)
				} else {
					response.Status = config.SUCCEED
				}
			} else {
				logger.Errorln(result.Error)
			}
		}
	} else if info[0] == "deleteManager" {
		result := DB.Where(&db.Managers{ManagerName: info[1]}).Delete(&db.Managers{})
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				response.Status = config.SUCCEED
			}
		} else {
			response.Status = config.SUCCEED
		}
	} else if info[0] == "updatePassword" {
		result := DB.Where(&db.Managers{ManagerName: info[1]}).
			Updates(&db.Managers{Password: info[2]})
		if result.Error != nil {
			logger.Errorln(result.Error)
		} else {
			response.Status = config.SUCCEED
		}
	}
	c.JSON(http.StatusOK, response)
}
