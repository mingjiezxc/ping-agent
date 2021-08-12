package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
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

		if nano := BodyUnixNanoCheck(body[4:]); nano == 0 {
			continue
		} else {
			// get ping time
			pingResult := time.Now().UnixNano() - nano

			// ip count + 1
			ipMap[ip].ReceiveCount++

			// ip info last time
			ipMap[ip].UpdateTime = time.Now().Unix()

			// add ping remaek
			ipMap[ip].Ms = append(ipMap[ip].Ms, pingResult)

			// set key and var and set pttl
			rdb.SetEX(ctx, AgentIPLastMsKey+ip, pingResult, time.Duration(ipMap[ip].PTLL)*time.Second)
		}

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

func BodyUnixNanoCheck(body []byte) (t int64) {
	// 1628740059096962098
	data := fmt.Sprintf("%s", body)

	t, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return
	}

	match, _ := regexp.MatchString("[1-9]{16}", data)
	if match {
		return
	}
	return 0
}
