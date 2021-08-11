package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
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
	AgentJobListKey     = "agent-job-list"
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

	AgentName = config.AgentName

	// read config file
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
		time.Sleep(3 * time.Second)
	}

}

func StartPingJobs() {
	// get data
	JobData, groupData, err := GetAgentPingJobs()

	if err != nil {
		log.Panicln(err.Error())
		return
	}

	groupMap = groupData
	// update ip PTLL & allowed loss
	for _, job := range JobData {
		for _, group := range job.Group {
			for _, ip := range groupData[group] {
				ipMap[ip].PTLL = job.PTLL
				ipMap[ip].PTLL = job.AllowedLoss
			}
		}
	}

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

			return
		}

		// job not change update ip group data
		groupMap = tmpGroupData

		// update ip PTLL & allowed loss
		for _, job := range JobData {
			for _, group := range job.Group {
				for _, ip := range groupData[group] {
					ipMap[ip].PTLL = job.PTLL
					ipMap[ip].PTLL = job.AllowedLoss
				}
			}
		}

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
	Name        string
	SPEC        string
	PTLL        int64
	AllowedLoss int64
	Group       []string
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
	AgentName                            string
	AgentNameOnlineTime                  int
	RedisAddr                            string
	RedisPassword                        string
	RedisDB                              int
	ErrIPRemoveJobSpec                   string
	MemoryIPStatusCheckSpec              string
	ZabbixTrapper                        bool
	ZabbixServer                         string
	zabbixPort                           int
	ZabbixSendSPEC                       string
	ErrIPRemoteListAllowedPacketLossData int64
	ErrIPICMPTimeOut                     int64
}

func JobCheck(newJob []PingJob) bool {
	// // check len
	// if len(JobData) != len(newJob) {
	// 	return errors.New("length inconsistent")
	// }

	// // check name and time, and update ip
	// for _, n := range newJob {
	// 	checkStatus := false
	// 	for _, o := range JobData {
	// 		if n.Name == o.Name {
	// 			if n.SPEC == o.SPEC {
	// 				o.Group = n.Group
	// 				checkStatus = true
	// 				break
	// 			}
	// 		}
	// 	}
	// 	if !checkStatus {
	// 		return errors.New("job is not update")
	// 	}
	// }

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
