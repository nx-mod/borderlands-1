# borderlands-1

**A new game server implementation by nx-mod** for the Nextendo Network.

NEX game server for **Borderlands: Game of the Year Edition** (Nintendo Switch, `010064800F66A000`), built on the NextendoNetwork [nextendo-nex](https://github.com/NextendoNetwork/nextendo-nex) core. Source only — no binaries, no certs, no game assets. Not affiliated with Gearbox, 2K or Nintendo.

## Status

Tested on **one console only** (a CFW Switch, 2026-09-13). There has been no two-sided test: no second player has ever joined.

- **Seen working live:** the game's server id (`0x241c6800`) and NEX access key (`018165a5`, the default here), login through the auth and secure servers, and creating a lobby (`CreateMatchmakeSessionWithParam`, `OpenParticipation`, `UpdateMatchmakeSessionAttribute`).
- **Not tested:** anything a second player does: browsing, joining, auto-matchmaking, leaving, or hosting for a friend. The nextendo-nex core has handlers for those calls (written for other games), but nobody has run them against Borderlands, so they may not fit. Any call the server does not handle is logged as `UNHANDLED`.
- The game showed a BCAT error (2122-0002) at start; that is Nintendo's delivery service, not this server.

See [NOTES.md](NOTES.md) for the calls observed and how the key and server id were found.

## Requirements

This server runs behind the rest of the Nextendo stack. It needs nothing cloned next to it: `go build` fetches the NEX core ([nextendo-nex](https://github.com/NextendoNetwork/nextendo-nex), a Go module) by itself.

| component | needed? | what it must provide |
|---|---|---|
| **sni-router** | required | A route sending `g241c6800-lp1.s.n.srv.nintendo.net` to `BACKEND_BL1` (default `127.0.0.1:8456`). The route is not in sni-router's `main` yet: it is the `feat/borderlands-1` branch of [nx-mod/sni-router](https://github.com/nx-mod/sni-router), one commit on top of `main`. |
| **nextendo-account** | required with `NEXTENDO_REQUIRE_ACCOUNT=1` | `GET /api/nsa` (a console's NSA id to a Nextendo account) and `POST /internal/online-check`, with `X-Internal-Key`. With the gate on, a login whose NSA id cannot be resolved is refused. |
| **nextendo-dashboard** | optional | A `bl1` source polling `/api/stats` on port 8094 (`DASH_BL1_URL`, `DASH_BL1_TOKEN`): the `feat/borderlands-1-stats` branch of [nx-mod/nextendo-dashboard](https://github.com/nx-mod/nextendo-dashboard). Without it the server works but is not on the shared dashboard. |
| **DNS** | required | `g241c6800-lp1.s.n.srv.nintendo.net` must resolve to the machine running sni-router, and never to Nintendo. On a console that is an Atmosphere hosts entry; the standard Nextendo hosts file already sends `*.srv.nintendo.net` to the stack. |
| **TLS certificate** | required | A certificate and key for the game's auth host, from a CA your clients trust (`CERT_FILE`, `KEY_FILE`). Yours to provide; none is shipped. |

The secure server is not behind the router: the game connects to `NEXTENDO_HOST:60012` directly, so `NEXTENDO_HOST` must be the address players can reach.

## Install

1. Build: `go build -o server.exe .` (Go 1.23 or later).
2. Copy `example.env` to `.env` and set the secrets. The server does **not** read `.env` itself: export the variables into its environment (a launcher or service file), and set `NEXTENDO_SECRET` or `NEXTENDO_SECRET_FILE`, `NEXTENDO_INTERNAL_KEY`, `NEXTENDO_SECURE_PASSWORD` and `DASH_TOKEN`.
3. Put `cert.pem` and `key.pem` next to the binary, or point `CERT_FILE` and `KEY_FILE` at them.
4. Start it. Auth listens on `AUTH_PORT` (`8456` behind sni-router), the secure server on `60012`, the dashboard on `8094`.

| setting | default | meaning |
|---|---|---|
| `BL1_ACCESS_KEY` | `018165a5` | the game's NEX access key, confirmed live |
| `BL1_NEX_VERSION` | `40000` | the value in use; it has not been confirmed against the game |

Game: Borderlands: Game of the Year Edition, title `010064800F66A000`, game server id `0x241c6800`.

## Known limits

- **One console only.** Nothing that a second player does has been run: browsing, joining, auto-matchmaking, leaving, or hosting for a friend. The nextendo-nex core has handlers for those calls (written for other games) but they have not been checked against Borderlands.
- **Unhandled calls** are logged as `[BL1 Secure] UNHANDLED ...` with the full request and answered with an empty success, so the game may carry on as if the call had worked. The secure server registers the core's matchmaking, matchmaking-extension, NAT traversal, ranking and utility handlers.
- **NEX version** (`BL1_NEX_VERSION`) is 4.0.0 as a working value; the game's real build has not been confirmed.
- **BCAT.** The game shows error 2122-0002 at start: that is Nintendo's delivery service, not this server.
- **Crash.** Starting a public lobby with no server up crashed the game once (2124-0400); not re-checked with the server up.
- No persistence: games and connections live in memory.

## To do

- Run a second player through browse, join and leave, and fix what the core's handlers get wrong for this game.
- Confirm the NEX version the game was built with.
- Add tests (there are none yet).

## Credits

- **[Nextendo Network](https://nextendo.network)** — the NEX core, gates, dashboard and server pattern this server follows (template: minecraft / luigis-mansion-3).
- **[exlaunch](https://github.com/shadowninja108/exlaunch)** by **Shadow** — the in-game instrumentation used to map the game's online calls (`bl1-hack`, an exlaunch logging module that is not part of this repository).
- **[kinnay/NintendoClients](https://github.com/kinnay/NintendoClients)** and its [wiki](https://github.com/kinnay/NintendoClients/wiki) — NEX protocol method ids and parameters, Switch error modules.
- **[bl-sdk/unrealsdk](https://github.com/bl-sdk/unrealsdk)** (LGPL-3.0) — Borderlands 1 Enhanced struct layouts, the PC base of this port.
- **[Pretendo Network](https://pretendo.network)** — NEX documentation ([developer docs](https://developer.pretendo.network/overview/nex)).

References were read and reimplemented; no code was copied.

## Credits

Built by nx-mod for the **Nextendo Network**, on the work of the Nextendo Network team — https://nextendo.network. Nextendo is awesome.
