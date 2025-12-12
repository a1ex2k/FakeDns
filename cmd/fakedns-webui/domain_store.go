package main

import (
	"bufio"
	"errors"
	"os"
	"strings"
	"sync"
)

// Ошибка, если домен уже существует
var ErrAlreadyExists = errors.New("domain already exists")

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

// Add — добавить домен (с проверкой на дубликаты)
func (s *DomainStore) Add(domain string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	domains, err := readDomains(s.path)
	if err != nil {
		return err
	}

	for _, d := range domains {
		if NormalizeDomain(d) == domain {
			return ErrAlreadyExists
		}
	}

	return appendDomain(s.path, domain)
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
	defer f.Close()

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

func appendDomain(path, domain string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(domain + "\n")
	return err
}

func writeDomains(path string, domains []string) error {
	f, err := os.OpenFile(path, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

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
