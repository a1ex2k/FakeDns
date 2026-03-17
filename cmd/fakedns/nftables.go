package main

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"strconv"
)

func RunNftCommand(args ...string) error {
	cmd := exec.Command("nft", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("nft command %v failed: %w\nOutput: %s", args, err, string(output))
	}
	return nil
}

func SetupNftables(ipv4Subnet, ipv6Subnet string, fwmarkMask uint) error {
	log.Println("Attempting nftables setup...")
	_ = RunNftCommand("add", "table", nftTableName)

	if fwmarkMask > 0 {
		mangleSpec := fmt.Sprintf("type filter hook prerouting priority %s", nftMarkingHookPrio)
		err := RunNftCommand("add chain", nftTableName, nftMarkingChainName, "{", mangleSpec, ";}")
		if err != nil {
			return fmt.Errorf("failed to add mangle chain: %w", err)
		}
		fwmarkString := strconv.FormatUint(uint64(fwmarkMask), 16)
		RunNftCommand("add rule", nftTableName, nftMarkingChainName, "meta mark set ct mark &", fwmarkString)
		RunNftCommand("add rule", nftTableName, nftMarkingChainName, "ct state new ip daddr", ipv4Subnet, "meta mark set meta mark |", fwmarkString, "ct mark set ct mark |", fwmarkString)
		RunNftCommand("add rule", nftTableName, nftMarkingChainName, "ct state new ip6 daddr", ipv6Subnet, "meta mark set meta mark |", fwmarkString, "ct mark set ct mark |", fwmarkString)
	}

	natSpec := fmt.Sprintf("type nat hook prerouting priority %s", nftHookPrio)
	err := RunNftCommand("add chain", nftTableName, nftChainName, "{", natSpec, ";}")
	if err != nil {
		return fmt.Errorf("failed to add nat chain: %w", err)
	}

	log.Println("nftables setup complete with fwmarking and NAT chains.")
	return nil
}

func AddDnat4Rule(realIP, fakeIP net.IP) error {
	ruleArgs := []string{
		"add", "rule", nftTableName, nftChainName,
		"ip", "daddr", fakeIP.String(),
		"dnat", "to", realIP.String(),
	}
	return RunNftCommand(ruleArgs...)
}

func AddDnat6Rule(realIP, fakeIP net.IP) error {
	ruleArgs := []string{
		"add", "rule", nftTableName, nftChainName,
		"ip6", "daddr", fakeIP.String(),
		"dnat", "to", realIP.String(),
	}
	return RunNftCommand(ruleArgs...)
}

func CleanupNftables() error {
	log.Println("Attempting nftables cleanup...")
	return RunNftCommand("delete", "table", nftTableName)
}
