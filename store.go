package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/fiatjaf/eventstore/lmdb"
	"github.com/fiatjaf/khatru"
	"github.com/nbd-wtf/go-nostr"
)

func newProxyStore(log logger, cfg Config) *proxyStore {
	return &proxyStore{
		log:    log,
		config: cfg,
		db:     &lmdb.LMDBBackend{Path: cfg.LocalDBPath},
	}
}

type proxyStore struct {
	log    logger
	config Config

	db   *lmdb.LMDBBackend
	pool *nostr.SimplePool
}

func (s *proxyStore) Init() error {
	if err := s.db.Init(); err != nil {
		return err
	}

	s.pool = nostr.NewSimplePool(context.Background())

	for _, url := range s.config.ReadRelays {
		if _, err := s.pool.EnsureRelay(url); err != nil {
			s.log.Errorf("pool connect to %v err: %v", url, err)
		}
	}

	return nil
}

func (s proxyStore) Close() {}

func (s proxyStore) DeleteEvent(ctx context.Context, evt *nostr.Event) error {
	return nil
}

func (s proxyStore) QueryEvents(ctx context.Context, filter nostr.Filter) (chan *nostr.Event, error) {
	tok := genToken()
	s.log.Infof("QueryEvents[%s]: starting", tok)

	var (
		events                  = make(chan *nostr.Event)
		localSent, upstreamSent = 0, 0

		seen sync.Map
		wg   sync.WaitGroup
	)

	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.config.QueryEventsTimeoutSeconds)*time.Second)

	// Local db
	wg.Add(1)
	local, err := s.db.QueryEvents(ctx, filter)
	if err != nil {
		s.log.Infof("err QueryEvents[%s] local err: %v", tok, err)
	}

	go func() {
		defer func() {
			wg.Done()
			s.log.Infof("QueryEvents[%s]: local done", tok)
		}()

		for event := range local {
			if _, ok := seen.Load(event.ID); ok {
				continue
			}

			events <- event
			localSent++

			seen.Store(event.ID, struct{}{})
		}
	}()

	// Upstream relays
	wg.Add(1)
	upstream := s.pool.SubManyEose(ctx, s.config.ReadRelays, []nostr.Filter{filter})
	go func() {
		defer func() {
			wg.Done()
			s.log.Infof("QueryEvents[%s]: upstream done", tok)
		}()
		defer func() {
			// TODO: Some calls to db.SaveEvent are panicing about an out of bounds idx
			if r := recover(); r != nil {
				s.log.Errorf("QueryEvents[%s]: panic recovered: %v", tok, r)
			}
		}()

		for event := range upstream {
			if _, ok := seen.Load(event.ID); ok {
				continue
			}

			events <- event.Event
			upstreamSent++

			seen.Store(event.ID, struct{}{})
			s.db.SaveEvent(ctx, event.Event)
		}
	}()

	go func() {
		wg.Wait()
		s.log.Infof("QueryEvents[%s]: complete. %d local %d upstream", tok, localSent, upstreamSent)
		cancel()
		close(events)
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

	s.db.SaveEvent(ctx, event)

	return nil
}

func (s proxyStore) RejectEvent(ctx context.Context, event *nostr.Event) (bool, string) {
	if len(s.config.AllowedNpubs) > 0 && !s.config.PubkeyIsAllowed(event.PubKey) {
		s.log.Infof("pubkey not authorized to write: %q", event.PubKey)
		return true, "unauthorized pubkey"
	}

	return false, ""
}

func (s proxyStore) RejectFilter(ctx context.Context, filter nostr.Filter) (bool, string) {
	pk := khatru.GetAuthed(ctx)
	if !s.config.DisableAuth && pk == "" {
		return true, "auth-required: only authenticated users can read from this relay"
	}

	if len(s.config.AllowedNpubs) > 0 && !s.config.PubkeyIsAllowed(pk) {
		s.log.Infof("pubkey not authorized to read: %q", pk)
		return true, "unauthorized pubkey"
	}

	return false, ""
}

func genToken() string {
	tkn := make([]byte, 4)
	rand.Read(tkn)
	return fmt.Sprintf("%x", tkn)
}
