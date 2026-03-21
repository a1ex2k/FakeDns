package main

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"
)

func RunNftCommand(args ...string) error {
	cmd := exec.Command("nft", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("nft command %v failed: %w\nOutput: %s", args, err, string(output))
	}
	return nil
}

func isNftAlreadyExists(err error) bool {
	return err != nil && strings.Contains(err.Error(), "File exists")
}

func isNftMissing(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "No such file or directory") || strings.Contains(msg, "does not exist")
}

func runNftAllowExists(args ...string) error {
	err := RunNftCommand(args...)
	if isNftAlreadyExists(err) {
		return nil
	}
	return err
}

func SetupNftables(ipv4Subnet, ipv6Subnet string, fwmarkMask uint) error {
	log.Println("Attempting nftables setup...")
	if err := runNftAllowExists("add", "table", nftFamily, nftTableName); err != nil {
		return fmt.Errorf("failed to add table: %w", err)
	}

	if fwmarkMask > 0 {
		err := runNftAllowExists(
			"add", "chain", nftFamily, nftTableName, nftMarkingChainName,
			"{", "type", "filter", "hook", "prerouting", "priority", nftMarkingHookPrio, ";", "}",
		)
		if err != nil {
			return fmt.Errorf("failed to add mangle chain: %w", err)
		}
		fwmarkString := fmt.Sprintf("0x%x", fwmarkMask)
		if err := runNftAllowExists(
			"add", "rule", nftFamily, nftTableName, nftMarkingChainName,
			"meta", "mark", "set", "ct", "mark", "&", fwmarkString,
		); err != nil {
			return fmt.Errorf("failed to add ct->meta mark sync rule: %w", err)
		}
		if err := runNftAllowExists(
			"add", "rule", nftFamily, nftTableName, nftMarkingChainName,
			"ct", "state", "new", "ip", "daddr", ipv4Subnet,
			"meta", "mark", "set", "meta", "mark", "|", fwmarkString,
			"ct", "mark", "set", "ct", "mark", "|", fwmarkString,
		); err != nil {
			return fmt.Errorf("failed to add IPv4 fwmark rule: %w", err)
		}
		if err := runNftAllowExists(
			"add", "rule", nftFamily, nftTableName, nftMarkingChainName,
			"ct", "state", "new", "ip6", "daddr", ipv6Subnet,
			"meta", "mark", "set", "meta", "mark", "|", fwmarkString,
			"ct", "mark", "set", "ct", "mark", "|", fwmarkString,
		); err != nil {
			return fmt.Errorf("failed to add IPv6 fwmark rule: %w", err)
		}
	}

	err := runNftAllowExists(
		"add", "chain", nftFamily, nftTableName, nftChainName,
		"{", "type", "nat", "hook", "prerouting", "priority", nftHookPrio, ";", "}",
	)
	if err != nil {
		return fmt.Errorf("failed to add nat chain: %w", err)
	}

	log.Println("nftables setup complete with fwmarking and NAT chains.")
	return nil
}

func AddDnat4Rule(realIP, fakeIP net.IP) error {
	ruleArgs := []string{
		"add", "rule", nftFamily, nftTableName, nftChainName,
		"ip", "daddr", fakeIP.String(),
		"dnat", "to", realIP.String(),
	}
	return RunNftCommand(ruleArgs...)
}

func AddDnat6Rule(realIP, fakeIP net.IP) error {
	ruleArgs := []string{
		"add", "rule", nftFamily, nftTableName, nftChainName,
		"ip6", "daddr", fakeIP.String(),
		"dnat", "to", realIP.String(),
	}
	return RunNftCommand(ruleArgs...)
}

func CleanupNftables() error {
	log.Println("Attempting nftables cleanup...")
	err := RunNftCommand("delete", "table", nftFamily, nftTableName)
	if isNftMissing(err) {
		return nil
	}
	return err
}
