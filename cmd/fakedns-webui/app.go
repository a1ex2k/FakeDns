package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type App struct {
	cfg    Config
	store  *DomainStore
	ui     *UI
	reload *ServiceReloader
}

func NewApp(cfg Config) (*App, error) {
	store := NewDomainStore(cfg.DomainsPath)
	reloader := NewServiceReloader(cfg.ServiceName)

	ui, err := NewUI()
	if err != nil {
		return nil, err
	}

	return &App{
		cfg:    cfg,
		store:  store,
		ui:     ui,
		reload: reloader,
	}, nil
}

// Run поднимает HTTP сервер и регистрирует роуты
func (a *App) Run() error {
	mux := http.NewServeMux()

	// =========================
	// Web UI (embedWeb)
	// =========================

	// UI доступен по /ui/*
	mux.Handle("/ui/",
		http.StripPrefix("/ui/", a.ui.Handler()),
	)

	// корень -> /ui/
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", http.StatusFound)
	})

	// =========================Ф
	// Actions / API
	// =========================

	mux.HandleFunc("/add", a.handleAdd)
	mux.HandleFunc("/delete", a.handleDelete)

	mux.HandleFunc("/api/domains", a.handleListDomains)

	return http.ListenAndServe(a.cfg.ListenAddr, mux)
}

// =========================
// Handlers
// =========================

func (a *App) handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	raw := r.FormValue("domains")
	domains := splitDomains(raw)

	if len(domains) == 0 {
		http.Redirect(w, r, "/ui/?msg=Empty", http.StatusSeeOther)
		return
	}

	added, skipped, removed, err := a.store.MergeMany(domains)
	if err != nil {
		http.Redirect(w, r, "/ui/?msg=Add+failed", http.StatusSeeOther)
		return
	}

	// reload только если были изменения
	if added > 0 || removed > 0 {
		if err := a.reload.Reload(); err != nil {
			http.Redirect(w, r, "/ui/?msg=Changed,+but+reload+failed", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/ui/?msg=Done", http.StatusSeeOther)
		return
	}

	if skipped > 0 {
		http.Redirect(w, r, "/ui/?msg=No+changes", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/ui/?msg=No+changes", http.StatusSeeOther)
}

func (a *App) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	target := NormalizeDomain(r.FormValue("domain"))
	if target == "" {
		http.Redirect(w, r, "/ui/?msg=Empty+domain", http.StatusSeeOther)
		return
	}

	removed, err := a.store.Delete(target)
	if err != nil {
		http.Redirect(w, r, "/ui/?msg=Delete+failed", http.StatusSeeOther)
		return
	}
	if !removed {
		http.Redirect(w, r, "/ui/?msg=Not+found", http.StatusSeeOther)
		return
	}

	if err := a.reload.Reload(); err != nil {
		http.Redirect(w, r, "/ui/?msg=Deleted,+but+reload+failed", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/ui/?msg=Deleted", http.StatusSeeOther)
}

func (a *App) handleListDomains(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	domains, err := a.store.List()
	if err != nil {
		http.Error(w, "failed to read domains", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(domains)
}

// =========================
// Utils
// =========================
func splitDomains(raw string) []string {
	lines := strings.Split(raw, "\n")

	seen := make(map[string]struct{})
	out := make([]string, 0, len(lines))

	for _, line := range lines {
		d := NormalizeDomain(line)
		if d == "" {
			continue
		}
		if strings.ContainsAny(d, " \t\r\n/") {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}

	return out
}
