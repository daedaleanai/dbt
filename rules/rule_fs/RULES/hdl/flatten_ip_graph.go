package hdl

import (
	"crypto/sha256"
	"fmt"
)

func flattenIpGraph(ip Ip, ipMap map[string]bool, ips *[]Ip) {
	for _, dep := range ip.Ips() {
		flattenIpGraph(dep, ipMap, ips)
	}
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%+v", ip)))
	sum := fmt.Sprintf("%x", h.Sum(nil))
	if _, ok := ipMap[sum]; !ok {
		ipMap[sum] = true
		*ips = append(*ips, ip)
	}
}

func FlattenIpGraph(ips []Ip) []Ip {
	ipMap := make(map[string]bool)
	ipsFlat := []Ip{}
	for _, ip := range ips {
		flattenIpGraph(ip, ipMap, &ipsFlat)
	}
	return ipsFlat
}
