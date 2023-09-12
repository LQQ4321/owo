package manager

import (
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/LQQ4321/owo/config"
	"github.com/LQQ4321/owo/db"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// {"login","root","root"}
// {"addManager","liqiquan","123456"}
// {"deleteManager","liqiquan"}
// {"updatePassword","liqiquan","qwe"}
// {"queryManagers"} return [{"liqiquan","123456","3"}]
func managerOperate(info []string, c *gin.Context) {
	var response struct {
		Status string `json:"status"`
		// ErrorInfo string `json:"errorInfo"`//或许可以给每个失败的操作一点出错信息反馈
	}
	response.Status = config.FAIL
	if info[0] == "login" {
		result := DB.Model(&db.Managers{}).
			Where(&db.Managers{ManagerName: info[1], Password: info[2]}).
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
		result := DB.Model(&db.Managers{}).
			Where(&db.Managers{ManagerName: info[1]}).
			First(&db.Managers{})
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				manager.ManagerName = info[1]
				manager.Password = info[2]
				result = DB.Model(&db.Managers{}).Create(&manager)
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
		result := DB.Model(&db.Managers{}).
			Where(&db.Managers{ManagerName: info[1]}).
			Delete(&db.Managers{})
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				response.Status = config.SUCCEED
			}
		} else {
			response.Status = config.SUCCEED
		}
	} else if info[0] == "updatePassword" {
		result := DB.Model(&db.Managers{}).
			Where(&db.Managers{ManagerName: info[1]}).
			Updates(&db.Managers{Password: info[2]})
		if result.Error != nil {
			logger.Errorln(result.Error)
		} else {
			response.Status = config.SUCCEED
		}
	} else if info[0] == "queryManagers" {
		var response struct {
			Status      string     `json:"status"`
			ManagerList [][]string `json:"managerList"`
		}
		response.Status = config.FAIL
		var managers []db.Managers
		result := DB.Model(&db.Managers{}).Find(&managers)
		if result.Error != nil {
			logger.Errorln(result.Error)
		} else {
			for _, v := range managers {
				var count int64
				result = DB.Model(&db.Contests{}).
					Where(&db.Contests{CreatorName: v.ManagerName}).
					Count(&count)
				if result.Error != nil {
					logger.Errorln(result.Error)
					c.JSON(http.StatusOK, response)
					return
				}
				list := make([]string, 3)
				list[0] = v.ManagerName
				list[1] = v.Password
				list[2] = strconv.Itoa(int(count))
				response.ManagerList = append(response.ManagerList, list)
			}
			response.Status = config.SUCCEED
		}
		c.JSON(http.StatusOK, response)
		return
	}
	c.JSON(http.StatusOK, response)
}

// {"广西大学第一届校赛","2020-7-25 15:29:10","root"}
func createANewContest(info []string, c *gin.Context) {
	var response struct {
		Status string `json:"status"`
	}
	response.Status = config.FAIL
	result := DB.Where(&db.Contests{ContestName: info[0]}).First(&db.Contests{})
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			var contest db.Contests
			contest.ContestName = info[0]
			contest.CreateTime = info[1]
			contest.StartTime = info[1]
			contest.EndTime = info[1]
			contest.CreatorName = info[2]
			result = DB.Create(&contest)
			if result.Error != nil {
				logger.Errorln(result.Error)
			} else {
				db.TableId = contest.ID
				err := DB.AutoMigrate(&db.Users{}, &db.Problems{}, &db.Submits{}, &db.News{})
				if err != nil {
					logger.Errorln(err)
				} else {
					contestDir := config.ALL_CONTEST + strconv.Itoa(contest.ID)
					err = os.RemoveAll(contestDir)
					if err != nil {
						logger.Errorln(err)
					} else {
						err = os.MkdirAll(contestDir, 0755)
						if err != nil {
							logger.Errorln(err)
						} else {
							response.Status = config.SUCCEED
						}
					}
				}
			}
		} else {
			logger.Errorln(result.Error)
		}
	}
	c.JSON(http.StatusOK, response)
}

// {"广西大学第一届校赛"}
func deleteAContest(info []string, c *gin.Context) {
	var response struct {
		Status string `json:"status"`
	}
	response.Status = config.FAIL
	var contest db.Contests
	result := DB.Model(&db.Contests{}).
		Where(&db.Contests{ContestName: info[0]}).
		First(&contest)
	if result.Error != nil {
		logger.Errorln(result.Error)
	} else {
		// type Table struct {
		// 	TableName string
		// }
		// var tables []Table
		// query := `SELECT table_name FROM information_schema.tables
		// WHERE table_schema = 'online_judge' AND table_name LIKE `
		// query += "'%" + "lqq" + strconv.Itoa(contest.ID) + "%'"
		// if err := DB.Raw(query).Scan(&tables).Error; err != nil {
		// 	logger.Errorln(err)
		// } else {
		// 	for _, table := range tables {
		// 		dropQuery := fmt.Sprintf("DROP TABLE `%s`", table.TableName)
		// 		if err := DB.Exec(dropQuery).Error; err != nil {
		// 			logger.Errorln(err)
		// 			c.JSON(http.StatusOK, response)
		// 			return
		// 		}
		// 	}
		// 	result = DB.Model(&db.Contests{}).
		// 		Where(&db.Contests{ContestName: info[0]}).
		// 		Delete(&db.Contests{})
		// 	if result.Error != nil {
		// 		logger.Errorln(result.Error)
		// 	} else {
		// 		response.Status = config.SUCCEED
		// 	}
		// }
	}
	c.JSON(http.StatusOK, response)
}
