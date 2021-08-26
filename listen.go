package main

import (
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/icmp"
)

type IcmpData struct {
	IP  net.Addr
	msg *icmp.Message
}

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

		IcmpListenChan <- IcmpData{IP: sourceIP, msg: rm}
	}

}

func IcmpListenJob() {
	for data := range IcmpListenChan {
		body, _ := data.msg.Body.Marshal(58)
		ip := data.IP.String()

		if nano := BodyUnixNanoCheck(body[4:]); nano == 0 {
			continue
		} else {
			// check ping data Unix Nano time
			timeNow := time.Now().UnixNano()
			// check val 是否正常
			if timeNow > nano {
				continue
			}
			// 计算 ping 时间值
			pingResult := timeNow - nano

			// 转换单位为 ms, 不足 1ms 计 1ms
			if pingResult < 1000000 {
				pingResult = 1
			} else {
				pingResult = pingResult / 1000000
			}

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

		// check
		if AgentIPLastMsKeyCompile.MatchString(payload) {
			s := strings.Split(payload, "_")
			rdb.SAdd(ctx, AgentErrListKey+AgentName, s[1]).Result()
			if _, ok := ipMap[s[1]]; ok {
				ipMap[s[1]].InErrList = true
			}

		}
	}

}

func BodyUnixNanoCheck(body []byte) (t int64) {
	// 1628740059096962098
	t = BytesToInt64(body)
	match, _ := regexp.MatchString("[0-9]{19}", fmt.Sprintf("%d", t))
	if match {
		return
	}
	return 0
}
