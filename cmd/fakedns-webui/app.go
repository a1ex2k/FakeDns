package main

import (
	"errors"
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

	domain := NormalizeDomain(r.FormValue("domain"))
	if domain == "" {
		http.Redirect(w, r, "/?msg=Empty+domain", http.StatusSeeOther)
		return
	}
	if strings.ContainsAny(domain, " \t\r\n/") {
		http.Redirect(w, r, "/?msg=Invalid+domain", http.StatusSeeOther)
		return
	}

	if err := a.store.Add(domain); err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			http.Redirect(w, r, "/?msg=Already+exists", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/?msg=Add+failed", http.StatusSeeOther)
		return
	}

	if err := a.reload.Reload(); err != nil {
		http.Redirect(w, r, "/?msg=Added,+but+reload+failed", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/?msg=Added", http.StatusSeeOther)
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
