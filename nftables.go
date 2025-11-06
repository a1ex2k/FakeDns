package fakedns

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"runtime"
	"strings"
)

func RunNftCommand(args ...string) error {
	cmd := exec.Command("nft", args...)
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

func SetupNftables() error {
	log.Println("Attempting nftables setup...")

	err := RunNftCommand("add table", nftTableName)
	if err != nil && runtime.GOOS == "linux" {
		if !strings.Contains(err.Error(), "exists") {
			return fmt.Errorf("failed to add nft table: %w", err)
		}
		log.Println("nft table already exists, continuing...")
	}

	chainSpec := fmt.Sprintf("type nat hook prerouting priority %s", nftHookPrio)
	err = RunNftCommand("add", "chain", nftTableName, nftChainName, "{", chainSpec, ";", "}")
	if err != nil && runtime.GOOS == "linux" {
		if !strings.Contains(err.Error(), "exists") {
			return fmt.Errorf("failed to add nft chain: %w", err)
		}
		log.Println("nft chain already exists, continuing...")
	}

	log.Println("nftables setup processed (logged only on non-Linux).")
	return nil
}

func AddDnat4Rule(realIP, fakeIP net.IP) error {
	ruleArgs := []string{
		"add rule", nftTableName, nftChainName,
		"ip daddr", fakeIP.String(),
		"dnat to", realIP.String(),
	}
	return RunNftCommand(ruleArgs...)
}

func AddDnat6Rule(realIP, fakeIP net.IP) error {
	ruleArgs := []string{
		"add rule", nftTableName, nftChainName,
		"ip6 daddr", fakeIP.String(),
		"dnat to", realIP.String(),
	}
	return RunNftCommand(ruleArgs...)
}

func CleanupNftables() error {
	log.Println("Attempting nftables cleanup...")
	err := RunNftCommand("delete table", nftTableName)

	if err != nil && runtime.GOOS == "linux" {
		log.Printf("Warning: Failed to delete nftables table '%s': %v", nftTableName, err)
		return err
	}
	log.Println("nftables cleanup processed (logged only on non-Linux).")
	return nil
}
