package main

import (
	"errors"
	"log"
	"os"
	"strings"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "passwd" {
		if err := runPasswd("/etc/fakedns/webui.passwd"); err != nil {
			log.Fatal(err)
		}
		log.Println("Password set successfully")
		return
	}

	cfg := mustParseArgs()
	app, err := NewApp(cfg)
	if err != nil {
		if errors.Is(err, ErrAuthNotInitialized) {
			log.Println("Auth enabled but password not initialized. Run: sudo fakedns-webui passwd")
			os.Exit(3)
		}
		log.Fatal(err)
	}

	if cfg.NoAuth {
		log.Println("WARNING: authentication disabled")
		if !strings.HasPrefix(cfg.ListenAddr, "127.0.0.1") {
			log.Println("WARNING: server is not bound to localhost")
		}
	}
	log.Printf("Listening on http://%s", cfg.ListenAddr)
	log.Fatal(app.Run())
}
