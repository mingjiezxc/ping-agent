package main

import "fmt"

// zabbix send host dir and ip status
func SendIpHostDirAndIPStatus() {
	var allIp []AgentIpDir

	for ip, _ := range ipMap {
		allIp = append(allIp, AgentIpDir{IP: ip})
	}

	var metrics []*Metric
	metrics = append(metrics, NewMetric(AgentName, "ip.dir", fmt.Sprintf("%s", allIp)))

	for ip, status := range ipMap {
		metrics = append(metrics, NewMetric(ip, "ip.loss", status.Loss))
		metrics = append(metrics, NewMetric(ip, "ip.lost", fmt.Sprintf("%d", status.Lost)))
		metrics = append(metrics, NewMetric(ip, "ip.ms", fmt.Sprintf("%d", status.MsAvg)))

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
