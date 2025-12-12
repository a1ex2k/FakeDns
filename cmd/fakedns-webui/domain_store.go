package main

import (
	"bufio"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
)

// DomainStore — file-backed хранилище доменов
type DomainStore struct {
	path string
	mu   sync.Mutex
}

func NewDomainStore(path string) *DomainStore {
	return &DomainStore{path: path}
}

// List — получить список доменов
func (s *DomainStore) List() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return readDomains(s.path)
}

// AddMany добавляет сразу несколько доменов под одним mutex.
// Возвращает (added, skipped, err)
func (s *DomainStore) AddMany(domains []string) (int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := readDomains(s.path)
	if err != nil {
		return 0, 0, err
	}

	exists := make(map[string]struct{}, len(current))
	for _, d := range current {
		exists[NormalizeDomain(d)] = struct{}{}
	}

	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return 0, 0, err
	}
	defer closeAndLog(f)

	added := 0
	skipped := 0

	for _, d := range domains {
		if _, ok := exists[d]; ok {
			skipped++
			continue
		}
		if _, err := f.WriteString(d + "\n"); err != nil {
			return added, skipped, err
		}
		exists[d] = struct{}{}
		added++
	}

	return added, skipped, nil
}

// Delete — удалить домен
// возвращает (удалён ли домен, ошибка)
func (s *DomainStore) Delete(domain string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	domains, err := readDomains(s.path)
	if err != nil {
		return false, err
	}

	var out []string
	removed := false

	for _, d := range domains {
		if NormalizeDomain(d) == domain {
			removed = true
			continue
		}
		out = append(out, d)
	}

	if !removed {
		return false, nil
	}

	if err := writeDomains(s.path, out); err != nil {
		return false, err
	}

	return true, nil
}

/* ===== helpers ===== */

func readDomains(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		// если файла ещё нет — считаем список пустым
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer closeAndLog(f)

	var res []string
	sc := bufio.NewScanner(f)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		res = append(res, line)
	}

	return res, sc.Err()
}

func writeDomains(path string, domains []string) error {
	f, err := os.OpenFile(path, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer closeAndLog(f)

	for _, d := range domains {
		if _, err := f.WriteString(d + "\n"); err != nil {
			return err
		}
	}
	return nil
}

func NormalizeDomain(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ".")
	s = strings.ToLower(s)
	return s
}

func closeAndLog(f *os.File) {
	if err := f.Close(); err != nil {
		fn := callerName(1)
		log.Printf("file close error (%s): %v", fn, err)
	}
}

func callerName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown"
	}
	if fn := runtime.FuncForPC(pc); fn != nil {
		return fn.Name()
	}
	return "unknown"
}
