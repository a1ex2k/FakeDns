package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

type DomainsList struct {
	targetDomains map[string]struct{}
	filePath      string
	mu            sync.RWMutex
}

func (d *DomainsList) Load() error {
	file, err := os.Open(d.filePath)
	if err != nil {
		return fmt.Errorf("failed to open domains file '%s': %w", d.filePath, err)
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
		return fmt.Errorf("error reading domains file '%s': %w", d.filePath, err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.targetDomains = domains
	if len(d.targetDomains) == 0 {
		log.Printf("Warning: No domains loaded from '%s'", d.filePath)
	} else {
		log.Printf("Loaded %d target domains from '%s'", len(d.targetDomains), d.filePath)
	}

	return nil
}

func DomainsListFromFile(filePath string) (*DomainsList, error) {
	d := &DomainsList{
		targetDomains: make(map[string]struct{}),
		filePath:      filePath,
	}

	if err := d.Load(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *DomainsList) MatchDomain(queryDomain string, maxLevel int) bool {
	if d == nil {
		return false
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

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
