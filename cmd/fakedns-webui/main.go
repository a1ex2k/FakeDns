package main

import "log"

func main() {
	cfg := mustParseArgs()
	app, err := NewApp(cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening on http://%s", cfg.ListenAddr)
	log.Fatal(app.Run())
}
