package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func loadDomains(filePath string) (map[string]struct{}, error) {
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
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		domain := strings.ToLower(strings.TrimSuffix(line, "."))
		if domain != "" {
			domains[domain] = struct{}{}
		} else {
			log.Printf("Warning: Skipping invalid domain entry on line %d: '%s'", lineNum, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading domains file '%s': %w", filePath, err)
	}

	if len(domains) == 0 {
		log.Printf("Warning: No domains loaded from '%s'", filePath)
	} else {
		log.Printf("Loaded %d target domains from '%s'", len(domains), filePath)
	}

	return domains, nil
}
