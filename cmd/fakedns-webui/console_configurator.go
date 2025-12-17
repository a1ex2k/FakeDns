package main

import (
	"errors"
	"flag"
)

type Config struct {
	ListenAddr  string
	DomainsPath string
	ServiceName string

	NoAuth     bool
	PasswdFile string
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

	noAuth := flag.Bool("no-auth", false, "Disable WebUI authentication (INSECURE)")
	passwd := flag.String("passwd-file", "/etc/fakedns/webui.passwd", "Path to bcrypt password hash file")

	flag.Parse()

	if *domains == "" {
		return Config{}, errors.New("you must specify -domains=/path/to/fake-domains.list")
	}

	return Config{
		ListenAddr:  *listen,
		DomainsPath: *domains,
		ServiceName: *service,
		NoAuth:      *noAuth,
		PasswdFile:  *passwd,
	}, nil
}
