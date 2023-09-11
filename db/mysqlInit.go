package db

import (
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
	err = DB.AutoMigrate(&Contests{})
	if err != nil {
		logger.Fatal("create Contests table fail : ", zap.Error(err))
	}
	err = DB.AutoMigrate(&Managers{})
	if err != nil {
		logger.Fatal("create Managers table fail : ", zap.Error(err))
	}
	logger.Sugar().Infoln("Database online_judge init succeed !")
}
