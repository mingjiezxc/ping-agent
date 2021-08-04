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
					o.IP = n.IP
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

func RequestJobData() (allData []PingJob, err error) {
	i := 0
	for {
		val, err := rdb.SMembers(ctx, AgentName+"_job_list").Result()

		// err 5s request
		if err != nil {
			log.Println(err.Error())
			time.Sleep(time.Duration(5) * time.Second)
			i++
			continue
		}

		if i > 10 {
			err = errors.New("time out")
			return allData, err
		}

		for _, v := range val {
			pingJobJsonStr, _ := rdb.Get(ctx, v).Result()
			var tmpData []PingJob
			err = json.Unmarshal([]byte(pingJobJsonStr), &tmpData)
			PrintErr(err)

			for _, c := range tmpData {
				for _, ip := range c.IP {
					ipMap[ip] = &IPStatus{
						IP:   ip,
						PTLL: c.PTLL,
					}

				}

			}

			allData = append(allData, tmpData...)
		}

	}

}

func StartPingConJob(endChan bool, v interface{}) {

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

	if endChan {
		<-pingConJobEndChan
	}

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

func (g PingJob) Run() {
	for _, ip := range g.IP {
		sendPingMsg(ip)
	}

}
