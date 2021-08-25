package main

import (
	"encoding/binary"
	"log"
	"net"
	"time"

	"github.com/robfig/cron/v3"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func Int64ToBytes(i int64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}
func BytesToInt64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}

func CreateICMPData() (wb []byte) {
	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: 11, Seq: 11,
			Data: Int64ToBytes(time.Now().Local().UnixNano()),
		},
	}
	wb, _ = wm.Marshal(nil)
	return

}

func sendPingMsg(addr string) {

	c, _ := net.Dial("ip4:icmp", addr)

	if _, err := c.Write(CreateICMPData()); err != nil {
		log.Println(err.Error())
	} else {
		if _, ok := ipMap[addr]; ok {
			ipMap[addr].SendCount++
		}
	}
}

func StartErrListPingJob() {
	c := cron.New(cron.WithSeconds())

	c.AddFunc("* * * * * *", func() {
		ips, _ := rdb.SMembers(ctx, AgentErrListKey+AgentName).Result()
		var metrics []*Metric

		for _, ip := range ips {
			sendPingMsg(ip)
			if time.Now().Unix()-ipMap[ip].UpdateTime > config.ErrIPICMPTimeOut {
				metrics = append(metrics, NewMetric(ip, "ip.ms", "0"))
			}

		}

		// Send packet to zabbix
		packet := NewPacket(metrics)
		z := NewSender(config.ZabbixServer, config.ZabbixPort)
		z.Send(packet)

	})

	c.Start()
	defer c.Stop()

	select {}

}
