package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

type DomainsList struct {
	targetDomains map[string]struct{}
}

func DomainsListFromMap(domains map[string]struct{}) *DomainsList {
	return &DomainsList{targetDomains: domains}
}

func DomainsListFromFile(filePath string) (*DomainsList, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open domains file '%s': %w", filePath, err)
	}
	defer file.Close()

	domains := make(map[string]struct{})
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if idx := strings.Index(line, "#"); idx != -1 {
			line = line[:idx]
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		domain := strings.ToLower(strings.TrimSuffix(line, "."))
		if domain == "" {
			log.Printf("Warning: Skipping invalid domain entry on line %d", lineNum)
			continue
		}

		domains[domain] = struct{}{}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading domains file '%s': %w", filePath, err)
	}

	if len(domains) == 0 {
		log.Printf("Warning: No domains loaded from '%s'", filePath)
	} else {
		log.Printf("Loaded %d target domains from '%s'", len(domains), filePath)
	}

	return &DomainsList{targetDomains: domains}, nil
}

func (d *DomainsList) MatchDomain(queryDomain string, maxLevel int) bool {
	queryDomain = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(queryDomain), "."))
	parts := strings.Split(queryDomain, ".")
	n := len(parts)

	if maxLevel == 0 || maxLevel > n {
		maxLevel = n
	}

	for i := n - maxLevel; i < n; i++ {
		domain := strings.Join(parts[i:], ".")
		if _, ok := d.targetDomains[domain]; ok {
			return true
		}
	}
	return false
}
