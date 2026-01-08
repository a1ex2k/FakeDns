package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

const (
	fakeIPMaxCount = 131070
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

	ipv4SubnetBase uint32
	ipv4MaxCount   uint32

	ipv6SubnetHigh uint64
	ipv6MaxCount   uint64
}

func NewFakeIPManager(
	fake4CIDR string,
	fake6CIDR string,
	nftCallback4 func(realIP, fakeIP net.IP) error,
	nftCallback6 func(realIP, fakeIP net.IP) error,
) (*FakeIPManager, error) {

	v4base, v4max, err := parseIPv4Pool(fake4CIDR)
	if err != nil {
		return nil, err
	}

	v6hi, v6max, err := parseIPv6Pool(fake6CIDR)
	if err != nil {
		return nil, err
	}

	return &FakeIPManager{
		ipv4Map:             make(map[IPv4Addr]net.IP),
		ipv6Map:             make(map[IPv6Addr]net.IP),
		addNftRule4Callback: nftCallback4,
		addNftRule6Callback: nftCallback6,
		ipv4SubnetBase:      v4base,
		ipv4MaxCount:        v4max,
		ipv6SubnetHigh:      v6hi,
		ipv6MaxCount:        v6max,
	}, nil
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
	if index > m.ipv4MaxCount {
		return nil
	}
	fakeIP = IPv4FromUint32(m.ipv4SubnetBase + index)

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
	if index > m.ipv6MaxCount {
		return nil
	}
	fakeIP = IPv6FromUint64(m.ipv6SubnetHigh, index)

	if err := m.addNftRule6Callback(realIP, fakeIP); err != nil {
		m.nextIPv6Counter.Add(^uint64(0))
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
func parseIPv4Pool(cidr string) (base uint32, maxCount uint32, err error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return 0, 0, err
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return 0, 0, fmt.Errorf("fake4 CIDR is not IPv4: %s", cidr)
	}

	ones, bits := ipnet.Mask.Size()
	if bits != 32 {
		return 0, 0, fmt.Errorf("fake4 CIDR mask is not IPv4: %s", cidr)
	}

	// total addresses in subnet = 2^(hostbits)
	hostBits := 32 - ones
	if hostBits <= 1 {
		return 0, 0, fmt.Errorf("fake4 subnet too small (need at least /30): %s", cidr)
	}

	total := uint32(1) << uint32(hostBits)
	usable := total - 2 // skip network and broadcast

	// keep old behavior: cap at fakeIPMaxCount
	if usable > fakeIPMaxCount {
		usable = fakeIPMaxCount
	}

	base = binary.BigEndian.Uint32(ipnet.IP.To4())
	return base, usable, nil
}

func parseIPv6Pool(cidr string) (hi uint64, maxCount uint64, err error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return 0, 0, err
	}
	ip16 := ip.To16()
	if ip16 == nil || ip.To4() != nil {
		return 0, 0, fmt.Errorf("fake6 CIDR is not IPv6: %s", cidr)
	}

	ones, bits := ipnet.Mask.Size()
	if bits != 128 {
		return 0, 0, fmt.Errorf("fake6 CIDR mask is not IPv6: %s", cidr)
	}

	// минимальное требование под текущую схему (индекс в low 64bits)
	if ones < 64 {
		return 0, 0, fmt.Errorf("fake6 prefix must be >= /64 for this build: %s", cidr)
	}

	hi = binary.BigEndian.Uint64(ipnet.IP[:8])

	// v6 адресов очень много — оставим тот же верхний лимит, как было
	maxCount = fakeIPMaxCount
	return hi, maxCount, nil
}
