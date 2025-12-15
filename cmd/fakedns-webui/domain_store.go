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

// MergeMany добавляет домены с дедупликацией по "родитель покрывает поддомены".
// Возвращает (added, skipped, removedSubdomains, err).
func (s *DomainStore) MergeMany(in []string) (int, int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := readDomains(s.path)
	if err != nil {
		return 0, 0, 0, err
	}

	// Нормализуем текущее в set + slice (без дублей)
	cur := make([]string, 0, len(current))
	curSet := make(map[string]struct{}, len(current))
	for _, d := range current {
		nd := NormalizeDomain(d)
		if nd == "" {
			continue
		}
		if _, ok := curSet[nd]; ok {
			continue
		}
		curSet[nd] = struct{}{}
		cur = append(cur, nd)
	}

	// Нормализуем вход и убираем дубли во входе
	inSet := make(map[string]struct{}, len(in))
	cleanIn := make([]string, 0, len(in))
	for _, d := range in {
		nd := NormalizeDomain(d)
		if nd == "" {
			continue
		}
		if _, ok := inSet[nd]; ok {
			continue
		}
		inSet[nd] = struct{}{}
		cleanIn = append(cleanIn, nd)
	}

	added := 0
	skipped := 0
	removed := 0

	// 1) Отсеиваем входные, которые уже покрыты существующим родителем
	toAdd := make([]string, 0, len(cleanIn))
	for _, d := range cleanIn {
		if isCoveredByParent(d, curSet) {
			skipped++
			continue
		}
		toAdd = append(toAdd, d)
	}

	// 2) Добавляем каждый новый домен:
	//    - выкидываем из текущего списка все его поддомены
	//    - добавляем домен
	for _, d := range toAdd {
		// удаляем поддомены d из cur
		if len(cur) > 0 {
			next := cur[:0]
			for _, x := range cur {
				if x != d && isSubdomainOf(x, d) {
					delete(curSet, x)
					removed++
					continue
				}
				next = append(next, x)
			}
			cur = next
		}

		// добавляем d
		cur = append(cur, d)
		curSet[d] = struct{}{}
		added++
	}

	// Если вообще не было изменений — файл не трогаем
	if added == 0 && removed == 0 {
		return 0, skipped, 0, nil
	}

	if err := writeDomains(s.path, cur); err != nil {
		return added, skipped, removed, err
	}

	return added, skipped, removed, nil
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

func isCoveredByParent(domain string, existing map[string]struct{}) bool {
	// domain уже нормализован
	if _, ok := existing[domain]; ok {
		return true
	}

	// a.b.example.com -> b.example.com -> example.com -> com
	for {
		i := strings.IndexByte(domain, '.')
		if i == -1 {
			return false
		}
		domain = domain[i+1:]
		if _, ok := existing[domain]; ok {
			return true
		}
	}
}

func isSubdomainOf(child, parent string) bool {
	if child == parent {
		return true
	}
	return strings.HasSuffix(child, "."+parent)
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
