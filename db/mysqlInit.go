package db

import (
	"errors"

	"github.com/LQQ4321/owo/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	logger *zap.Logger
	DB     *gorm.DB
	RDB    *redis.Client
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
	// 需要这么显示使用吗？之前不用好像也行好像不要啊
	// 报错：Error 1046 (3D000): No database selected
	// DB = DB.Exec("USE online_judge")
	// if DB.Error != nil {
	// 	logger.Fatal("unable to use the database online_judge :", zap.Error(err))
	// }
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
	logger.Sugar().Infoln("mysql init succeed !")
	// 下面是redis的连接配置
	RDB = redis.NewClient(&redis.Options{
		Addr:     config.REDIS_PATH + config.REDIS_PORT,
		Password: "",
		DB:       0,
	})
	logger.Sugar().Infoln("redis init succeed !")
}
