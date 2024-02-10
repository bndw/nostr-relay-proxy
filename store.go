package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/fiatjaf/eventstore"
	"github.com/fiatjaf/eventstore/lmdb"
	"github.com/fiatjaf/eventstore/nullstore"
	"github.com/fiatjaf/khatru"
	"github.com/nbd-wtf/go-nostr"
)

func newProxyStore(log logger, cfg Config) *proxyStore {
	return &proxyStore{
		log:    log,
		config: cfg,
		db:     &nullstore.NullStore{},
	}
}

type proxyStore struct {
	log    logger
	config Config

	db   eventstore.Store
	pool *nostr.SimplePool
}

func (s *proxyStore) Init() error {
	if !s.config.DisableLocalDB {
		s.db = &lmdb.LMDBBackend{Path: s.config.LocalDBPath}
		if err := s.db.Init(); err != nil {
			return err
		}
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
		wg                        sync.WaitGroup
		seen                      sync.Map
		events                    = make(chan *nostr.Event)
		localCount, upstreamCount = 0, 0
	)

	// Local db
	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
			s.log.Infof("QueryEvents[%s]: local done", tok)
		}()

		local, err := s.db.QueryEvents(ctx, filter)
		if err != nil {
			s.log.Infof("err QueryEvents[%s] local err: %v", tok, err)
			return
		}

		for event := range local {
			if _, ok := seen.Load(event.ID); ok {
				continue
			}

			events <- event
			localCount++

			seen.Store(event.ID, struct{}{})
		}
	}()

	// Upstream relays
	wg.Add(1)
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

		upstream := s.pool.SubMany(ctx, s.config.ReadRelays, []nostr.Filter{filter})
		for event := range upstream {
			if _, ok := seen.Load(event.ID); ok {
				continue
			}

			events <- event.Event
			upstreamCount++

			seen.Store(event.ID, struct{}{})
			s.db.SaveEvent(ctx, event.Event)
		}
	}()

	go func() {
		<-ctx.Done()
		wg.Wait()
		s.log.Infof("QueryEvents[%s]: complete. %d local %d upstream", tok, localCount, upstreamCount)
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
