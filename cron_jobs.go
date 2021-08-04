package main

import (
	"regexp"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

func StartCronJobs() {
	c := cron.New()

	c.AddFunc("* * * * *", CheckErrIPRemove)
	c.AddFunc("* * * * *", MemoryErrIPCheck)
	c.AddFunc("* * * * *", AgentInit)
	c.Start()
	defer c.Stop()

	select {}
}

func CheckErrIPRemove() {

	// check err job ip list ,if restore remove ip on err list
	for i := 0; i > len(errJob.IP); i++ {
		now := time.Now()
		sub1 := now.Sub(ipMap[errJob.IP[i]].LastTime).Seconds()
		if ipMap[errJob.IP[i]].Count > 3 && ipMap[errJob.IP[i]].PTLL > sub1 {
			errJob.IP = append(errJob.IP[:i], errJob.IP[i+1:]...)
		}
	}

}

func MemoryErrIPCheck() {
	// check ip status if now - LastTime > ptll,add to err job
	for _, v := range JobData {
		for _, ip := range v.IP {
			now := time.Now()
			sub1 := now.Sub(ipMap[ip].LastTime).Seconds()
			if ipMap[ip].PTLL < sub1 {
				errJob.IP = append(errJob.IP, ip)
			}
		}
	}
}

func SubExpiredTLL() {
	pubsub := rdb.Subscribe(ctx, "__keyevent@0__:expired")
	_, err := pubsub.Receive(ctx)
	PrintErr(err)

	// Go channel which receives messages.
	ch := pubsub.Channel()

	// Consume messages.
	for msg := range ch {
		payload := msg.Payload

		match, _ := regexp.MatchString(AgentName+`_((0|[1-9]\d?|1\d\d|2[0-4]\d|25[0-5])\.){3}(0|[1-9]\d?|1\d\d|2[0-4]\d|25[0-5])$`, payload)
		if match {
			s := strings.Split(payload, "_")
			errJob.IP = append(errJob.IP, s[1])
		}
	}

}
