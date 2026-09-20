# borderlands-1

NEX game server for **Borderlands: Game of the Year Edition** (Nintendo Switch, `010064800F66A000`), built on the NextendoNetwork [nextendo-nex](https://github.com/NextendoNetwork/nextendo-nex) core. Source only — no binaries, no certs, no game assets. Not affiliated with Gearbox, 2K or Nintendo.

## Status

Tested on **one console only** (a CFW Switch, 2026-09-13). There has been no two-sided test: no second player has ever joined.

- **Seen working live:** the game's server id (`0x241c6800`) and NEX access key (`018165a5`, the default here), login through the auth and secure servers, and creating a lobby (`CreateMatchmakeSessionWithParam`, `OpenParticipation`, `UpdateMatchmakeSessionAttribute`).
- **Not tested:** anything a second player does: browsing, joining, auto-matchmaking, leaving, or hosting for a friend. The nextendo-nex core has handlers for those calls (written for other games), but nobody has run them against Borderlands, so they may not fit. Any call the server does not handle is logged as `UNHANDLED`.
- The game showed a BCAT error (2122-0002) at start; that is Nintendo's delivery service, not this server.

See [NOTES.md](NOTES.md) for the calls observed and how the key and server id were found.

## Build

Clone this repo and `nextendo-nex` side by side, then:

    go build -o server.exe .

See `example.env` for configuration.

## Credits

- **[Nextendo Network](https://nextendo.network)** — the NEX core, gates, dashboard and server pattern this server follows (template: minecraft / luigis-mansion-3).
- **[exlaunch](https://github.com/shadowninja108/exlaunch)** by **Shadow** — the in-game instrumentation used to map the game's online calls (`bl1-hack`, an exlaunch logging module that is not part of this repository).
- **[kinnay/NintendoClients](https://github.com/kinnay/NintendoClients)** and its [wiki](https://github.com/kinnay/NintendoClients/wiki) — NEX protocol method ids and parameters, Switch error modules.
- **[bl-sdk/unrealsdk](https://github.com/bl-sdk/unrealsdk)** (LGPL-3.0) — Borderlands 1 Enhanced struct layouts, the PC base of this port.
- **[Pretendo Network](https://pretendo.network)** — NEX documentation ([developer docs](https://developer.pretendo.network/overview/nex)).

References were read and reimplemented; no code was copied.
