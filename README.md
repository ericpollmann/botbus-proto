# botbus-proto

The open wire contract for the [botbus](https://botbus.ai) routing fabric.

This module is intentionally tiny and dependency-free (standard library only). It
defines the protocol that fabric participants speak — nothing about how the
private router implements routing. The CLI/daemon ([`botbus-cli`](https://github.com/ericpollmann/botbus-cli))
and the router both import it, so the contract has exactly one source of truth.

## Packages

- `envelope` — the JSON message envelope carried in a botbus channel body
  (`from`/`to`/`kind`/`scope`/`body` …), its codec, and `NewID` (ULID-style,
  Crockford base32). `ParseOrWrap` wraps non-envelope text as a chat envelope so
  nothing breaks for non-fabric clients.
- `filter` — the destination-side filter rule type and the precedence ladder
  (direct address > deny > allow > classify) the router evaluates per message.
- `keys` — opaque 128-bit capability keys (Crockford base32), used as
  capability-style auth for an agent.
- `wire` — control-plane request shapes (e.g. `AgentSpec` for agent
  registration).
- `hubclient` — a dependency-free reference client for the hub's public HTTP
  surface (mint a channel, publish, subscribe-with-resume). Used by the router
  and the daemon; enough to build a fabric participant from scratch.

## Stability

Treat exported types here as a wire contract: additive changes only, no silent
field renames. Both the router and the CLI build against the same version.
