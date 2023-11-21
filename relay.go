package main

import (
	"context"
	"log"
	"time"

	"github.com/fiatjaf/eventstore"
	"github.com/nbd-wtf/go-nostr"
)

func newRelay(log *log.Logger, cfg Config) *relay {
	return &relay{
		log:    log,
		config: cfg,
		storage: &proxyStore{
			log:    log,
			config: cfg,
		},
	}
}

type relay struct {
	log     *log.Logger
	config  Config
	storage *proxyStore
}

func (r *relay) Name() string {
	return "nostr-relay-proxy"
}

func (r *relay) Storage(ctx context.Context) eventstore.Store {
	return r.storage
}

func (r *relay) Init() error {
	return nil
}

func (r *relay) AcceptEvent(ctx context.Context, event *nostr.Event) bool {
	if !r.config.PubkeyIsAllowedToWrite(event.PubKey) {
		r.log.Printf("pubkey not authorized to write: %q\n", event.PubKey)
		return false
	}

	return true
}

// proxyStore implements the relay's eventstore interface against the
// configured read and write relays.
type proxyStore struct {
	log    *log.Logger
	config Config
}

func (s proxyStore) Init() error {
	return nil
}

func (s proxyStore) Close() {}

func (s proxyStore) DeleteEvent(ctx context.Context, evt *nostr.Event) error {
	return nil
}

func (s proxyStore) QueryEvents(ctx context.Context, filter nostr.Filter) (chan *nostr.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.config.QueryEventsTimeoutSeconds)*time.Second)

	var (
		events       = make(chan *nostr.Event)
		subscription = nostr.NewSimplePool(ctx).SubMany(ctx, s.config.ReadRelays, []nostr.Filter{filter})
	)

	go func() {
		for event := range subscription {
			events <- event.Event
		}

		cancel()
		close(events)
	}()

	return events, nil
}

func (s proxyStore) SaveEvent(ctx context.Context, event *nostr.Event) error {
	s.log.Printf("received event: %#v\n", *event)

	for _, url := range s.config.WriteRelays {
		relay, err := nostr.RelayConnect(ctx, url)
		if err != nil {
			s.log.Printf("relay connect: %v", err)
			continue
		}
		defer relay.Close()

		_, err = relay.Publish(ctx, *event)
		if err != nil {
			s.log.Printf("relay publish: %v", err)
			continue
		}

		s.log.Printf("published to %s\n", url)
	}

	return nil
}
