package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/fiatjaf/khatru"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	store := newProxyStore(log, cfg)
	if err := store.Init(); err != nil {
		log.Errorf("store init: %v", err)
		os.Exit(1)
	}

	relay := khatru.NewRelay()
	relay.ServiceURL = cfg.RelayURL
	relay.Info.Name = cfg.RelayName
	relay.Info.PubKey = cfg.decodeRelayNpub()
	relay.Info.Contact = cfg.RelayContact
	relay.Info.Description = cfg.RelayDescription
	relay.Info.Icon = cfg.RelayIconURL
	relay.Info.Software = "https://github.com/bndw/nostr-relay-proxy"
	relay.Info.Version = cfg.RelayVersion
	relay.StoreEvent = append(relay.StoreEvent, store.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, store.QueryEvents)
	relay.DeleteEvent = append(relay.DeleteEvent, store.DeleteEvent)
	relay.RejectEvent = append(relay.RejectEvent, store.RejectEvent)
	relay.RejectFilter = append(relay.RejectFilter, store.RejectFilter)
	relay.Router().HandleFunc("/", indexHandler(store))
	relay.Router().Handle("/metrics", promhttp.Handler())

	listenAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Errorf("listening on: %s", listenAddr)
	http.ListenAndServe(listenAddr, relay)
}
