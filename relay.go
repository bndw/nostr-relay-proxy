package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/fiatjaf/eventstore"
	"github.com/nbd-wtf/go-nostr"
)

func newRelay(log logger, cfg Config) *relay {
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
	log     logger
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
	if !r.config.PubkeyIsAllowed(event.PubKey) {
		r.log.Infof("pubkey not authorized to write: %q", event.PubKey)
		return false
	}

	return true
}

func (r *relay) ServiceURL() string {
	return r.config.NIP42ServiceURL
}

func (r *relay) AcceptReq(ctx context.Context, id string, filters nostr.Filters, pk string) bool {
	if pk == "" {
		return false
	}

	if !r.config.PubkeyIsAllowed(pk) {
		r.log.Infof("pubkey not authorized to read: %q", pk)
		return false
	}

	return true
}

// proxyStore implements the relay's eventstore interface against the
// configured read and write relays.
type proxyStore struct {
	log    logger
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
	tok := genToken()
	s.log.Infof("QueryEvents[%s]: %#v", tok, filter)

	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.config.QueryEventsTimeoutSeconds)*time.Second)

	var (
		events       = make(chan *nostr.Event)
		subscription = nostr.NewSimplePool(ctx).SubMany(ctx, s.config.ReadRelays, []nostr.Filter{filter})
	)

	go func() {
		defer func() {
			s.log.Infof("QueryEvents[%s]: close chan", tok)
			cancel()
			close(events)
		}()

		for event := range subscription {
			events <- event.Event
		}
	}()

	return events, nil
}

func (s proxyStore) SaveEvent(ctx context.Context, event *nostr.Event) error {
	s.log.Infof("SaveEvent: %#v", *event)

	for _, url := range s.config.WriteRelays {
		relay, err := nostr.RelayConnect(ctx, url)
		if err != nil {
			s.log.Infof("err: relay connect %q: %v", url, err)
			continue
		}
		defer relay.Close()

		err = relay.Publish(ctx, *event)
		if err != nil {
			s.log.Infof("err: relay publish %q: %v", url, err)
			continue
		}

		s.log.Infof("published to %s", url)
	}

	return nil
}

func genToken() string {
	tkn := make([]byte, 4)
	rand.Read(tkn)
	return fmt.Sprintf("%x", tkn)
}
