package main

import (
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

	ui, err := NewUI(cfg.TemplatesDir, cfg.StaticDir)
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

	// статика: /static/styles.css
	mux.Handle("/static/", a.ui.StaticHandler())

	// страницы/действия
	mux.HandleFunc("/", a.handleIndex)
	mux.HandleFunc("/add", a.handleAdd)
	mux.HandleFunc("/delete", a.handleDelete)

	return http.ListenAndServe(a.cfg.ListenAddr, mux)
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	domains, err := a.store.List()
	if err != nil {
		http.Error(w, "failed to read domains: "+err.Error(), http.StatusInternalServerError)
		return
	}

	model := UIModel{
		Domains: domains,
		Message: r.URL.Query().Get("msg"),
	}

	if err := a.ui.RenderIndex(w, model); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *App) handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	raw := r.FormValue("domains")
	domains := splitDomains(raw)

	if len(domains) == 0 {
		http.Redirect(w, r, "/?msg=Empty", http.StatusSeeOther)
		return
	}

	added, skipped, removed, err := a.store.MergeMany(domains)
	if err != nil {
		http.Redirect(w, r, "/?msg=Add+failed", http.StatusSeeOther)
		return
	}

	// reload только если были изменения (added или removed)
	if added > 0 || removed > 0 {
		if err := a.reload.Reload(); err != nil {
			http.Redirect(w, r, "/?msg=Changed,+but+reload+failed", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/?msg=Done", http.StatusSeeOther)
		return
	}

	// изменений не было
	if skipped > 0 {
		http.Redirect(w, r, "/?msg=No+changes", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/?msg=No+changes", http.StatusSeeOther)
}

func (a *App) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	target := NormalizeDomain(r.FormValue("domain"))
	if target == "" {
		http.Redirect(w, r, "/?msg=Empty+domain", http.StatusSeeOther)
		return
	}

	removed, err := a.store.Delete(target)
	if err != nil {
		http.Redirect(w, r, "/?msg=Delete+failed", http.StatusSeeOther)
		return
	}
	if !removed {
		http.Redirect(w, r, "/?msg=Not+found", http.StatusSeeOther)
		return
	}

	if err := a.reload.Reload(); err != nil {
		http.Redirect(w, r, "/?msg=Deleted,+but+reload+failed", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/?msg=Deleted", http.StatusSeeOther)
}

func splitDomains(raw string) []string {
	lines := strings.Split(raw, "\n")

	seen := make(map[string]struct{})
	out := make([]string, 0, len(lines))

	for _, line := range lines {
		d := NormalizeDomain(line)
		if d == "" {
			continue
		}
		// очень базовая фильтрация мусора
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
