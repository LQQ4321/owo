package manager

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

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
			logger.Errorln(result.Error)
		} else {
			response.Status = config.SUCCEED
		}
	} else if info[0] == "addManager" {
		result := DB.Model(&db.Managers{}).
			Where(&db.Managers{ManagerName: info[1]}).
			First(&db.Managers{})
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				result = DB.Model(&db.Managers{}).
					Create(&db.Managers{ManagerName: info[1], Password: info[2]})
				if result.Error != nil {
					logger.Errorln(result.Error)
				} else {
					response.Status = config.SUCCEED
				}
			} else {
				logger.Errorln(result.Error)
			}
		} else {
			logger.Errorln(fmt.Errorf("manager name : " + info[1] + "really exists"))
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
		logger.Errorln(fmt.Errorf("contest name : " + info[0] + "really exists"))
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
		err := DB.Transaction(func(tx *gorm.DB) error {
			for _, tableSuffix := range tablesSuffix {
				if err := tx.Exec("DROP TABLE IF EXISTS " + tablePrefix + tableSuffix).Error; err != nil {
					return err
				}
			}
			result := tx.Model(&db.Contests{}).
				Where(&db.Contests{ContestName: info[0]}).
				Delete(&db.Contests{})
			if result.Error != nil {
				return result.Error
			}
			contestDir := config.ALL_CONTEST + strconv.Itoa(contest.ID)
			if err := os.RemoveAll(contestDir); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			logger.Errorln(err)
		} else {
			response.Status = config.SUCCEED
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
				err := DB.Transaction(func(tx *gorm.DB) error {
					var problem db.Problems
					problem.ProblemName = info[1]
					result := tx.Table(problemTableName).
						Create(&problem)
					if result.Error != nil {
						return result.Error
					}
					problemDir := config.ALL_CONTEST +
						strconv.Itoa(contest.ID) + "/" +
						strconv.Itoa(problem.ID) + "/" +
						config.USER_SUBMIT_PATH
					if err := os.MkdirAll(problemDir, 0755); err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					logger.Errorln(err)
				} else {
					response.Status = config.SUCCEED
				}
			} else {
				logger.Errorln(result.Error)
			}
		} else {
			logger.Error(fmt.Errorf("problem name : " + info[1] + " really exists"))
		}
	}
	c.JSON(http.StatusOK, response)
}

// {"广西大学第一届校赛","两数之和"}
func deleteAProblem(info []string, c *gin.Context) {
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
		var problem db.Problems
		db.TableId = contest.ID //试一下用Model，而不是Table
		result = DB.Model(&db.Problems{}).
			Where(&db.Problems{ProblemName: info[1]}).
			First(&problem)
		if result.Error != nil {
			logger.Errorln(result.Error)
		} else {
			err := DB.Transaction(func(tx *gorm.DB) error {
				result := tx.Model(&db.Problems{}).Delete(&problem)
				if result.Error != nil {
					return result.Error
				}
				problemDir := config.ALL_CONTEST +
					strconv.Itoa(contest.ID) + "/" +
					strconv.Itoa(problem.ID)
				if err := os.RemoveAll(problemDir); err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				logger.Errorln(err)
			} else {
				response.Status = config.SUCCEED
			}
		}
	}
	c.JSON(http.StatusOK, response)
}

// {"liqiquan"}
func requestContestList(info []string, c *gin.Context) {
	var response struct {
		Status      string     `json:"status"`
		ContestList [][]string `json:"contestList"`
	}
	response.Status = config.FAIL
	var contests []db.Contests
	// 所以说每次产生的result要不要关闭，毕竟是一个指针，应该不用吧
	if err := DB.Model(&db.Contests{}).
		Where(&db.Contests{CreatorName: info[0]}).
		Find(&contests).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.ContestList = make([][]string, 0)
			response.Status = config.SUCCEED
		} else {
			logger.Errorln(err)
		}
	} else {
		for _, v := range contests {
			// 话说原本response.ContestList原本是nil，不需要make初始化的吗？(经过测试好像不需要)
			response.ContestList = append(response.ContestList,
				[]string{v.ContestName, v.CreatorName, v.CreateTime, v.StartTime, v.EndTime})
		}
		response.Status = config.SUCCEED
	}
	c.JSON(http.StatusOK, response)
}

// {"广西大学第一届校赛"}
func requestProbelmList(info []string, c *gin.Context) {
	var response struct {
		Status      string     `json:"status"`
		ProblemList [][]string `json:"problemList"`
	}
	response.Status = config.FAIL
	var contest db.Contests
	if err := DB.Model(&db.Contests{}).
		Where(&db.Contests{ContestName: info[0]}).
		First(&contest); err != nil {
		logger.Errorln(err)
	} else {
		db.TableId = contest.ID
		var problems []db.Problems
		if err := DB.Model(&db.Problems{}).Find(&problems); err != nil {
			logger.Errorln(err)
		} else {
			for _, v := range problems {
				pdfFile := "false"
				ioFiles := "false"
				if v.Pdf {
					pdfFile = "true"
				}
				if v.TestFiles != "" {
					ioFiles = "true"
				}
				response.ProblemList = append(response.ProblemList,
					[]string{v.ProblemName,
						strconv.FormatInt(v.TimeLimit, 10),
						strconv.FormatInt(v.MemoryLimit, 10),
						strconv.FormatInt(v.MaxFileLimit, 10),
						pdfFile, ioFiles})
			}
			response.Status = config.SUCCEED
		}
	}
	c.JSON(http.StatusOK, response)
}

// 测试没有赋值的成员，返回到前端后的值
func requestTestNil(info []string, c *gin.Context) {
	/*
			{
		    "int": 0,
		    "string": "",
		    "list": null,
		    "lists": null,
		    "list1": [],
		    "lists1": [],
		    "list2": [
		        "",
		        ""
		    ],
		    "lists2": [
		        null,
		        null
		    ]
		}
	*/
	var response struct {
		Int    int        `json:"int"`
		String string     `json:"string"`
		List   []string   `json:"list"`
		Lists  [][]string `json:"lists"`
		List1  []string   `json:"list1"`
		Lists1 [][]string `json:"lists1"`
		List2  []string   `json:"list2"`
		Lists2 [][]string `json:"lists2"`
	}
	response.List1 = make([]string, 0)
	response.Lists1 = make([][]string, 0)
	response.List2 = make([]string, 2)
	response.Lists2 = make([][]string, 2)
	c.JSON(http.StatusOK, response)
}

// ======= not file up ============= file down =========================

// {"addUsersFromFile","contestName","file"}
func addUsersFromFile(c *gin.Context) {
	var response struct {
		Status string `json:"status"`
	}
	response.Status = config.FAIL
	file, err := c.FormFile("file")
	contestName := c.Request.FormValue("contestName")
	if err != nil {
		logger.Errorln(err)
	} else if contestName == "" {
		logger.Errorln(fmt.Errorf("parse contest name fail"))
	} else {
		fileReader, err := file.Open()
		defer func() {
			err := fileReader.Close()
			if err != nil {
				logger.Errorln(err)
			}
		}()
		if err != nil {
			logger.Errorln(err)
		} else {
			var contest db.Contests
			if result := DB.Model(&db.Contests{}).
				Where(&db.Contests{ContestName: contestName}).
				First(&contest); result.Error != nil {
				logger.Errorln(result.Error)
			} else {
				db.TableId = contest.ID
				err := DB.Transaction(func(tx *gorm.DB) error {
					deleteSql := "TRUNCATE TABLE " + db.Users{}.TableName()
					if result := tx.Exec(deleteSql); result.Error != nil {
						return result.Error
					}
					return nil
				})
				if err != nil {
					logger.Errorln(err)
				} else {
					scanner := bufio.NewScanner(fileReader)
					users := make([]db.Users, 0)
					for scanner.Scan() {
						line := scanner.Text()
						userInfo := strings.Fields(line)
						flag := false
						// studentNumber,studentName,schoolName,password
						for _, v := range users {
							if userInfo[0] == v.StudentNumber {
								flag = true
								break
							}
						}
						if flag {
							continue
						}
						users = append(users, db.Users{
							StudentNumber: userInfo[0],
							StudentName:   userInfo[1],
							SchoolName:    userInfo[2],
							Password:      userInfo[3],
						})
					}
					err := DB.Transaction(func(tx *gorm.DB) error {
						if result := tx.Model(&db.Users{}).
							Create(&users); result.Error != nil {
							return result.Error
						}
						return nil
					})
					if err != nil {
						logger.Errorln(err)
					} else {
						response.Status = config.SUCCEED
					}
				}
			}
		}
	}
	c.JSON(http.StatusOK, response)
}
