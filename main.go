package main

import (
	crypto_rand "crypto/rand"
	"encoding/binary"
	"log"
	math_rand "math/rand"
	"os"
	"runtime"
	"time"

	_ "github.com/LQQ4321/owo/client" //别忘了初始化路由
	"github.com/LQQ4321/owo/client/manager"
	"github.com/LQQ4321/owo/config"
	"github.com/LQQ4321/owo/db"
	"github.com/LQQ4321/owo/judger"
	"go.uber.org/zap"
)

var logger *zap.Logger

func main() {
	var err error
	logger, err = zap.NewProduction()
	defer logger.Sync()
	if err != nil {
		log.Fatalln("init logger fail :", err)
	}
	initRand()
	initFiles()
	work := initWorker()
	work.Start()
	judger.JudgerInit(logger.Sugar())
	db.MysqlInit(logger)
	manager.ManagerInit(logger.Sugar())
	time.Sleep(time.Hour * 3)
}

func initWorker() judger.Worker {
	parallelism := runtime.NumCPU()
	return judger.New(judger.Config{
		Parallelism: parallelism,
	})
}

func initRand() {
	var b [8]byte
	_, err := crypto_rand.Read(b[:])
	if err != nil {
		logger.Fatal("random generator init failed :", zap.Error(err))
	}
	sd := int64(binary.LittleEndian.Uint64(b[:]))
	logger.Sugar().Infof("random seed : %d", sd)
	math_rand.Seed(sd)
}

func initFiles() {
	// 在该项目下创造两个文件夹
	// files/allContest用于储存每次比赛的文件，如题目描述文件pdf，测试和样例文件，选手提交代码文件等等
	// files/share_judger用于储存输出文件和选手编译后的可执行文件，和judger共享，方便后端处理judger产生的文件
	if err := os.MkdirAll(config.ALL_CONTEST, 0755); err != nil {
		logger.Fatal(config.ALL_CONTEST+" create fail :", zap.Error(err))
	}
	if err := os.MkdirAll(config.SHARE_JUDGER, 0755); err != nil {
		logger.Fatal(config.SHARE_JUDGER+" create fail :", zap.Error(err))
	}
}
