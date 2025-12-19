package main

import (
	"fmt"
	"log"
	"net"
	"os/exec"
)

func RunNftCommand(args ...string) error {
	cmd := exec.Command("nft", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("nft command %v failed: %w\nOutput: %s", args, err, string(output))
	}
	return nil
}

func SetupNftables(ipv4Subnet, ipv6Subnet string, routingMark uint32) error {
	log.Println("Attempting nftables setup...")
	_ = RunNftCommand("add", "table", nftTableName)

	mangleSpec := fmt.Sprintf("type filter hook prerouting priority %s", nftMarkingHookPrio)
	err := RunNftCommand("add", "chain", nftTableName, nftMarkingChainName, "{", mangleSpec, ";", "}")
	if err != nil {
		return fmt.Errorf("failed to add mangle chain: %w", err)
	}

	RunNftCommand("add", "rule", nftTableName, nftMarkingChainName, "meta", "mark", "set", "ct", "mark")
	RunNftCommand("add", "rule", nftTableName, nftMarkingChainName, "ip", "daddr", ipv4Subnet, "meta", "mark", "set", routingMark, "ct", "mark", "set", "meta", "mark")
	RunNftCommand("add", "rule", nftTableName, nftMarkingChainName, "ip6", "daddr", ipv6Subnet, "meta", "mark", "set", routingMark, "ct", "mark", "set", "meta", "mark")

	// 4. Create NAT Chain (Priority -101)
	natSpec := fmt.Sprintf("type nat hook prerouting priority %s", nftHookPrio)
	err = RunNftCommand("add", "chain", nftTableName, nftChainName, "{", natSpec, ";", "}")
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
