package main

import (
	"embed"
	"io/fs"
	"net/http"
)

// Вшиваем весь Web UI в бинарь
// Структура:
// cmd/fakedns-webui/embedWeb/
//
//	index.html
//	styles.css
//	scripts.js
//
//go:embed embedWeb/*
var embedWebFS embed.FS

type UI struct {
	handler http.Handler
}

func NewUI() (*UI, error) {
	sub, err := fs.Sub(embedWebFS, "embedWeb")
	if err != nil {
		return nil, err
	}

	return &UI{
		handler: http.FileServer(http.FS(sub)),
	}, nil
}

// Handler возвращает HTTP-хендлер для Web UI
// Обычно монтируется так:
// mux.Handle("/ui/", http.StripPrefix("/ui/", ui.Handler()))
func (u *UI) Handler() http.Handler {
	return u.handler
}
