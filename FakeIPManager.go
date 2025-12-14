package main

import (
	"encoding/binary"
	"net"
	"sync"
	"sync/atomic"
)

const (
	fakeIPv4Subnet     = uint32(0xC6120000) // 198.18.0.0
	fakeIPMaxCount     = 131070             // up to 198.19.255.254
	fakeIPv6SubnetHigh = uint64(0xabcd_0bad_c0de_0000)
)

type FakeIPManager struct {
	ipv4Map             map[IPv4Addr]net.IP
	ipv6Map             map[IPv6Addr]net.IP
	rwMutex4            sync.RWMutex
	rwMutex6            sync.RWMutex
	addNftRule4Callback func(realIP net.IP, fakeIP net.IP) error
	addNftRule6Callback func(realIP net.IP, fakeIP net.IP) error
	nextIPv4Counter     atomic.Uint32
	nextIPv6Counter     atomic.Uint64
}

func NewFakeIPManager(
	nftCallback4 func(realIP, fakeIP net.IP) error,
	nftCallback6 func(realIP, fakeIP net.IP) error) *FakeIPManager {
	return &FakeIPManager{
		ipv4Map:             make(map[IPv4Addr]net.IP),
		ipv6Map:             make(map[IPv6Addr]net.IP),
		addNftRule4Callback: nftCallback4,
		addNftRule6Callback: nftCallback6,
	}
}

func (m *FakeIPManager) GetFakeIP(domain string, realIP net.IP, ipVersion int) net.IP {
	if realIP == nil {
		return nil
	}
	switch ipVersion {
	case 4:
		return m.GetFakeIPv4(realIP)
	case 6:
		return m.GetFakeIPv6(realIP)
	default:
		return nil
	}
}

func (m *FakeIPManager) GetFakeIPv4(realIP net.IP) net.IP {
	ip := MustIPv4FromIP(realIP)

	m.rwMutex4.RLock()
	fakeIP, ok := m.ipv4Map[ip]
	m.rwMutex4.RUnlock()
	if ok {
		return fakeIP
	}

	m.rwMutex4.Lock()
	defer m.rwMutex4.Unlock()

	if fakeIP, ok := m.ipv4Map[ip]; ok {
		return fakeIP
	}

	index := m.nextIPv4Counter.Add(1)
	if index > fakeIPMaxCount {
		return nil // out of addresses
	}

	fakeIP = IPv4FromUint32(fakeIPv4Subnet + index)
	if err := m.addNftRule4Callback(realIP, fakeIP); err != nil {
		m.nextIPv4Counter.Add(^uint32(0)) // decrement counter on failure
		return nil
	}

	m.ipv4Map[ip] = fakeIP
	return fakeIP
}

func (m *FakeIPManager) GetFakeIPv6(realIP net.IP) net.IP {
	ip := MustIPv6FromIP(realIP)

	m.rwMutex6.RLock()
	fakeIP, ok := m.ipv6Map[ip]
	m.rwMutex6.RUnlock()
	if ok {
		return fakeIP
	}

	m.rwMutex6.Lock()
	defer m.rwMutex6.Unlock()

	if fakeIP, ok := m.ipv6Map[ip]; ok {
		return fakeIP
	}

	index := m.nextIPv6Counter.Add(1)
	if index > fakeIPMaxCount {
		return nil
	}

	fakeIP = IPv6FromUint64(fakeIPv6SubnetHigh, index)
	if err := m.addNftRule6Callback(realIP, fakeIP); err != nil {
		m.nextIPv6Counter.Add(^uint64(0)) // decrement on failure
		return nil
	}

	m.ipv6Map[ip] = fakeIP
	return fakeIP
}

func IPv6FromUint64(hi, lo uint64) net.IP {
	var b [16]byte
	binary.BigEndian.PutUint64(b[0:8], hi)
	binary.BigEndian.PutUint64(b[8:16], lo)
	return net.IP(b[:])
}

func IPv4FromUint32(v uint32) net.IP {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], v)
	return net.IP(b[:])
}
