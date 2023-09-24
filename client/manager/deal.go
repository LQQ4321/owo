package manager

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/LQQ4321/owo/config"
	"github.com/LQQ4321/owo/db"
	"github.com/LQQ4321/owo/judger"
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
			First(&db.Managers{}) //临时创建一个指针变量，将查询到的值放入其中，然后一段时间后被gc回收
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
			// 这张表只有三个字段，所以可以用Updates，如果后续要扩展这个表的时候，
			// 就应该先查找该表，从而获得原本的数据（此次不更新的字段），
			// 然后再更改该变量，从而更新
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
	// 是不是First导致原本的数据消失了
	result := DB.Model(&db.Contests{}).
		Where(&db.Contests{ContestName: info[0]}).
		First(&db.Contests{})
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			contest := db.Contests{
				ContestName: info[0],
				CreateTime:  info[1],
				StartTime:   info[1],
				EndTime:     info[1],
				CreatorName: info[2],
			}
			result = DB.Model(&db.Contests{}).Create(&contest)
			if result.Error != nil {
				logger.Errorln(result.Error)
			} else {
				err := DB.Transaction(func(tx *gorm.DB) error {
					err := tx.Table(db.GetTableName(contest.ID, config.USER_TABLE_SUFFIX)).
						AutoMigrate(&db.Users{})
					if err != nil {
						return err
					}
					err = tx.Table(db.GetTableName(contest.ID, config.PROBLEM_TABLE_SUFFIX)).
						AutoMigrate(&db.Problems{})
					if err != nil {
						return err
					}
					err = tx.Table(db.GetTableName(contest.ID, config.NEW_TABLE_SUFFIX)).
						AutoMigrate(&db.News{})
					if err != nil {
						return err
					}
					err = tx.Table(db.GetTableName(contest.ID, config.SUBMIT_TABLE_SUFFIX)).
						AutoMigrate(&db.Submits{})
					if err != nil {
						return err
					}
					// 创建存放该场比赛文件的文件夹files/allContest/contestId
					contestDir := config.ALL_CONTEST + strconv.Itoa(contest.ID)
					err = os.RemoveAll(contestDir)
					if err != nil {
						return err
					} else {
						err = os.MkdirAll(contestDir, 0755)
						if err != nil {
							return err
						} else {
							//创建该场比赛的缓存键值对
							// contestId := strconv.Itoa(contest.ID)
							// // db.CacheDataMu.Lock()//感觉要等太久了，还是暂时冒一点数据不完整的风险吧
							// db.UpdateCh[contestId] = make(chan struct{}, 1)
							// db.WaitCh[contestId] = make(chan struct{})
							// db.CacheDataMu.Unlock()
						}
					}
					return nil
				})
				if err != nil {
					logger.Errorln(err)
				} else {
					response.Status = config.SUCCEED
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

// {"contestId"}
func deleteAContest(info []string, c *gin.Context) {
	var response struct {
		Status string `json:"status"`
	}
	response.Status = config.FAIL
	tablesSuffix := []string{config.PROBLEM_TABLE_SUFFIX, config.USER_TABLE_SUFFIX,
		config.SUBMIT_TABLE_SUFFIX, config.NEW_TABLE_SUFFIX}
	err := DB.Transaction(func(tx *gorm.DB) error {
		for _, tableSuffix := range tablesSuffix {
			if err := tx.Exec("DROP TABLE IF EXISTS " +
				db.GetTableName(info[0], tableSuffix)).
				Error; err != nil {
				return err
			}
		}
		contestId, err := strconv.Atoi(info[0])
		if err != nil {
			logger.Errorln(err)
			return err
		}
		result := tx.Model(&db.Contests{}).
			Where(&db.Contests{ID: contestId}).
			Delete(&db.Contests{})
		if result.Error != nil {
			return result.Error
		}
		contestDir := config.ALL_CONTEST + info[0]
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
	c.JSON(http.StatusOK, response)
}

// {"contestId","两数之和"}
func createANewProblem(info []string, c *gin.Context) {
	var response struct {
		Status string `json:"status"`
	}
	response.Status = config.FAIL
	result := DB.Table(db.GetTableName(info[0],
		config.PROBLEM_TABLE_SUFFIX)).
		Where(&db.Problems{ProblemName: info[1]}).
		First(&db.Problems{})
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			err := DB.Transaction(func(tx *gorm.DB) error {
				var problem db.Problems
				problem.ProblemName = info[1]
				result := tx.Table(db.GetTableName(info[0],
					config.PROBLEM_TABLE_SUFFIX)).
					Create(&problem)
				if result.Error != nil {
					return result.Error
				}
				// 创建文件夹
				problemDir := config.ALL_CONTEST +
					info[0] + "/" +
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
	c.JSON(http.StatusOK, response)
}

// {"contestId","两数之和"}
func deleteAProblem(info []string, c *gin.Context) {
	var response struct {
		Status string `json:"status"`
	}
	response.Status = config.FAIL
	var problem db.Problems
	result := DB.Table(db.GetTableName(info[0],
		config.PROBLEM_TABLE_SUFFIX)).
		Where(&db.Problems{ProblemName: info[1]}).
		First(&problem)
	if result.Error != nil {
		logger.Errorln(result.Error)
	} else {
		err := DB.Transaction(func(tx *gorm.DB) error {
			result := tx.Table(db.GetTableName(info[0],
				config.PROBLEM_TABLE_SUFFIX)).
				Delete(&problem)
			if result.Error != nil {
				return result.Error
			}
			problemDir := config.ALL_CONTEST +
				info[0] + "/" +
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
				[]string{strconv.Itoa(v.ID), v.ContestName,
					v.CreatorName, v.CreateTime, v.StartTime, v.EndTime})
		}
		response.Status = config.SUCCEED
	}
	c.JSON(http.StatusOK, response)
}

// {"contestId"}
func requestProblemList(info []string, c *gin.Context) {
	var response struct {
		Status      string     `json:"status"`
		ProblemList [][]string `json:"problemList"`
	}
	response.Status = config.FAIL
	var problems []db.Problems
	if err := DB.Table(db.GetTableName(info[0], config.PROBLEM_TABLE_SUFFIX)).
		Find(&problems); err != nil {
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
	c.JSON(http.StatusOK, response)
}

// {"contestId","ICPC-ACM 第五十九届","2023-10-10 11:00:00","2023-10-10 16:00:00"}
func changeContestConfig(info []string, c *gin.Context) {
	var response struct {
		Status string `json:"status"`
	}
	response.Status = config.FAIL
	contestId, err := strconv.Atoi(info[0])
	if err != nil {
		logger.Errorln(err)
	} else {
		result := DB.Model(&db.Contests{}).
			Where(&db.Contests{ID: contestId}).
			Updates(&db.Contests{ContestName: info[1], //有些地方没有字段不包含在此次更新内，不知道这些字段会不会被默认值的覆盖掉
				StartTime: info[2], EndTime: info[3]})
		if result.Error != nil {
			logger.Errorln(result.Error)
		} else {
			response.Status = config.SUCCEED
		}
	}
	c.JSON(http.StatusOK, response)
}

// {"contestId","两数之和","100","128","10"}
func changeProblemConfig(info []string, c *gin.Context) {
	var response struct {
		Status string `json:"status"`
	}
	response.Status = config.FAIL
	timeLimit, err := strconv.ParseInt(info[2], 10, 64)
	if err != nil {
		logger.Errorln(err)
	}
	memoryLimit, err := strconv.ParseInt(info[3], 10, 64)
	if err != nil {
		logger.Errorln(err)
	}
	submitFileLimit, err := strconv.ParseInt(info[4], 10, 64)
	if err != nil {
		logger.Errorln(err)
	}
	result := DB.Table(db.GetTableName(info[0], config.PROBLEM_TABLE_SUFFIX)).
		Where(&db.Problems{ProblemName: info[1]}).
		Updates(&db.Problems{TimeLimit: timeLimit,
			MemoryLimit: memoryLimit, MaxFileLimit: submitFileLimit})
	if result.Error != nil {
		logger.Errorln(result.Error)
	} else {
		response.Status = config.SUCCEED
	}
	c.JSON(http.StatusOK, response)
}

// {"contestId"}
func downloadPlayerList(info []string, c *gin.Context) {
	var response struct {
		Status      string `json:"status"`
		PlayersFile string `json:"playersFile"`
	}
	response.Status = config.FAIL
	var fileContent string
	var users []db.Users
	result := DB.
		Table(db.GetTableName(info[0], config.USER_TABLE_SUFFIX)).
		Find(&users)
	if result.Error != nil {
		logger.Errorln(result.Error)
	} else {
		for _, v := range users {
			list := make([]string, 4)
			list[0] = v.StudentNumber
			list[1] = v.StudentName
			list[2] = v.SchoolName
			list[3] = v.Password
			//特判：内容太长了，超过一行
			fileContent += strings.Join(list, "\t") + "\n"
		}
		response.PlayersFile = fileContent
		response.Status = config.SUCCEED
	}
	c.JSON(http.StatusOK, response)
}

// {"contestId","managerName","text","sendTime"}
func sendNews(info []string, c *gin.Context) {
	var response struct {
		Status string `json:"status"`
	}
	response.Status = config.FAIL
	news := db.News{IsManager: true, Identifier: info[1],
		Text: info[2], SendTime: info[3]}
	result := DB.Table(db.GetTableName(info[0], config.NEW_TABLE_SUFFIX)).
		Create(&news)
	if result.Error != nil {
		logger.Error(result.Error)
	} else {
		response.Status = config.SUCCEED
	}
	c.JSON(http.StatusOK, response)
}

// {"coontestId"}
// func requestProblemsInfo(info []string, c *gin.Context) {
// 	var response struct {
// 		Status   string        `json:"status"`
// 		Problems []db.Problems `json:"problems"`
// 	}
// 	response.Status = config.FAIL
// 	result := DB.Table(db.GetTableName(info[0], config.PROBLEM_TABLE_SUFFIX)).
// 		Find(&response.Problems)
// 	if result.Error != nil {
// 		logger.Errorln(result.Error)
// 	} else {
// 		for i, _ := range response.Problems {
// 			response.Problems[i].TestFiles = ""
// 		}
// 		response.Status = config.SUCCEED
// 	}
// 	c.JSON(http.StatusOK, response)
// }

// {"contestId"}
func requestUsersInfo(info []string, c *gin.Context) {
	var response struct {
		Status string     `json:"status"`
		Users  []db.Users `json:"users"`
	}
	response.Status = config.FAIL
	result := DB.Table(db.GetTableName(info[0], config.USER_TABLE_SUFFIX)).
		Find(&response.Users)
	if result.Error != nil {
		logger.Errorln(result.Error)
	} else {
		response.Status = config.SUCCEED
	}
	c.JSON(http.StatusOK, response)
}

// {"contestId","users.id"}//每次获取的20条数据
func requestSubmitsInfo(info []string, c *gin.Context) {
	var response struct {
		Status  string       `json:"status"`
		Submits []db.Submits `json:"submits"`
	}
	response.Status = config.FAIL
	highId, err := strconv.Atoi(info[1])
	if err != nil {
		logger.Errorln(err)
	} else {
		lowId := highId - 21
		if lowId < 0 {
			lowId = 0
		}
		result := DB.Table(db.GetTableName(info[0], config.SUBMIT_TABLE_SUFFIX)).
			Where("id > ? AND id < ?", lowId, highId).
			Find(&response.Submits)
		if result.Error != nil {
			logger.Errorln(result.Error)
		} else {
			response.Status = config.SUCCEED
		}
	}
	c.JSON(http.StatusOK, response)
}

// {"contestId","news.id"}//每次获取20条数据
func requestNewsInfo(info []string, c *gin.Context) {
	var response struct {
		Status string    `json:"status"`
		News   []db.News `json:"news"`
	}
	response.Status = config.FAIL
	highId, err := strconv.Atoi(info[1])
	if err != nil {
		logger.Errorln(err)
	} else {
		lowId := highId - 21
		if lowId < 0 {
			lowId = 0
		}
		result := DB.Table(db.GetTableName(info[0], config.NEW_TABLE_SUFFIX)).
			Where("id > ? ADN id < ?", lowId, highId).
			Find(&response.News)
		if result.Error != nil {
			logger.Errorln(result.Error)
		} else {
			response.Status = config.SUCCEED
		}
	}
	c.JSON(http.StatusOK, response)
}

// {"contestId"}
// func requestContestCacheData(info []string, c *gin.Context) {
// 	var response struct {
// 		Status      string         `json:"status"`
// 		ContestInfo db.ContestInfo `json:"contestInfo"`
// 	}
// 	response.Status = config.FAIL
// 	db.CacheDataMu.RLock()
// 	// 因为该请求可能在获取到缓存数据后还要处理一些耗时的操作，所以不应该在该函数最后再解锁，
// 	// 可以得到数据后就马上解锁，这样可以增加清理缓存函数执行的机率
// 	// 但是如果该函数意外退出，执行不到解锁那一步的话，清理函数可能就要一直等待，从而导致死锁了
// 	isLock := true
// 	defer func() {
// 		if isLock {
// 			db.CacheDataMu.RUnlock()
// 		}
// 	}()
// 	// 如果不存在该场比赛，直接返回错误
// 	if _, ok := db.WaitCh[info[0]]; !ok {
// 		c.JSON(http.StatusOK, response)
// 		return
// 	}
// 	db.InitContestCache(info[0])
// 	// 判断该场比赛的缓存数据是否初始化了，修改版本：
// 	// 由于db.WaitCh[info[0]]一直没有关闭，所以这里其实是堵塞的
// 	// select {
// 	// // 还没有初始化数据，如果该管道已经关闭，那么应该是优先执行这个case
// 	// case <-db.WaitCh[info[0]]:
// 	// 	// 前面在阻塞,执行这里
// 	// default:
// 	// 	// 如果抢到了令牌，就去更新数据
// 	// 	// 抢不到，就阻塞等别人更新完数据，然后使用
// 	// 	db.InitContestCache(info[0])
// 	// }
// 	// if _, ok := <-db.WaitCh[info[0]]; ok { //当前通道未关闭
// 	// 	// 如果抢到了令牌，就去更新数据
// 	// 	// 抢不到，就阻塞等别人更新完数据，然后使用
// 	// 	db.InitContestCache(info[0])
// 	// }
// 	// 这里有问题，这里的外层if判断的只是WaitCh这个map中是否存在键值对，但是就算该场比赛没有初始化，该键值对也是存在的
// 	// 所以应该再来一个内层if判断到底初始化了没有
// 	// if v, ok := db.WaitCh[info[0]]; ok { //还未初始化该场比赛的缓存数据
// 	// 	// 如果抢到了令牌，就去更新数据
// 	// 	// 抢不到，就阻塞在这里等别人更新完数据，然后使用
// 	// 	if _, ok = <-v; ok {
// 	// 		db.InitContestCache(info[0])
// 	// 	}
// 	// } else {
// 	// 	// 如果选择在这里初始化的话，可能会存在竞态，所以还是在程序启动的时候就初始化好
// 	// 	db.WaitCh[info[0]] = make(chan struct{})
// 	// 	db.UpdateCh[info[0]] = make(chan struct{}, 1)
// 	// 	db.InitContestCache(info[0])
// 	// }
// 	// 总之一句话，加锁到解锁之间的时间要短,最好这期间只有赋值这一步操作
// 	// 更新时间
// 	db.CacheMap[info[0]].TimeMu.Lock()
// 	db.CacheMap[info[0]].LatestReqTime = time.Now()
// 	db.CacheMap[info[0]].TimeMu.Unlock()
// 	// 获取读令牌
// 	// 实际上这里不能将获取读令牌的操作放到db.CacheMap[info[0]].DataMu.RLock()锁区间里面，
// 	// 因为如果我加上了锁，进入锁区间,但是读令牌没有了，那么我就会阻塞在这里，
// 	// 而且db.SetValue方法因为不能把写锁加上，所以也会阻塞，从而导致死锁，
// 	db.CacheMap[info[0]].ReadToken <- struct{}{}
// 	// 更新数据
// 	db.CacheMap[info[0]].DataMu.RLock()
// 	response.ContestInfo = *db.CacheMap[info[0]].ContestInfo
// 	db.CacheMap[info[0]].DataMu.RUnlock()
// 	// 及时解锁，增加清理缓存函数执行的几率
// 	db.CacheDataMu.RUnlock()
// 	isLock = false
// 	if response.ContestInfo.Error == nil {
// 		response.Status = config.SUCCEED
// 	}
// 	c.JSON(http.StatusOK, response)
// 	// // 在这期间无法执行删除操作
// 	// // 好像不能递归地加锁？？？(在加锁和解锁之间再加锁的意思吧)
// 	// db.CacheDataMu.RLock()
// 	// defer db.CacheDataMu.RUnlock()
// 	// if _, ok := db.CacheMap[info[0]]; ok { //该场比赛已经初始化
// 	// } else { //还未初始化
// 	// }
// 	// db.CacheMap[info[0]].DataMu.Lock()
// }

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

// {"contestId","file"}
func addUsersFromFile(c *gin.Context) {
	var response struct {
		Status string `json:"status"`
	}
	response.Status = config.FAIL
	file, err := c.FormFile("file")
	contestId := c.Request.FormValue("contestId")
	if err != nil {
		logger.Errorln(err)
	} else if contestId == "" {
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
				if result := tx.Table(
					db.GetTableName(contestId, config.USER_TABLE_SUFFIX)).
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
	c.JSON(http.StatusOK, response)
}

// {"contestId","problemName","file"}
func uploadPdfFile(c *gin.Context) {
	var response struct {
		Status string `json:"status"`
	}
	response.Status = config.FAIL
	contestId := c.Request.FormValue("contestId")
	problemName := c.Request.FormValue("problemName")
	if contestId == "" || problemName == "" {
		logger.Errorln("parse field fail")
		c.JSON(http.StatusOK, response)
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		logger.Errorln(err)
	} else {
		var problem db.Problems
		result := DB.Table(db.GetTableName(contestId, config.PROBLEM_TABLE_SUFFIX)).
			Where(&db.Problems{ProblemName: problemName}).
			First(&problem)
		if result.Error != nil {
			logger.Errorln(result.Error)
		} else {
			pdfPath := config.ALL_CONTEST +
				contestId + "/" +
				strconv.Itoa(problem.ID) + "/" +
				config.PDF_FILE_NAME
			if err := c.SaveUploadedFile(file, pdfPath); err != nil {
				logger.Errorln(err)
			}
			if !problem.Pdf {
				err := DB.Transaction(func(tx *gorm.DB) error {
					problem.Pdf = true
					return tx.Table(
						db.GetTableName(contestId, config.PROBLEM_TABLE_SUFFIX)).
						Save(&problem).Error
				})
				if err != nil {
					logger.Errorln(err)
				} else {
					response.Status = config.SUCCEED
				}
			} else {
				response.Status = config.SUCCEED
			}
		}
	}
	c.JSON(http.StatusOK, response)
}

// {"contestId","problemName","file"}
func uploadIoFiles(c *gin.Context) {
	var response struct {
		Status string `json:"status"`
	}
	response.Status = config.FAIL
	contestId := c.Request.FormValue("contestId")
	problemName := c.Request.FormValue("problemName")
	file, err := c.FormFile("file")
	if err != nil {
		logger.Errorln(err)
	} else {
		var problem db.Problems
		result := DB.Table(
			db.GetTableName(contestId, config.PROBLEM_TABLE_SUFFIX)).
			Where(&db.Problems{ProblemName: problemName}).
			First(&problem)
		if result.Error != nil {
			logger.Errorln(result.Error)
		} else {
			filePath := config.ALL_CONTEST +
				contestId + "/" +
				strconv.Itoa(problem.ID)
			ioPath := filePath + "/" + file.Filename
			if err := c.SaveUploadedFile(file, ioPath); err != nil {
				logger.Errorln(err)
			} else {
				// 删除除submit和根目录以外的所有文件夹,
				// 为了接下来解压zip文件后，方便寻找解压后的目录名
				dirs := make([]string, 0)
				err := filepath.Walk(filePath,
					func(path string, info fs.FileInfo, err error) error {
						if err != nil {
							return err
						}
						if info.IsDir() {
							if info.Name() != config.USER_SUBMIT_PATH && path != filePath {
								dirs = append(dirs, filepath.Join(filePath, info.Name()))
							}
							if path == filePath {
								return nil
							}
							return filepath.SkipDir //跳出当前目录，不再对当前目录进行递归
						}
						return nil
					})
				if err != nil {
					logger.Errorln(err)
				} else {
					for _, v := range dirs {
						if err := os.RemoveAll(v); err != nil {
							logger.Errorln(err)
							c.JSON(http.StatusOK, response)
							return
						}
					}
					if err := Unzip(ioPath, filePath); err != nil {
						logger.Errorln(err)
					} else {
						// 寻找解压后的目录名,(解压前的zip文件和解压后得到的目录，名称可能不一样)
						// 在windows上新建一个文件夹，取名为A，然后压缩，改名为B
						// 那么在linux上解压前名为B，解压后名为A
						var ioDir string
						err := filepath.Walk(filePath,
							func(path string, info fs.FileInfo, err error) error {
								if err != nil {
									return err
								}
								if info.IsDir() {
									if info.Name() != config.USER_SUBMIT_PATH && path != filePath {
										ioDir = info.Name()
									}
									if path != filePath {
										return filepath.SkipDir
									}
								}
								return nil
							})
						if err != nil {
							logger.Errorln(err)
						} else {
							testDir := filePath + "/" + ioDir + "/" + config.TEST_FILE_NAME + "/"
							exampleDir := filePath + "/" + ioDir + "/" + config.EXAMPLE_FILE_NAME + "/"
							testMap, err := visit(testDir)
							if err != nil {
								logger.Errorln(err)
								c.JSON(http.StatusOK, response)
								return
							}
							testList := make([]string, 0)
							for key, value := range testMap {
								if value[0] == "null" || value[1] == "null" {
									continue
								}
								list := []string{key, testDir + value[0], testDir + value[1]}
								// 将windows格式的txt文件转为linux格式的txt文件
								err = judger.FileConversion(testDir+value[1], filePath+"/"+value[1])
								if err != nil {
									logger.Errorln(err)
									c.JSON(http.StatusOK, response)
									return
								}
								h := judger.GenerateHashValue(filePath + "/" + value[1])
								if h == "" {
									c.JSON(http.StatusOK, response)
									return
								}
								list = append(list, h)
								//id|inid_path|outid_path|hashValue#
								testList = append(testList, strings.Join(list, "|"))
							}
							exampleMap, err := visit(exampleDir)
							if err != nil {
								logger.Errorln(err)
								c.JSON(http.StatusOK, response)
								return
							}
							exampleList := make([]string, 0)
							for key, value := range exampleMap {
								if value[0] == "null" || value[1] == "null" {
									continue
								}
								list := []string{key, exampleDir + value[0], exampleDir + value[1]}
								//id|inid_path|outid_path#
								exampleList = append(exampleList, strings.Join(list, "|"))
							}
							err = cleanTempFile(filePath)
							if err != nil {
								logger.Errorln(err)
								c.JSON(http.StatusOK, response)
								return
							}
							// 下面的Save是全字段更新，所以该方法里面的结构体应该包含原本的数据，
							// 不应该通过新建一个结构体变量的方式来更新数据行
							problem.TestFiles = strings.Join(testList, "#")
							problem.ExampleFiles = strings.Join(exampleList, "#")
							err = DB.Transaction(func(tx *gorm.DB) error {
								return tx.Table(
									db.GetTableName(contestId, config.PROBLEM_TABLE_SUFFIX)).
									Save(&problem).Error
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
		}
	}
	c.JSON(http.StatusOK, response)
}

// 清理文件格式从windows转linux过程中产生的中间out.txt文件
func cleanTempFile(filePath string) error {
	return filepath.Walk(filePath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != filePath { //特判根目录
			return filepath.SkipDir
		}
		if !info.IsDir() && info.Name() != config.PDF_FILE_NAME {
			if err := os.Remove(filepath.Join(filePath, info.Name())); err != nil {
				return err
			}
		}
		return nil
	})
}

// 遍历test和example目录，获取in.txt和out.txt的文件名
func visit(dirPath string) (map[string][]string, error) {
	myMap := make(map[string][]string)
	err := filepath.Walk(dirPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path == dirPath ||
				info.Name() == config.TEST_FILE_NAME ||
				info.Name() == config.EXAMPLE_FILE_NAME {
				return nil
			}
			return filepath.SkipDir
		} else {
			if strings.Contains(info.Name(), ".txt") {
				var start int
				if strings.Contains(info.Name(), "out") {
					start = 3
				} else if strings.Contains(info.Name(), "in") {
					start = 2
				} else {
					return nil
				}
				// rune好像是可以代表汉字的，但是最好还是不要包含汉字了
				var id string
				// out1.txt,id 是 1,"out"到"."之间最好只包含数字编号就好了
				// 后续不应该吧id转为数字类型
				for i, v := range info.Name() {
					if i < start {
						continue
					}
					if v == '.' {
						break
					}
					id += string(v)
				}
				if _, ok := myMap[id]; !ok {
					myMap[id] = []string{"null", "null"}
				}
				myMap[id][start-2] = info.Name()
			}
		}
		return nil
	})
	return myMap, err
}

// 解压缩zip文件到指定路径
func Unzip(zipFilePath, destination string) error {
	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return err
	}
	defer r.Close()
	// 遍历zip的每个文件/文件夹
	for _, file := range r.File {
		// 构建解压缩后的文件路径
		// 我们这里应该是可以将第一个file看作为解压后的根目录的，
		// 那么根目录的名称理论上我们就可以指定为我们想要的，代替掉原本的file.Name即可
		// 后续可以优化一下这里，就不用专门去找解压后的目录名称了
		extractedFilePath := filepath.Join(destination, file.Name)
		if file.FileInfo().IsDir() {
			err = os.MkdirAll(extractedFilePath, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}
		// 创建解压缩后的文件
		extractedFile, err := os.Create(extractedFilePath)
		if err != nil {
			return err
		}
		defer extractedFile.Close()

		// 打开zip中的文件
		zippedFile, err := file.Open()
		if err != nil {
			return err
		}
		defer zippedFile.Close()
		// 将zip文件中的内容复制到解压缩后的文件
		_, err = io.Copy(extractedFile, zippedFile)
		if err != nil {
			return err
		}
	}
	return nil
}
