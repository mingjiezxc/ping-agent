package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/robfig/cron/v3"
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

	// all send icmp con job manger chan
	pingConJobEndChan = make(chan int, 1)

	JobData  []PingJob
	ipMap    = make(map[string]*IPStatus)
	groupMap = make(map[string][]string)
)

// redis key
var (
	// pls add AgentName
	AgentListKey        = "agent-list"
	AgentOnlineKey      = "agent-online_"
	AgentErrListKey     = "agent-err-list_"
	AgentJobListKey     = "agent-job-list_"
	AgentAllIPStatusKey = "agent-all-ip-status_"
	// pls add AgentName & ipAddr
	AgentIPLastMsKey = "agent-ip-last-ms_"

	GroupListKey = "group-list"
	// pl add groupName
	GroupNameKey = "group_"

	JobListKey = "job-list"
	// pl add jobName
	JobNameKey = "job_"
)

func init() {

	// read config file
	configfile, err := ioutil.ReadFile("./config.yaml")
	if CheckPrintErr(err) {
		os.Exit(1)
	}

	// yaml marshal config
	err = yaml.Unmarshal(configfile, &config)
	if CheckPrintErr(err) {
		os.Exit(2)
	}

	// create redis client
	rdb = redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	// config agent name
	AgentName = config.AgentName

}

func main() {

	go IcmpListenServer()
	go SubExpiredTLL()
	go StartCronJobs()
	go StartErrListPingJob()

	log.Println("app start")

	for {
		StartPingJobs()
		time.Sleep(3 * time.Second)
	}

}

func StartPingJobs() {
	// get data
	jobs, groupData, err := GetAgentPingJobs()
	JobData = jobs

	if err != nil {
		log.Panicln(err.Error())
		return
	}

	groupMap = groupData
	// update ip PTLL & allowed loss
	UpdateIpMapData()

	// start job
	go StartPingConJob(JobData)

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
			log.Println("Config Change Restart Ping Job")
			return
		}

		// job not change update ip group data
		groupMap = tmpGroupData

		// update ip PTLL & allowed loss
		UpdateIpMapData()

		time.Sleep(time.Duration(60) * time.Second)

	}

}

func UpdateIpMapData() {
	for _, job := range JobData {
		for _, group := range job.Group {
			for _, ip := range groupMap[group] {

				if _, ok := ipMap[ip]; ok {
					ipMap[ip].PTLL = job.PTLL
					ipMap[ip].AllowedLoss = job.AllowedLoss
				} else {
					ipMap[ip] = &IPStatus{
						IP:          ip,
						PTLL:        job.PTLL,
						AllowedLoss: job.AllowedLoss,
						Agent:       AgentName,
					}
				}

			}
		}
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
	Name        string   `json:"name" example:"job1"`
	SPEC        string   `json:"spec" example:"*/10 * * * * *"`
	PTLL        int64    `json:"pttl" example:"3"`
	AllowedLoss int64    `json:"allowedloss" example:"0"`
	Group       []string `json:"group" example:"abc,kkk"`
}

func (g PingJob) Run() {
	for _, groupName := range g.Group {
		for _, ip := range groupMap[groupName] {
			sendPingMsg(ip)
		}

	}

}

type IPStatus struct {
	Agent        string
	IP           string
	PTLL         int64
	AllowedLoss  int64
	SendCount    int64
	ReceiveCount int64
	InErrList    bool
	Ms           []int64
	MsAvg        int64
	Loss         string
	Lost         int64
	UpdateTime   int64
}

type ConfigYaml struct {
	AgentName               string `yaml:"AgentName"`
	AgentNameOnlineTime     int    `yaml:"AgentNameOnlineTime"`
	RedisAddr               string `yaml:"RedisAddr"`
	RedisPassword           string `yaml:"RedisPassword"`
	RedisDB                 int    `yaml:"RedisDB"`
	ErrIPRemoveJobSpec      string `yaml:"ErrIPRemoveJobSpec"`
	MemoryIPStatusCheckSpec string `yaml:"MemoryIPStatusCheckSpec"`
	ZabbixTrapper           bool   `yaml:"ZabbixTrapper"`
	ZabbixServer            string `yaml:"ZabbixServer"`
	ZabbixPort              int    `yaml:"ZabbixPort"`
	ZabbixSendSpec          string `yaml:"ZabbixSendSpec"`
	ErrIPICMPTimeOut        int64  `yaml:"ErrIPICMPTimeOut"`
}

func JobCheck(newJob []PingJob) bool {
	return reflect.DeepEqual(JobData, newJob)
}

func GetAgentPingJobs() (allPingJob []PingJob, tmpGroupMap map[string][]string, err error) {
	// get agent config
	jobs, err := rdb.SMembers(ctx, AgentJobListKey+AgentName).Result()
	if CheckPrintErr(err) {
		return
	}

	// get job config
	for _, jobName := range jobs {

		// get job data
		// data: {SPEC:"* * * * *" , name:"xxxx", group:["xxx", "ccc"]}
		job, err := rdb.Get(ctx, JobNameKey+jobName).Result()
		if CheckPrintErr(err) {
			continue
		}
		var tmpPingJob PingJob
		err = json.Unmarshal([]byte(job), &tmpPingJob)
		if CheckPrintErr(err) {
			continue
		}

		allPingJob = append(allPingJob, tmpPingJob)

	}

	// get all ip group
	groups, err := rdb.SMembers(ctx, GroupListKey).Result()

	tmpGroupMap = make(map[string][]string)

	for _, groupName := range groups {
		ips, err := rdb.SMembers(ctx, GroupNameKey+groupName).Result()
		if CheckPrintErr(err) {
			continue
		}
		tmpGroupMap[groupName] = ips

	}

	return

}

func StartPingConJob(jobs []PingJob) {

	c := cron.New(cron.WithSeconds())
	for _, job := range jobs {
		c.AddJob(job.SPEC, job)
	}

	c.Start()
	defer c.Stop()

	<-pingConJobEndChan

}
