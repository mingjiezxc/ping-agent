package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"log"
	"net"
	"time"

	"github.com/robfig/cron/v3"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func JobCheck(newJob []PingJob) error {
	// check len
	if len(JobData) != len(newJob) {
		return errors.New("length inconsistent")
	}

	// check name and time, and update ip
	for _, n := range newJob {
		checkStatus := false
		for _, o := range JobData {
			if n.Name == o.Name {
				if n.SPEC == o.SPEC {
					o.Group = n.Group
					checkStatus = true
					break
				}
			}
		}
		if !checkStatus {
			return errors.New("job is not update")
		}
	}
	return nil
}

func GetAgentPingJobs() (allPingJob []PingJob, tmpGroupMap map[string][]string, err error) {
	// get agent all job name
	// data: ["xxx", "kkk", "ccc"]
	jobs, err := rdb.SMembers(ctx, "agent-jobs_"+AgentName).Result()
	if PrintErr(err) {
		return
	}

	// get job config
	for _, jobName := range jobs {

		// get job data
		// data: {SPEC:"* * * * *" , name:"xxxx", group:["xxx", "ccc"]}
		job, err := rdb.Get(ctx, "agent_"+AgentName+"_"+jobName).Result()
		if PrintErr(err) {
			continue
		}
		var tmpPingJob PingJob
		err = json.Unmarshal([]byte(job), &tmpPingJob)
		if PrintErr(err) {
			continue
		}

		allPingJob = append(allPingJob, tmpPingJob)

	}

	// get all ip group
	groups, err := rdb.SMembers(ctx, "groups-list").Result()

	tmpGroupMap = make(map[string][]string)

	for _, groupName := range groups {
		ips, err := rdb.SMembers(ctx, "group_"+groupName).Result()
		if PrintErr(err) {
			continue
		}
		tmpGroupMap[groupName] = ips

	}

	return

}

func StartPingConJob(v interface{}) {

	c := cron.New(cron.WithSeconds())

	switch p := v.(type) {
	case []PingJob:
		for i := 0; i > len(p); i++ {
			p[i].EntryID, _ = c.AddJob(p[i].SPEC, p[i])

		}
	case PingJob:
		p.EntryID, _ = c.AddJob(p.SPEC, p)
	}

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
		for _, ip := range errIP {
			sendPingMsg(ip)
			ipMap[ip].SendCount++
		}
	})

	c.Start()
	defer c.Stop()

	select {}

}
