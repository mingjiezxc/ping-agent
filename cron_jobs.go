package main

import (
	"encoding/json"
	"time"

	"github.com/robfig/cron/v3"
)

func StartCronJobs() {
	c := cron.New()

	// check err ip list ,if restore remove
	c.AddFunc(config.ErrIPRemoveJobSpec, CheckErrIPRemove)

	// check Meomory ip status add to err ip list
	c.AddFunc(config.MemoryIPStatusCheckSpec, MemoryErrIPCheck)

	// update redis agent status and regedit
	c.AddFunc("* * * * *", AgentOnline)

	// zabbix data send
	c.AddFunc("*/2 * * * *", SendIpHostDirAndIPStatus)

	c.Start()
	defer c.Stop()
	select {}
}

func CheckErrIPRemove() {

	ips, _ := rdb.SMembers(ctx, AgentName+"err_ip").Result()

	// check err job ip list ,if restore remove ip on err list
	for _, ip := range ips {
		now := time.Now()
		sub1 := now.Sub(ipMap[ip].UpdateTime).Seconds()
		if (ipMap[ip].SendCount-ipMap[ip].ReceiveCount) < 10 && ipMap[ip].PTLL > sub1 {
			rdb.SRem(ctx, AgentName+"err_ip", ip).Result()
		}
	}

}

func MemoryErrIPCheck() {
	// check ip status if now - LastTime > ptll,add to err job
	for _, v := range groupMap {
		for _, ip := range v {
			now := time.Now()
			sub1 := now.Sub(ipMap[ip].UpdateTime).Seconds()
			if ipMap[ip].PTLL < sub1 {

				// check ip on errIP list
				if err := rdb.SIsMember(ctx, AgentName+"err_ip", ip).Err(); err != nil {
					continue
				}

				rdb.SAdd(ctx, AgentName+"err_ip", ip).Result()

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

func AgentOnline() {
	err := rdb.SAdd(ctx, "agent-list", AgentName).Err()
	CheckPrintErr(err)

	rdb.SetEX(ctx, "agent-online_"+AgentName, "ok", time.Duration(120)*time.Second).Result()

}

func UpdateAgnetAllIpStatus() {
	data, err := json.Marshal(ipMap)
	if CheckPrintErr(err) {
		return
	}

	err = rdb.SAdd(ctx, AgentName+"_all-ip-status", data).Err()
	CheckPrintErr(err)
}
