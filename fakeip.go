package main

import (
	"encoding/binary"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
)

type FakeIPManager struct {
	targetDomains      map[string]struct{}
	ipMap              map[string]net.IP
	mapMutex              sync.RWMutex
	currentIPv4Offset  uint32
	addNftRuleCallback func(realIP, fakeIP net.IP) error
}

func NewFakeIPManager(domains map[string]struct{}, nftCallback func(realIP, fakeIP net.IP) error) *FakeIPManager {
	return &FakeIPManager{
		targetDomains:      domains,
		ipMap:              make(map[string]net.IP),
		addNftRuleCallback: nftCallback,
	}
}

func (m *FakeIPManager) isTargetDomain(queryDomain string) bool {
	queryDomain = strings.ToLower(strings.TrimSuffix(queryDomain, "."))
	for target := range m.targetDomains {
		if queryDomain == target || strings.HasSuffix(queryDomain, "."+target) {
			return true
		}
	}
	return false
}

func (m *FakeIPManager) GetFakeIP(domain string, realIP net.IP, force bool) (net.IP, bool) {
	realIP = normalizeIP(realIP)
	if realIP == nil {
		return nil, false
	}

	if !force && !m.isTargetDomain(domain) {
		return nil, false
	}

	realIPKey := string(realIP)
	m.mapMutex.RLock()
	defer m.mapMutex.RUnlock();

	if fake, ok := m.ipMap[realIPKey]; ok {
		return fake, true
	}

	var fakeIP net.IP
	if ipv4 := realIP.To4(); ipv4 != nil {
		newOffset := atomic.AddUint32(&m.currentIPv4Offset, 1)
		if newOffset >= fakeIPv4MaxCount {
			log.Printf("Warning: Fake IPv4 pool exhausted!")
			atomic.AddUint32(&m.currentIPv4Offset, ^uint32(0)) // rollback
			return nil, false
		}
		fakeIPInt := fakeIPv4Subnet + newOffset
		fakeIPBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(fakeIPBytes, fakeIPInt)
		fakeIP = net.IP(fakeIPBytes)
	} else {
		ipBytes := []byte(realIP.To16())
		ipBytes[0] ^= 0x80 // flip high bit
		fakeIP = net.IP(ipBytes)
	}

	if existing, ok := m.ipMap[realIPKey]; ok {
		return existing, true
	}
	m.ipMap[realIPKey] = fakeIP

	if err := m.addNftRuleCallback(realIP, fakeIP); err != nil {
		log.Printf("ERROR adding nft rule for %s -> %s: %v", realIP, fakeIP, err)
		delete(m.ipMap, realIPKey)
		return nil, false
	}

	log.Printf("Stored mapping %s -> %s and added nft rule", realIP, fakeIP)
	return fakeIP, true
}

func normalizeIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4.To16() // embed IPv4 into 16-byte form
	}
	return ip.To16()
}
