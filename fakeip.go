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
	ipMap              sync.Map
	currentIPv4Offset  uint32
	addNftRuleCallback func(realIP, fakeIP net.IP) error
	mu                 sync.RWMutex
}

func NewFakeIPManager(domains map[string]struct{}, nftCallback func(realIP, fakeIP net.IP) error) *FakeIPManager {
	return &FakeIPManager{
		targetDomains:      domains,
		addNftRuleCallback: nftCallback,
	}
}

func (m *FakeIPManager) isTargetDomain(queryDomain string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	queryDomain = strings.ToLower(strings.TrimSuffix(queryDomain, "."))
	for target := range m.targetDomains {
		if queryDomain == target || strings.HasSuffix(queryDomain, "."+target) {
			return true
		}
	}
	return false
}

func (m *FakeIPManager) GetFakeIP(domain string, realIP net.IP, force bool) (net.IP, bool) {
	realIPStr := realIP.String()

	// 1. Check if domain should be faked
	if !force || !m.isTargetDomain(domain) {
		return nil, false
	}

	// 2. Check existing mapping
	if fakeIPVal, ok := m.ipMap.Load(realIPStr); ok {
		return fakeIPVal.(net.IP), true
	}

	// 3. Generate new fake IP
	var fakeIP net.IP
	if realIP.To4() != nil {
		newOffset := atomic.AddUint32(&m.currentIPv4Offset, 1)
		if newOffset >= fakeIPv4MaxCount {
			log.Printf("Warning: Fake IPv4 address pool exhausted!")
			atomic.AddUint32(&m.currentIPv4Offset, ^uint32(0)) // Decrement back
			return nil, false
		}

		fakeIPInt := fakeIPv4Subnet + newOffset
		fakeIPBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(fakeIPBytes, fakeIPInt)
		fakeIP = net.IP(fakeIPBytes)
	} else if ipBytes := []byte(realIP.To16()); ipBytes[0] < 0x50 {
		fakeIPBytes := make([]byte, len(ipBytes))
		copy(fakeIPBytes, ipBytes)
		fakeIPBytes[0] += 0x80
		fakeIP = net.IP(fakeIPBytes)
	}

	// 4. Store mapping atomically
	actualFakeIPVal, loaded := m.ipMap.LoadOrStore(realIPStr, fakeIP)
	actualFakeIP := actualFakeIPVal.(net.IP)

	// 5. If this goroutine stored the mapping, add the nft rule
	if !loaded {
		err := m.addNftRuleCallback(realIP, actualFakeIP)
		if err != nil {
			log.Printf("ERROR adding nftables rule for %s -> %s: %v", realIP, actualFakeIP, err)
			m.ipMap.Delete(realIPStr)
			log.Printf("Removed mapping %s -> %s due to nftables error", realIP, actualFakeIP)
			return nil, false
		}
		log.Printf("Stored mapping %s -> %s and processed nft rule (logged on non-Linux)", realIP, actualFakeIP)
		return actualFakeIP, true
	} else {
		log.Printf("Mapping %s -> %s already existed", realIP, actualFakeIP)
		return actualFakeIP, true
	}
}
