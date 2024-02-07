package main

import (
	"os"
	"strings"

	"github.com/nbd-wtf/go-nostr/nip19"
	"gopkg.in/yaml.v3"
)

const (
	defaultPort                      = 8001
	defaultHost                      = "0.0.0.0"
	defaultQueryEventsTimeoutSeconds = 120
	defaultAuthDeadlineSeconds       = 5
)

// Config is the relay configuration.
type Config struct {
	// RelayURL is the websocket address of the relay.
	RelayURL string `yaml:"relay_url"`
	// Port is the listen port.
	Port int `yaml:"port"`
	// Host is the listen host.
	Host string `yaml:"host"`
	// LogLevel sets the log level, either 'debug' or 'error'. Defaults error.
	LogLevel string `yaml:"log_level"`
	// AllowedNpubs is a list of npubs the relay will accept events from.
	AllowedNpubs []string `yaml:"allowed_npubs"`
	// ReadRelays is a list of relay URLs events will be read from.
	ReadRelays []string `yaml:"read_relays"`
	// WriteRelays is a list of relay URLs new events will be written to.
	WriteRelays []string `yaml:"write_relays"`
	// QueryEventsTimeoutSeconds is the number of seconds to hold open a query
	// against an upstream relay.
	QueryEventsTimeoutSeconds int `yaml:"query_events_timeout_seconds"`
	// AuthDeadlineSeconds is the number of seconds a client must respond to
	// the NIP-42 auth challenge within before the connection is closed.
	AuthDeadlineSeconds int `yaml:"auth_deadline_seconds"`
}

func LoadConfig(path string) (Config, error) {
	var c Config

	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&c); err != nil {
		return c, err
	}

	c.setDefaults()

	return c, nil
}

func (c *Config) setDefaults() {
	if c.Port == 0 {
		c.Port = defaultPort
	}
	if c.Host == "" {
		c.Host = defaultHost
	}
	if c.LogLevel == "" {
		c.LogLevel = "error"
	}
	if c.QueryEventsTimeoutSeconds == 0 {
		c.QueryEventsTimeoutSeconds = defaultQueryEventsTimeoutSeconds
	}
	if c.AuthDeadlineSeconds == 0 {
		c.AuthDeadlineSeconds = defaultAuthDeadlineSeconds
	}
}

func (c Config) PubkeyIsAllowed(pk string) bool {
	if pk == "" {
		return false
	}

	npub, err := nip19.EncodePublicKey(pk)
	if err != nil {
		return false
	}

	for _, allowedNpub := range c.AllowedNpubs {
		if strings.EqualFold(allowedNpub, npub) {
			return true
		}
	}

	return false
}
