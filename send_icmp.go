package main

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"net"
	"reflect"
	"time"

	"github.com/robfig/cron/v3"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

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
	// get agent all job name
	// data: ["xxx", "kkk", "ccc"]
	jobs, err := rdb.SMembers(ctx, "agent-jobs_"+AgentName).Result()
	if CheckPrintErr(err) {
		return
	}

	// get job config
	for _, jobName := range jobs {

		// get job data
		// data: {SPEC:"* * * * *" , name:"xxxx", group:["xxx", "ccc"]}
		job, err := rdb.Get(ctx, "agent_"+AgentName+"_"+jobName).Result()
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
	groups, err := rdb.SMembers(ctx, "groups-list").Result()

	tmpGroupMap = make(map[string][]string)

	for _, groupName := range groups {
		ips, err := rdb.SMembers(ctx, "group_"+groupName).Result()
		if CheckPrintErr(err) {
			continue
		}
		tmpGroupMap[groupName] = ips

	}

	return

}

func StartPingConJob(v interface{}) {

	c := cron.New(cron.WithSeconds())

	c.Start()
	defer c.Stop()

	<-pingConJobEndChan

}

func Int64ToBytes(i int64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}
func BytesToInt64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}

func CreateICMPData() (wb []byte) {
	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: 11, Seq: 11,
			Data: Int64ToBytes(time.Now().Local().UnixNano()),
		},
	}
	wb, _ = wm.Marshal(nil)
	return

}

func sendPingMsg(addr string) {

	c, _ := net.Dial("ip4:icmp", addr)

	if _, err := c.Write(CreateICMPData()); err != nil {
		log.Println(err.Error())
	}
}

func StartErrListPingJob() {
	c := cron.New(cron.WithSeconds())

	c.AddFunc("* * * * * *", func() {
		ips, _ := rdb.SMembers(ctx, AgentName+"err_ip").Result()
		var metrics []*Metric

		for _, ip := range ips {
			sendPingMsg(ip)
			ipMap[ip].SendCount++
			if time.Now().Unix()-ipMap[ip].UpdateTime > config.ErrIPICMPTimeOut {
				metrics = append(metrics, NewMetric(ip, "ip.ms", "0"))
			}

		}

		// Send packet to zabbix
		packet := NewPacket(metrics)
		z := NewSender(config.ZabbixServer, config.zabbixPort)
		z.Send(packet)

	})

	c.Start()
	defer c.Stop()

	select {}

}
