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
		CheckPrintErr(err)

		rm, err := icmp.ParseMessage(58, rb[:n])
		CheckPrintErr(err)

		body, _ := rm.Body.Marshal(58)

		ip := sourceIP.String()

		pingResult := time.Now().UnixNano() - BytesToInt64(body[4:])

		// ip count + 1
		ipMap[ip].ReceiveCount++

		// ip info last time
		ipMap[ip].UpdateTime = time.Now().Unix()

		// set key and var and set pttl
		rdb.SetEX(ctx, AgentIPLastMsKey+ip, pingResult, time.Duration(ipMap[ip].PTLL)*time.Second)

		ipMap[ip].Ms = append(ipMap[ip].Ms, pingResult)

	}

}

func SubExpiredTLL() {
	// sub time out key
	pubsub := rdb.Subscribe(ctx, "__keyevent@0__:expired")
	_, err := pubsub.Receive(ctx)
	CheckPrintErr(err)

	// Go channel which receives messages.
	ch := pubsub.Channel()

	// Consume messages.
	for msg := range ch {
		payload := msg.Payload

		match, _ := regexp.MatchString(AgentIPLastMsKey+`((0|[1-9]\d?|1\d\d|2[0-4]\d|25[0-5])\.){3}(0|[1-9]\d?|1\d\d|2[0-4]\d|25[0-5])$`, payload)
		if match {
			s := strings.Split(payload, "_")
			rdb.SAdd(ctx, AgentErrListKey, s[1]).Result()
		}
	}

}
