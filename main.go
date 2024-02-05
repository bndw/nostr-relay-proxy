package main

import (
	"flag"
	"os"

	"github.com/fiatjaf/relayer/v2"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	log := newLogger()

	cfg, err := LoadConfig(*cfgPath)
	if err != nil {
		log.Errorf("load config: %v", err)
		os.Exit(1)
	}
	log.setLogLevel(parseLogLevel(cfg.LogLevel))

	log.Infof("loaded config: %#v", cfg)

	r := newRelay(log, cfg)

	server, err := relayer.NewServer(r)
	if err != nil {
		log.Errorf("new server: %v", err)
		os.Exit(1)
	}
	server.Log = log

	log.Errorf("listening on: 0.0.0.0:%d", cfg.Port)
	if err := server.Start("0.0.0.0", cfg.Port); err != nil {
		log.Errorf("server terminated: %v", err)
		os.Exit(1)
	}
}
