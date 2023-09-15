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

// {"contestName","problemName","file"}
func uploadPdfFile(c *gin.Context) {
	var response struct {
		Status string `json:"status"`
	}
	response.Status = config.FAIL
	contestName := c.Request.FormValue("contestName")
	problemName := c.Request.FormValue("problemName")
	file, err := c.FormFile("file")
	if err != nil {
		logger.Errorln(err)
	} else {
		var contest db.Contests
		result := DB.Model(&db.Contests{}).
			Where(&db.Contests{ContestName: contestName}).
			First(&contest)
		if result.Error != nil {
			logger.Errorln(result.Error)
		} else {
			db.TableId = contest.ID
			var problem db.Problems
			result := DB.Model(&db.Problems{}).
				Where(&db.Problems{ProblemName: problemName}).
				First(&problem)
			if result.Error != nil {
				logger.Errorln(result.Error)
			} else {
				pdfPath := config.ALL_CONTEST +
					strconv.Itoa(contest.ID) + "/" +
					strconv.Itoa(problem.ID) + "/" +
					config.PDF_FILE_NAME
				if err := c.SaveUploadedFile(file, pdfPath); err != nil {
					logger.Errorln(err)
				}
				if !problem.Pdf {
					err := DB.Transaction(func(tx *gorm.DB) error {
						problem.Pdf = true
						return tx.Model(&db.Problems{}).Save(&problem).Error
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
	}
	c.JSON(http.StatusOK, response)
}

// {"contestName","problemName","file"}
func uploadIoFiles(c *gin.Context) {
	var response struct {
		Status string `json:"status"`
	}
	response.Status = config.FAIL
	contestName := c.Request.FormValue("contestName")
	problemName := c.Request.FormValue("problemName")
	file, err := c.FormFile("file")
	if err != nil {
		logger.Errorln(err)
	} else {
		var contest db.Contests
		result := DB.Model(&db.Contests{}).
			Where(&db.Contests{ContestName: contestName}).
			First(&contest)
		if result.Error != nil {
			logger.Errorln(result.Error)
		} else {
			db.TableId = contest.ID
			var problem db.Problems
			result := DB.Model(&db.Problems{}).
				Where(&db.Problems{ProblemName: problemName}).
				First(&problem)
			if result.Error != nil {
				logger.Errorln(result.Error)
			} else {
				filePath := config.ALL_CONTEST +
					strconv.Itoa(contest.ID) + "/" +
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
								problem.TestFiles = strings.Join(testList, "#")
								problem.ExampleFiles = strings.Join(exampleList, "#")
								err = DB.Transaction(func(tx *gorm.DB) error {
									return tx.Model(&db.Problems{}).Save(&problem).Error
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
