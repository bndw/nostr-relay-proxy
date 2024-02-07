# nostr-relay-proxy

> ⚠️ WARNING: Experimental software. Use at your own risk.

A fan-in / fan-out nostr relay proxy. Instead of connecting your
nostr client to many relays, connect it to a single nostr-relay-proxy.
Let the proxy take care of deduplicating events and shipping to your client
over a single websocket connection.

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
