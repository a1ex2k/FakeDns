package main

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"runtime"
	"strings"
)

func runNftCommand(args ...string) error {
	if runtime.GOOS != "linux" {
		log.Printf("Skipping nftables command on %s OS: nft %s", runtime.GOOS, strings.Join(args, " "))
		return nil
	}

	cmd := exec.Command("nft", args...)
	log.Printf("Running command: nft %s", strings.Join(args, " "))
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("nft command failed: %v\nOutput:\n%s", err, string(output))
		return fmt.Errorf("nft command failed: %w\nOutput: %s", err, string(output))
	}
	if len(strings.TrimSpace(string(output))) > 0 {
		log.Printf("nft command output:\n%s", string(output))
	}
	return nil
}

func setupNftables() error {
	log.Println("Attempting nftables setup...")

	err := runNftCommand("add", "table", nftTableName)
	if err != nil && runtime.GOOS == "linux" {
		if !strings.Contains(err.Error(), "exists") {
			return fmt.Errorf("failed to add nft table: %w", err)
		}
		log.Println("nft table already exists, continuing...")
	}

	chainSpec := fmt.Sprintf("type nat hook prerouting priority %s", nftHookPrio)
	err = runNftCommand("add", "chain", nftTableName, nftChainName, "{", chainSpec, ";", "}")
	if err != nil && runtime.GOOS == "linux" {
		if !strings.Contains(err.Error(), "exists") {
			return fmt.Errorf("failed to add nft chain: %w", err)
		}
		log.Println("nft chain already exists, continuing...")
	}

	log.Println("nftables setup processed (logged only on non-Linux).")
	return nil
}

func addNftRule(realIP, fakeIP net.IP) error {
	var addrFamily string
	if realIP.To4() != nil && fakeIP.To4() != nil {
		addrFamily = "ip"
	} else if realIP.To16() != nil && fakeIP.To16() != nil && realIP.To4() == nil && fakeIP.To4() == nil {
		addrFamily = "ip6"
	} else {
		return fmt.Errorf("mismatched or invalid IP address families: real=%s fake=%s", realIP, fakeIP)
	}

	ruleArgs := []string{
		"add", "rule", nftTableName, nftChainName,
		addrFamily, "daddr", fakeIP.String(),
		"dnat", "to", realIP.String(),
	}
	return runNftCommand(ruleArgs...)
}

func cleanupNftables() error {
	log.Println("Attempting nftables cleanup...")
	err := runNftCommand("delete", "table", nftTableName)

	if err != nil && runtime.GOOS == "linux" {
		log.Printf("Warning: Failed to delete nftables table '%s': %v", nftTableName, err)
		return err
	}
	log.Println("nftables cleanup processed (logged only on non-Linux).")
	return nil
}
