package main

import (
	"context"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/robfig/cron/v3"
)

var (
	ListenAddr = "0.0.0.0"
	AgentName  = ""
	rdb        = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // redis地址
		Password: "",               // redis没密码，没有设置，则留空
		DB:       0,                // 使用默认数据库
	})

	// all con job manger chan
	pingConJobEndChan = make(chan int, 1)

	// redis context
	ctx     = context.Background()
	JobData []PingJob
	ipMap   = make(map[string]*IPStatus)
	errJob  = PingJob{SPEC: "* * * * * *"}
)

func main() {

	go IcmpListenServer()
	go StartPingConJob(false, &errJob)
	go SubExpiredTLL()
	go StartCronJobs()

	for {
		StartJob()
	}

}

func StartJob() {
	// get data
	JobData, err := RequestJobData()
	if err != nil {
		log.Panicln(err.Error())
		return
	}

	// start job
	go StartPingConJob(true, &JobData)

	// check request data change
	for {
		tmpData, err := RequestJobData()
		if err != nil {
			log.Println(err.Error())
		}
		err = JobCheck(tmpData)

		if err != nil {
			pingConJobEndChan <- 0
			return
		}
		time.Sleep(time.Duration(60) * time.Second)

	}

}

func AgentInit() {
	_, err := rdb.SAdd(ctx, "agents", AgentName).Result()
	PrintErr(err)

	rdb.SetEX(ctx, AgentName+"_online", "ok", time.Duration(60)*time.Second)

}

func PrintErr(err error) {
	if err != nil {
		log.Println(err.Error())
	}
}

type PingJob struct {
	SPEC    string
	IP      []string
	Name    string
	EntryID cron.EntryID
	PTLL    float64
}

type IPStatus struct {
	IP       string
	PTLL     float64
	Count    int64
	ErrList  bool
	Del      bool
	LastTime time.Time
}
