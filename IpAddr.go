package fakedns

import (
	"encoding/binary"
	"fmt"
	"net"
)

type IPv4Addr = uint32
type IPv6Addr struct {
	hi, lo uint64
}

func IPv4FromIP(ip net.IP) (IPv4Addr, error) {
	ip4 := ip.To4()
	if ip4 == nil {
		return 0, fmt.Errorf("not an IPv4 address: %v", ip)
	}
	return binary.BigEndian.Uint32(ip4), nil
}

func IPv6FromIP(ip net.IP) (IPv6Addr, error) {
	ip16 := ip.To16()
	if ip16 == nil || ip.To4() != nil {
		return IPv6Addr{hi: 0, lo: 0}, fmt.Errorf("not an IPv6 address: %v", ip)
	}

	hi := binary.BigEndian.Uint64(ip16[:8])
	lo := binary.BigEndian.Uint64(ip16[8:])
	return IPv6Addr{hi: hi, lo: lo}, nil
}

func MustIPv4FromIP(ip net.IP) IPv4Addr {
	v, err := IPv4FromIP(ip)
	if err != nil {
		panic(err)
	}
	return v
}

func MustIPv6FromIP(ip net.IP) IPv6Addr {
	v, err := IPv6FromIP(ip)
	if err != nil {
		panic(err)
	}
	return v
}
