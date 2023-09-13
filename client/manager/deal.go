package manager

import (
	"errors"
	"fmt"
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
	} else {
		logger.Errorln(fmt.Errorf("contest name really exists"))
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
		tablesSuffix := []string{config.PROBLEM_TABLE_SUFFIX, config.USER_TABLE_SUFFIX,
			config.SUBMIT_TABLE_SUFFIX, config.NEW_TABLE_SUFFIX}
		tablePrefix := config.TABLE_PREFIX + strconv.Itoa(contest.ID)
		for _, tableSuffix := range tablesSuffix {
			if err := DB.Exec("DROP TABLE IF EXISTS " + tablePrefix + tableSuffix).Error; err != nil {
				logger.Errorln(err)
				c.JSON(http.StatusOK, response)
				return
			}
		}
		result = DB.Model(&db.Contests{}).
			Where(&db.Contests{ContestName: info[0]}).
			Delete(&db.Contests{})
		if result.Error != nil {
			logger.Errorln(result.Error)
		} else {
			contestDir := config.ALL_CONTEST + strconv.Itoa(contest.ID)
			if err := os.RemoveAll(contestDir); err != nil {
				logger.Errorln(err)
			} else {
				response.Status = config.SUCCEED
			}
		}
	}
	c.JSON(http.StatusOK, response)
}

// {"广西大学第一届校赛","两数之和"}
func createANewProblem(info []string, c *gin.Context) {
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
		problemTableName := config.TABLE_PREFIX +
			strconv.Itoa(contest.ID) + config.PROBLEM_TABLE_SUFFIX
		result = DB.Table(problemTableName).
			Where(&db.Problems{ProblemName: info[1]}).
			First(&db.Problems{})
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				var problem db.Problems
				problem.ProblemName = info[1]
				result = DB.Table(problemTableName).
					Create(&problem)
				if result.Error != nil {
					logger.Errorln(result.Error)
				} else {
					problemDir := config.ALL_CONTEST +
						strconv.Itoa(contest.ID) + "/" +
						strconv.Itoa(problem.ID) + "/" +
						config.USER_SUBMIT_PATH
					if err := os.MkdirAll(problemDir, 0755); err != nil {
						logger.Errorln(err)
					} else {
						response.Status = config.SUCCEED
					}
				}
			} else {
				logger.Errorln(result.Error)
			}
		} else {
			logger.Error(fmt.Errorf("probem name really exists"))
		}
	}
	c.JSON(http.StatusOK, response)
}
