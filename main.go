package main

import (
	"flag"
	"log"

	"github.com/fiatjaf/relayer/v2"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	log := log.Default()

	cfg, err := LoadConfig(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	log.Printf("loaded config: %#v", cfg)

	r := newRelay(log, cfg)

	server, err := relayer.NewServer(r)
	if err != nil {
		log.Fatalf("new server: %v", err)
	}

	if err := server.Start("0.0.0.0", cfg.Port); err != nil {
		log.Fatalf("server terminated: %v", err)
	}
}
