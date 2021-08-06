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

	// redis context
	ctx = context.Background()
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // redis地址
		Password: "",               // redis没密码，没有设置，则留空
		DB:       0,                // 使用默认数据库
	})

	// all con job manger chan
	pingConJobEndChan = make(chan int, 1)

	JobData  []PingJob
	ipMap    = make(map[string]*IPStatus)
	groupMap = make(map[string][]string)
	errIP    []string
)

func main() {

	go IcmpListenServer()
	go SubExpiredTLL()
	go StartCronJobs()
	go StartErrListPingJob()

	for {
		StartPingJobs()
	}

}

func StartPingJobs() {
	// get data
	JobData, groupData, err := GetAgentPingJobs()

	groupMap = groupData

	if err != nil {
		log.Panicln(err.Error())
		return
	}

	// start job
	go StartPingConJob(&JobData)

	// check request data change
	for {
		tmpJobData, tmpGroupData, err := GetAgentPingJobs()
		if err != nil {
			log.Println(err.Error())
			continue
		}
		err = JobCheck(tmpJobData)
		if err != nil {
			pingConJobEndChan <- 0
			return
		}
		groupMap = tmpGroupData

		time.Sleep(time.Duration(60) * time.Second)

	}

}

func PrintErr(err error) bool {
	if err != nil {
		log.Println(err.Error())
		return true
	}
	return false
}

type PingJob struct {
	SPEC    string
	Name    string
	Group   []string
	EntryID cron.EntryID
	PTLL    float64
}

func (g PingJob) Run() {
	for _, groupName := range g.Group {
		for _, ip := range groupMap[groupName] {
			sendPingMsg(ip)
		}

	}

}

type IPStatus struct {
	IP           string
	PTLL         float64
	SendCount    int64
	ReceiveCount int64
	ErrList      bool
	Del          bool
	LastTime     time.Time
}
