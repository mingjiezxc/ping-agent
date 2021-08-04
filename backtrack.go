package main

import (
	"log"
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
		ipMap[sourceIP.String()].Count++

		// ip info last time
		ipMap[sourceIP.String()].LastTime = time.Now()

		// set key and var and set pttl
		rdb.SetEX(ctx, AgentName+"_"+sourceIP.String(), pingResult, time.Duration(pttl)*time.Second)

	}

}
