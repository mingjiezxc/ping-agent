package main

import (
	"log"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/icmp"
)

func IcmpListenServer() {
	c, err := icmp.ListenPacket("ip4:icmp", ListenAddr)
	if err != nil {
		log.Println(err.Error())
	}
	defer c.Close()
	rb := make([]byte, 1500)
	for {
		n, sourceIP, err := c.ReadFrom(rb)
		PrintErr(err)

		rm, err := icmp.ParseMessage(58, rb[:n])
		PrintErr(err)

		body, _ := rm.Body.Marshal(58)

		pingResult := time.Now().UnixNano() - BytesToInt64(body[4:])

		// get key pttl
		pttl := ipMap[sourceIP.String()].PTLL

		// ip count + 1
		ipMap[sourceIP.String()].ReceiveCount++

		// ip info last time
		ipMap[sourceIP.String()].LastTime = time.Now()

		// set key and var and set pttl
		rdb.SetEX(ctx, AgentName+"_"+sourceIP.String(), pingResult, time.Duration(pttl)*time.Second)

	}

}

func SubExpiredTLL() {
	// sub time out key
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
			errIP = append(errIP, s[1])
			ipMap[s[1]].ReceiveCount = 0
			ipMap[s[1]].SendCount = 0

		}
	}

}
