package main

import (
	"os"
	"strings"

	"github.com/nbd-wtf/go-nostr/nip19"
	"gopkg.in/yaml.v3"
)

const (
	defaultPort = 8001
	defaultHost = "0.0.0.0"
)

// Config is the relay configuration.
type Config struct {
	// Port is the listen port.
	Port int `yaml:"port"`
	// Host is the listen host.
	Host string `yaml:"host"`
	// AllowedNpubs is a list of npubs the relay will accept events from.
	AllowedNpubs []string `yaml:"allowed_npubs"`
	// ReadRelays is a list of relay URLs events will be read from.
	ReadRelays []string `yaml:"read_relays"`
	// WriteRelays is a list of relay URLs new events will be written to.
	WriteRelays     []string `yaml:"write_relays"`
	NIP42ServiceURL string   `yaml:"nip42_service_url"`
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
