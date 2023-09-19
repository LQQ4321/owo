package db

import (
	"errors"
	"time"

	"github.com/LQQ4321/owo/config"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	logger *zap.Logger
	DB     *gorm.DB
)

func MysqlInit(loggerInstance *zap.Logger) {
	logger = loggerInstance
	var err error
	DB, err = gorm.Open(mysql.Open(config.MYSQL_DSN), &gorm.Config{})
	if err != nil {
		logger.Fatal("connect database fail :", zap.Error(err))
	}
	err = DB.Exec("CREATE DATABASE IF NOT EXISTS online_judge").Error
	if err != nil {
		logger.Fatal("create database online_judge fail :", zap.Error(err))
	}
	err = DB.Exec("USE online_judge").Error
	if err != nil {
		logger.Fatal("unable to use the database online_judge :", zap.Error(err))
	}
	err = DB.AutoMigrate(&Contests{}, &Managers{})
	if err != nil {
		logger.Fatal("create Contests and Managers table fail : ", zap.Error(err))
	}
	result := DB.Model(&Managers{}).
		Where(&Managers{ManagerName: "root"}).
		First(&Managers{})
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			result = DB.Model(&Managers{}).
				Create(&Managers{ManagerName: "root", Password: "root"})
			if result.Error != nil {
				logger.Fatal("create root role fail : ", zap.Error(result.Error))
			}
		} else {
			logger.Fatal("create root role fail : ", zap.Error(result.Error))
		}
	}
	logger.Sugar().Infoln("Database online_judge init succeed !")
	go cacheLoop()
}

func cacheLoop() {
	// 更新定时器的间隔
	updateInterval := time.Minute * 1
	updateTicker := time.NewTicker(updateInterval)
	defer updateTicker.Stop()
	// updateTicker.C是一个chan，返回值的类型使time.Time
	// 因为我们用不到返回值，所以不必写出这样：
	// for v := range updateTicker.C {}
	for range updateTicker.C {
		updateFunc()
	}

	// for {
	// 	select {
	// 	case <-updateTicker.C:
	// 		updateFunc() //还是说是 go updateFunc()???
	// 	}
	// }
}

func updateFunc() {
	for k, _ := range CacheMap {
		go func(contestId string) {
			CacheMap[contestId].Token <- struct{}{} //获取令牌
			defer func() {
				<-CacheMap[contestId].Token
			}()

		}(k)
	}
}
