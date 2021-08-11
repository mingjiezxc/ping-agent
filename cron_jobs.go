package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

func StartCronJobs() {
	c := cron.New()

	// update redis agent status and regedit
	c.AddFunc("* * * * *", AgentOnline)

	// check err ip list ,if restore remove
	c.AddFunc(config.ErrIPRemoveJobSpec, CheckErrIPRemove)

	// check Meomory ip status add to err ip list
	c.AddFunc(config.MemoryIPStatusCheckSpec, MemoryErrIPCheck)

	// zabbix data send
	c.AddFunc(config.ZabbixSendSPEC, SendIpHostDirAndIPStatus)

	c.Start()
	defer c.Stop()
	select {}
}

func CheckErrIPRemove() {

	ips, _ := rdb.SMembers(ctx, AgentName+"err_ip").Result()

	// check err job ip list ,if restore remove ip on err list
	for _, ip := range ips {
		sub1 := time.Now().Unix() - ipMap[ip].UpdateTime

		if (ipMap[ip].SendCount-ipMap[ip].ReceiveCount) < config.ErrIPRemoteListAllowedPacketLossData && ipMap[ip].PTLL > sub1 {
			err := rdb.SRem(ctx, AgentName+"err_ip", ip).Err()
			CheckPrintErr(err)
		}

		// clear count
		ipMap[ip].MsAvg = Int64Avg(ipMap[ip].Ms)
		ipMap[ip].Loss = fmt.Sprintf("%.2f", float64(ipMap[ip].ReceiveCount)/float64(ipMap[ip].SendCount))
		ipMap[ip].Lost = ipMap[ip].SendCount - ipMap[ip].ReceiveCount
		ipMap[ip].ReceiveCount = 0
		ipMap[ip].SendCount = 0
		ipMap[ip].Ms = []int64{}

	}

}

func MemoryErrIPCheck() {
	// check ip status if now - LastTime > ptll,add to err job
	for _, v := range groupMap {
		for _, ip := range v {
			sub1 := time.Now().Unix() - ipMap[ip].UpdateTime

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

	err = rdb.SetEX(ctx, "agent-online_"+AgentName, "ok", time.Duration(config.AgentNameOnlineTime)*time.Second).Err()
	CheckPrintErr(err)

}

func UpdateAgnetAllIpStatus() {
	data, err := json.Marshal(ipMap)
	if CheckPrintErr(err) {
		return
	}

	err = rdb.SAdd(ctx, AgentName+"_all-ip-status", data).Err()
	CheckPrintErr(err)
}
