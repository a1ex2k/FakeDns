package main

import (
	"errors"
	"flag"
	"path/filepath"
)

type Config struct {
	ListenAddr   string
	DomainsPath  string
	ServiceName  string
	TemplatesDir string
	StaticDir    string
}

func mustParseArgs() Config {
	cfg, err := parseArgs()
	if err != nil {
		panic(err) // позже заменим на log.Fatal если захочешь
	}
	return cfg
}

func parseArgs() (Config, error) {
	listen := flag.String("listen", "127.0.0.1:8080", "listen address (ip:port)")
	domains := flag.String("domains", "", "path to fake-domains.list")
	service := flag.String("service", "fakedns.service", "systemd service name")

	templatesDir := flag.String(
		"templates-dir",
		filepath.Join("cmd", "fakedns-webui", "templates"),
		"templates directory",
	)
	staticDir := flag.String(
		"static-dir",
		filepath.Join("cmd", "fakedns-webui", "static"),
		"static files directory",
	)

	flag.Parse()

	if *domains == "" {
		return Config{}, errors.New("you must specify -domains=/path/to/fake-domains.list")
	}

	return Config{
		ListenAddr:   *listen,
		DomainsPath:  *domains,
		ServiceName:  *service,
		TemplatesDir: *templatesDir,
		StaticDir:    *staticDir,
	}, nil
}
