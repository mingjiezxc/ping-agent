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

	// update redis agent all ip status key
	c.AddFunc("* * * * *", UpdateAgnetAllIpStatus)

	// check err ip list ,if restore remove
	c.AddFunc(config.ErrIPRemoveJobSpec, CheckErrIPRemove)

	// check Meomory ip status add to err ip list
	c.AddFunc(config.MemoryIPStatusCheckSpec, MemoryErrIPCheck)

	// zabbix data send
	c.AddFunc(config.ZabbixSendSpec, SendIpHostDirAndIPStatus)

	c.Start()
	defer c.Stop()
	select {}
}

func CheckErrIPRemove() {

	ips, _ := rdb.SMembers(ctx, AgentErrListKey+AgentName).Result()

	// check err job ip list ,if restore remove ip on err list
	for _, ip := range ips {
		sub1 := time.Now().Unix() - ipMap[ip].UpdateTime
		if (ipMap[ip].SendCount-ipMap[ip].ReceiveCount) <= ipMap[ip].AllowedLoss && ipMap[ip].PTLL > sub1 {
			err := rdb.SRem(ctx, AgentErrListKey+AgentName, ip).Err()
			ipMap[ip].InErrList = false
			CheckPrintErr(err)
		} else {
			ipMap[ip].InErrList = true
		}

	}

}

func MemoryErrIPCheck() {
	// check ip status if now - LastTime > ptll,add to err job
	for _, v := range groupMap {
		for _, ip := range v {

			if _, ok := ipMap[ip]; !ok {
				continue
			}

			sub1 := time.Now().Unix() - ipMap[ip].UpdateTime
			if ipMap[ip].PTLL < sub1 {
				rdb.SAdd(ctx, AgentErrListKey+AgentName, ip).Err()
				if _, ok := ipMap[ip]; ok {
					ipMap[ip].InErrList = true
				}
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
	err := rdb.SAdd(ctx, AgentListKey, AgentName).Err()
	CheckPrintErr(err)

	err = rdb.SetEX(ctx, AgentOnlineKey+AgentName, "online", time.Duration(config.AgentNameOnlineTime)*time.Second).Err()
	CheckPrintErr(err)

}

func UpdateAgnetAllIpStatus() {

	// for ipStr, _ := range ipMap {
	// 	ipMap[ipStr].Agent = AgentName
	// }

	data, err := json.Marshal(ipMap)
	if CheckPrintErr(err) {
		return
	}

	err = rdb.Set(ctx, AgentAllIPStatusKey+AgentName, data, 0).Err()
	CheckPrintErr(err)
}
