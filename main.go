package main

import (
	"context"
	"io/ioutil"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"gopkg.in/yaml.v2"
)

var (
	config ConfigYaml

	ListenAddr = "0.0.0.0"

	// redis man name
	AgentName string
	// redis context
	ctx = context.Background()
	rdb *redis.Client

	// all con job manger chan
	pingConJobEndChan = make(chan int, 1)

	JobData  []PingJob
	ipMap    = make(map[string]*IPStatus)
	groupMap = make(map[string][]string)
)

func init() {

	AgentName = config.AgentName

	configfile, err := ioutil.ReadFile("./config.yaml")
	log.Panicln(err.Error())

	err = yaml.Unmarshal(configfile, &config)
	log.Panicln(err.Error())

	rdb = redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

}

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

		// job change restart StartPingConJob
		if !JobCheck(tmpJobData) {
			pingConJobEndChan <- 0

			return
		}

		// job not change update ip group data
		groupMap = tmpGroupData

		time.Sleep(time.Duration(60) * time.Second)

	}

}

func CheckPrintErr(err error) bool {
	if err != nil {
		log.Println(err.Error())
		return true
	}
	return false
}

type PingJob struct {
	SPEC  string
	Name  string
	Group []string
	PTLL  float64
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
	InErrList    bool
	Ms           []int64
	Del          bool
	UpdateTime   time.Time
}

type ConfigYaml struct {
	AgentName               string
	RedisAddr               string
	RedisPassword           string
	RedisDB                 int
	ErrIPRemoveJobSpec      string
	MemoryIPStatusCheckSpec string
	ZabbixTrapper           bool
	ZabbixServer            string
	zabbixPort              int
}
