# nostr-relay-proxy

> ⚠️ WARNING: Experimental software. Use at your own risk.

A fan-in / fan-out nostr relay proxy. Instead of connecting your
nostr client to many relays, connect it to a single nostr relay proxy and enjoy:

- **Efficient bandwidth usage**: events are deduplicated before being sent to the client.
- **Increased performance**: offloading deduplication in addition to event caching optimizes query performance.
- **Client IP obfuscation**: upstream relays only see the IP of the nostr relay proxy, not the client.

## Features

#### Authentication

Access to your nostr relay proxy may be restricted to a whitelist of allowed users.
There are two layers of authentication employed to restrict access:

1. **Public key allowlist**: A list of `allowed_npubs` in the config file restricts reads and writes to whitelisted users.
2. **[NIP-42](https://github.com/nostr-protocol/nips/blob/master/42.md)**: 
Queries are restricted to authenticated clients only. Unauthenticated connections are proactively closed.

#### Local database

Events from upstream relays are stored locally on disk in a [LMDB](https://en.wikipedia.org/wiki/Lightning_Memory-Mapped_Database).
As queries are received, the proxy will check its local database first and return
matching events if found. This read-through style cache dramatically speeds up query responses.

## Quickstart

1. Build the source code
  ```
  make build
  ```

2. Create a config file
  ```
  cp example.config.yaml config.yaml
  ```

3. Run the nostr-relay-proxy
  ```
  ./bin/nostr-relay-proxy -config ./path/to/config.yaml
  ```
