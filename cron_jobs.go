package main

import (
	"time"

	"github.com/robfig/cron/v3"
)

func StartCronJobs() {
	c := cron.New()

	// check err ip list ,if restore remove
	c.AddFunc("* * * * *", CheckErrIPRemove)

	// check Meomory ip status add to err ip list
	c.AddFunc("* * * * *", MemoryErrIPCheck)

	// update redis agent status and regedit
	c.AddFunc("* * * * *", AgentInit)

	c.Start()
	defer c.Stop()
	select {}
}

func CheckErrIPRemove() {

	// check err job ip list ,if restore remove ip on err list
	for i := 0; i > len(errIP); i++ {
		now := time.Now()
		sub1 := now.Sub(ipMap[errIP[i]].LastTime).Seconds()
		if (ipMap[errIP[i]].SendCount-ipMap[errIP[i]].ReceiveCount) < 10 && ipMap[errIP[i]].PTLL > sub1 {
			errIP = append(errIP[:i], errIP[i+1:]...)
		}
	}

	// clear ip count
	for ip, _ := range ipMap {
		ipMap[ip].ReceiveCount = 0
		ipMap[ip].SendCount = 0
	}

}

func MemoryErrIPCheck() {
	// check ip status if now - LastTime > ptll,add to err job
	for _, v := range groupMap {
		for _, ip := range v {
			now := time.Now()
			sub1 := now.Sub(ipMap[ip].LastTime).Seconds()
			if ipMap[ip].PTLL < sub1 {

				// check ip on errIP list
				if ArryInCheck(ip, errIP) {
					continue
				}

				errIP = append(errIP, ip)
			}
		}
	}

}

func ArryInCheck(val string, arry []string) bool {
	for _, k := range arry {
		if k == val {
			return true
		}
	}
	return false
}

func AgentInit() {
	_, err := rdb.SAdd(ctx, "agent-list", AgentName).Result()
	PrintErr(err)

	rdb.SetEX(ctx, "agent-online_"+AgentName, "ok", time.Duration(120)*time.Second).Result()

}
