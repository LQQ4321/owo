package db

import (
	"errors"
	"strconv"

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
	// 因为每次重启，原本的缓存消息都会丢失，所以这里应该查询所有的比赛，将比赛缓存加载进来
	var contests []Contests
	result = DB.Model(&Contests{}).Find(&contests)
	if result.Error != nil {
		logger.Fatal("find all contest info fail : ", zap.Error(result.Error))
	} else {
		for _, v := range contests {
			logger.Sugar().Infoln(v.ID)
			contestId := strconv.Itoa(v.ID)
			UpdateCh[contestId] = make(chan struct{}, 1)
			WaitCh[contestId] = make(chan struct{})
		}
	}
	logger.Sugar().Infoln("Database online_judge init succeed !")
	go cacheUpdateLoop()
}

// func cacheLoop() {
// 	// 更新定时器的间隔
// 	updateInterval := time.Minute * 1
// 	updateTicker := time.NewTicker(updateInterval)
// 	defer updateTicker.Stop()
// 	// updateTicker.C是一个chan，返回值的类型使time.Time
// 	// 因为我们用不到返回值，所以不必写出这样：
// 	// for v := range updateTicker.C {}
// 	for range updateTicker.C {
// 		updateFunc() //还是说是 go updateFunc()???
// 	}

// 	// for {
// 	// 	select {
// 	// 	case <-updateTicker.C:
// 	// 		updateFunc() //还是说是 go updateFunc()???
// 	// 	}
// 	// }
// }

// func updateFunc() {
// 	for k, _ := range CacheMap {
// 		go func(contestId string) {
// 			CacheMap[contestId].Token <- struct{}{} //获取令牌
// 			defer func() {
// 				<-CacheMap[contestId].Token
// 			}()

// 		}(k)
// 	}
// }

// 如果你的需求中读操作远远多于写操作，并且希望使用通道来实现读写安全，
// 你可以考虑使用读写锁（sync.RWMutex）和通道的组合来实现。
// 读操作可以直接读取变量的值，写操作则需要通过通道来进行同步。

// 下面是一个示例代码，演示了如何使用读写锁和通道来实现对变量的读写安全，
// 其中读操作直接读取变量的值，写操作通过通道来进行同步：

// 可能产生的后果，一直有读操作，那么写操作可能就一直没有机会执行，
// 也许的解决办法，弄一个具有缓存的管道，没读一次，就向该管道中传递一个值，
// 待到管道满了以后，就不能在进行读操作
// 顺序：
// 1.向缓存管道获取一个令牌
// 2.读锁锁上（可能存在多个并行的读操作：2 ~ 4 步）
// 3.进行读操作
// 4.读锁解锁
// 5.写锁锁上(发生情况：缓存通道满了，新的读操作无法再将读锁锁上,之前存在的所有读锁也都将解锁)
// 6.清空缓存通道
// 7.进行写操作
// 8.写锁解锁
// 新的轮回
// type SafeVariable struct {
// 	mu    sync.RWMutex //读写锁：读锁锁上的时候可以读，写锁锁上的时候不能读也不能写
// 	value int
// 	setCh chan int
// }

// func NewSafeVariable(initialValue int) *SafeVariable {
// 	sv := &SafeVariable{
// 		value: initialValue,
// 		setCh: make(chan int),
// 	}

// 	go sv.updateValue()

// 	return sv
// }

// func (sv *SafeVariable) GetValue() int {
// 	sv.mu.RLock()
// 	defer sv.mu.RUnlock()

// 	return sv.value
// }

// func (sv *SafeVariable) SetValue(newValue int) {
// 	sv.setCh <- newValue
// }

// func (sv *SafeVariable) updateValue() {
// 	for {
// 		value := <-sv.setCh

// 		sv.mu.Lock()
// 		sv.value = value
// 		sv.mu.Unlock()
// 	}
// }

// func main() {
// 	sv := NewSafeVariable(0)

// 	// 读取变量的值
// 	go func() {
// 		for {
// 			value := sv.GetValue()
// 			fmt.Println("Value:", value)
// 			time.Sleep(1 * time.Second)
// 		}
// 	}()

// 	// 修改变量的值
// 	go func() {
// 		for i := 1; i <= 5; i++ {
// 			sv.SetValue(i)
// 			time.Sleep(2 * time.Second)
// 		}
// 	}()

// 	time.Sleep(10 * time.Second)
// }

// 在上面的示例中，我们定义了一个 SafeVariable 结构体，其中包含一个读写锁、一个值变量和一个设置通道 setCh。
// SafeVariable 提供了 GetValue 和 SetValue 方法来读取和修改变量的值。
// 在 updateValue 方法中，我们通过通道 setCh 来接收写操作传递的新值，并在加锁的情况下更新变量的值。

// 在主函数中，我们启动了两个协程，一个用于读取变量的值，并输出到控制台，另一个用于修改变量的值。
// 通过使用读写锁和通道的组合，我们实现了对变量的读写安全，并保证了读操作的并发性能。
