package main

import "fmt"

func SendIpHostDirAndIPStatus() {
	var allIp []AgentIpDir

	for ip, _ := range ipMap {
		allIp = append(allIp, AgentIpDir{IP: ip})
	}

	var metrics []*Metric
	metrics = append(metrics, NewMetric(AgentName, "ip.dir", fmt.Sprintf("%s", allIp)))

	for ip, status := range ipMap {
		metrics = append(metrics, NewMetric(ip, "ip.loss", fmt.Sprintf("%d", status.ReceiveCount/status.SendCount)))
		metrics = append(metrics, NewMetric(ip, "ip.lost", fmt.Sprintf("%d", status.SendCount-status.ReceiveCount)))
		metrics = append(metrics, NewMetric(ip, "ip.ms", fmt.Sprintf("%d", Int64Avg(status.Ms))))
		ipMap[ip].ReceiveCount = 0
		ipMap[ip].SendCount = 0
	}

	// create packet
	packet := NewPacket(metrics)

	// Send packet to zabbix
	z := NewSender(config.ZabbixServer, config.zabbixPort)
	z.Send(packet)

}

func Int64Avg(i []int64) int64 {

	var intSum int64
	for _, v := range i {
		intSum = intSum + v
	}

	return intSum / int64(len(i))

}

type AgentIpDir struct {
	IP string `json:"{#IP}"`
}
