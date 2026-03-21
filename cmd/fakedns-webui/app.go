package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrAuthNotInitialized = errors.New("auth not initialized")

const maxJSONBodyBytes = 1 << 20

type App struct {
	cfg          Config
	store        *DomainStore
	ui           *UI
	reload       *ServiceReloader
	authUser     string
	passwordHash string
}

func NewApp(cfg Config) (*App, error) {

	store := NewDomainStore(cfg.DomainsPath)
	reloader := NewServiceReloader(cfg.ServiceName)

	ui, err := NewUI()
	if err != nil {
		return nil, err
	}
	var authUser, passwordHash string
	if !cfg.NoAuth {
		u, h, err := loadAuthFile(cfg.PasswdFile)
		if err != nil {
			return nil, ErrAuthNotInitialized
		}
		authUser, passwordHash = u, h
	}
	return &App{
		cfg:          cfg,
		store:        store,
		ui:           ui,
		reload:       reloader,
		authUser:     authUser,
		passwordHash: passwordHash,
	}, nil
}

func (a *App) Run() error {
	mux := http.NewServeMux()
	var uiHandler http.Handler = http.StripPrefix("/ui/", a.ui.Handler())
	if !a.cfg.NoAuth {
		uiHandler = basicAuth(a.authUser, a.passwordHash, uiHandler)
	}
	mux.Handle("/ui/", uiHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/ui/", http.StatusFound)
	})

	apiMux := http.NewServeMux()
	apiMux.Handle("/api/add", http.HandlerFunc(a.handleAdd))
	apiMux.Handle("/api/delete", http.HandlerFunc(a.handleDelete))
	apiMux.Handle("/api/list", http.HandlerFunc(a.handleListDomains))
	apiMux.Handle("/api/action", http.HandlerFunc(a.handleServiceAction))

	var apiHandler http.Handler = apiMux
	if !a.cfg.NoAuth {
		apiHandler = basicAuth(a.authUser, a.passwordHash, apiHandler)
	}
	mux.Handle("/api/", apiHandler)

	server := &http.Server{
		Addr:              a.cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	return server.ListenAndServe()
}

// =========================
// Handlers
// =========================

func (a *App) reply(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ApiResponse{
		Status:  status,
		Message: message,
	})
}

func (a *App) handleServiceAction(w http.ResponseWriter, r *http.Request) {
	var req ServiceActionRequest
	if err := readJSON(r, &req); err != nil {
		a.reply(w, http.StatusBadRequest, err.Error())
		return
	}

	var err error
	switch req.Action {
	case "start":
		err = a.reload.Start()
	case "restart":
		err = a.reload.Restart()
	case "reload":
		err = a.reload.Reload()
	case "stop":
		err = a.reload.Stop()
	default:
		err = fmt.Errorf("Invalid action requested")
	}

	if err != nil {
		a.reply(w, http.StatusInternalServerError, "Failed to run action: "+err.Error())
		return
	}

	a.reply(w, http.StatusOK, fmt.Sprintf("FakeDNS service %sed!", req.Action))
}

func (a *App) handleAdd(w http.ResponseWriter, r *http.Request) {
	var req AddDomainRequest
	if err := readJSON(r, &req); err != nil {
		a.reply(w, http.StatusBadRequest, err.Error())
		return
	}

	normalized := NormalizeDomains(req.Domains)
	if len(normalized) == 0 {
		a.reply(w, http.StatusBadRequest, "No valid domains provided")
		return
	}

	if len(normalized) != len(req.Domains) {
		a.reply(w, http.StatusBadRequest, "Some domain entries are invalid")
		return
	}

	added, skipped, removed, err := a.store.MergeMany(normalized)
	if err != nil {
		a.reply(w, http.StatusInternalServerError, "Failed to update storage: "+err.Error())
		return
	}

	if added > 0 || removed > 0 {
		a.reply(w, http.StatusOK, "Domain(s) added")
		return
	}

	if skipped > 0 {
		a.reply(w, http.StatusNoContent, "No changes")
		return
	}

	a.reply(w, http.StatusNoContent, "No changes")
}

func (a *App) handleDelete(w http.ResponseWriter, r *http.Request) {
	var req RemoveDomainRequest
	if err := readJSON(r, &req); err != nil {
		a.reply(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Domain = NormalizeDomain(req.Domain); req.Domain == "" {
		a.reply(w, http.StatusBadRequest, "Domain is invalid")
		return
	}

	removed, err := a.store.Delete(req.Domain)
	if err != nil {
		a.reply(w, http.StatusInternalServerError, "Failed to delete from storage: "+err.Error())
		return
	}
	if !removed {
		a.reply(w, http.StatusNotFound, "Domain not found")
		return
	}

	a.reply(w, http.StatusOK, "Deleted")
}

func (a *App) handleListDomains(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.reply(w, http.StatusMethodNotAllowed, "Method not allowed, must be GET")
		return
	}

	domains, err := a.store.List()
	if err != nil {
		a.reply(w, http.StatusInternalServerError, "Failed to read domains: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(DomainListResponse{
		Domains: &domains})
}

// =========================
// Utils
// =========================

func readJSON[T any](r *http.Request, dst *T) error {
	if r.Method != http.MethodPost {
		return fmt.Errorf("Method not allowed, must be POST")
	}

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return fmt.Errorf("Content-type must be application/json")
	}

	dec := json.NewDecoder(io.LimitReader(r.Body, maxJSONBodyBytes))
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("request body must contain only one JSON object")
	}
	return nil
}
