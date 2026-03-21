package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
		if os.IsNotExist(err) {
			// Create empty file (and parent dirs) and continue
			if mkErr := os.MkdirAll(filepath.Dir(d.filePath), 0755); mkErr != nil {
				return fmt.Errorf("failed to create directories for '%s': %w", d.filePath, mkErr)
			}
			f, createErr := os.OpenFile(d.filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
			if createErr != nil && !os.IsExist(createErr) {
				return fmt.Errorf("failed to create domains file '%s': %w", d.filePath, createErr)
			}
			if createErr == nil {
				if closeErr := f.Close(); closeErr != nil {
					log.Printf("Warning: Failed to close newly-created domains file '%s': %v", d.filePath, closeErr)
				}
			}

			// Now open for reading
			file, err = os.Open(d.filePath)
			if err != nil {
				return fmt.Errorf("failed to open domains file '%s' after creating: %w", d.filePath, err)
			}

			log.Printf("Domains file '%s' did not exist; created empty file", d.filePath)
		} else {
			return fmt.Errorf("failed to open domains file '%s': %w", d.filePath, err)
		}
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Warning: Failed to close domains file '%s': %v", d.filePath, closeErr)
		}
	}()

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

	queryDomain = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(queryDomain), "."))
	if queryDomain == "" {
		return false
	}

	labelCount := 1 + strings.Count(queryDomain, ".")
	if maxLevel == 0 || maxLevel > labelCount {
		maxLevel = labelCount
	}
	skipLevels := labelCount - maxLevel

	d.mu.RLock()
	defer d.mu.RUnlock()

	suffix := queryDomain
	for level := 0; ; level++ {
		if level >= skipLevels {
			if _, ok := d.targetDomains[suffix]; ok {
				return true
			}
		}

		dotIdx := strings.IndexByte(suffix, '.')
		if dotIdx == -1 {
			break
		}
		suffix = suffix[dotIdx+1:]
	}
	return false
}
