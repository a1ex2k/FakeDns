package main

import (
	"html/template"
	"net/http"
	"path/filepath"
)

type UI struct {
	tpl       *template.Template
	staticDir string
}

type UIModel struct {
	Domains []string
	Message string
}

func NewUI(templatesDir, staticDir string) (*UI, error) {
	indexPath := filepath.Join(templatesDir, "index.html")

	tpl, err := template.ParseFiles(indexPath)
	if err != nil {
		return nil, err
	}

	return &UI{
		tpl:       tpl,
		staticDir: staticDir,
	}, nil
}

// StaticHandler раздаёт /static/* файлы (например styles.css)
func (u *UI) StaticHandler() http.Handler {
	return http.StripPrefix("/static/", http.FileServer(http.Dir(u.staticDir)))
}

// RenderIndex рендерит страницу index.html
func (u *UI) RenderIndex(w http.ResponseWriter, model UIModel) error {
	return u.tpl.Execute(w, model)
}
