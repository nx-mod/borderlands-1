# borderlands-1

NEX game server for **Borderlands: Game of the Year Edition** (Nintendo Switch, `010064800F66A000`), built on the NextendoNetwork [nextendo-nex](https://github.com/NextendoNetwork/nextendo-nex) core. Source only — no binaries, no certs, no game assets. Not affiliated with Gearbox, 2K or Nintendo.

Work in progress: the server refuses to start until the game's NEX access key is known. See [NOTES.md](NOTES.md).

## Build

Clone this repo and `nextendo-nex` side by side, then:

    go build -o server.exe .

See `example.env` for configuration.

## Credits

- **[Nextendo Network](https://nextendo.network)** — the NEX core, gates, dashboard and server pattern this server follows (template: minecraft / luigis-mansion-3).
- **[exlaunch](https://github.com/shadowninja108/exlaunch)** by **Shadow** — the in-game instrumentation used to map the game's online calls (`bl1-hack`).
- **[kinnay/NintendoClients](https://github.com/kinnay/NintendoClients)** and its [wiki](https://github.com/kinnay/NintendoClients/wiki) — NEX protocol method ids and parameters, Switch error modules.
- **[bl-sdk/unrealsdk](https://github.com/bl-sdk/unrealsdk)** (LGPL-3.0) — Borderlands 1 Enhanced struct layouts, the PC base of this port.
- **[Pretendo Network](https://pretendo.network)** — NEX documentation ([developer docs](https://developer.pretendo.network/overview/nex)).

References were read and reimplemented; no code was copied.
