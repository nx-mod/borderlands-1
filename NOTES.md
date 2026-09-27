# Borderlands GOTY (Switch) — Nextendo NEX server: notes

Keep this file updated as you go: it is the map for this server.

- Game: Borderlands: Game of the Year Edition, title `010064800F66A000`
- Binary: `sd:/atmosphere/contents/010064800F66A000/exefs/main` (an Atmosphere exefs
  override: may be a patched main, not stock — confirm before trusting offsets).
  Copy + segments in `bl1-hack/capture/nso/` (not in this repository).
  - SHA-256 `2051DA55540AEC785619A717B07EC01D7B9616F6789F24D7AA8DFBCCAE950305`,
    NSO build id `1c37c3673e0e4e7aadf7860078d55f63`
  - segments: text @0x0 (31.4 MB), rodata @0x01DFA000, data @0x03178000
  - Ghidra: `bl1-hack/tools/ghidra_scripts/AddNsoSegments <dir> 0x01DFA000 0x03178000`
- Status (2026-09-13): **game server ID 0x241c6800** and **access key 018165a5** both
  confirmed live; sni-router routes the host to this server (auth :8456, secure :60012).

## Identified

- **Game server ID `0x241c6800`**, seen live: the console opened a NEX WebSocket to
  `g241c6800-lp1.s.n.srv.nintendo.net` (`Sec-Websocket-Protocol: NEX`) right after its BaaS
  login, when "searching lobbies". The same u32 is in rodata at VA 0x1FBD6A7. With no
  sni-router route it fell through to baas-proxy → real Nextendo, which answered a plain
  200 (no upgrade) → "server communication error".
- **Access key `018165a5` — confirmed.** bl1-hack logged the game passing that literal to
  `BackEndServices::SetSandboxAccessKey` and `StreamManager::SetSandboxAccessKey` (x1 =
  the C string in rodata, right before the `UOnlineSubsystemSwitch` exec strings; x0 = a
  16-byte nn::nex::String whose second word holds the same 8 ASCII bytes).
- Login flow seen: `nsd resolve 'g241c6800-%.s.n.srv.nintendo.net'` →
  `g241c6800-lp1…`, getaddrinfo → <router IP>, non-blocking connect to :443 (from
  main+0x3660), retried three times while no server answered.
- With no server, starting a public lobby crashed the game: 2124-0400, SDK abort on
  `QueuedThread2`. Most likely the game's own handling of a dead NEX login; recheck with
  the server up.
- bl1-hack log file: while the game runs, sys-ftpd shows `log.txt` locked at 0 bytes; close
  the game before pulling it.

## First live session (2026-09-13, CFW Switch)

Login and hosting work up to the lobby attributes:

```
Auth    TicketGranting ValidateAndRequestTicketWithParam (0xA/6)  username = console NSA id
        -> nextendo-account /api/nsa -> Nextendo PID (1800000003 on the local stack)
Secure  SecureConnection Register (0xB/1)
        MatchmakeExtension CreateMatchmakeSessionWithParam (0x6D/38) -> gid 1
        NATTraversal ReportNATProperties (0x3/5)
        MatchmakeExtension OpenParticipation (0x6D/2)
        MatchmakeExtension UpdateMatchmakeSessionPart (0x6D/44)
        MatchmakeExtension UpdateMatchmakeSessionAttribute (0x6D/12)  <- was NotImplemented
```

- **2306-0103** "starting game lobby" / switching friends-only → online = the core refusing
  0x6D/12 with Core::NotImplemented. 12 = `UpdateMatchmakeSessionAttribute(u32 gid,
  List<u32> attribs)`, no return value (NintendoClients wiki); Borderlands sends 6 attributes.
  Handled in main.go by applying each through the core's ModifyCurrentGameAttribute.
  The lobby still switched to public in the menu: creation and OpenParticipation succeeded.
- **2122-0002 "unable to load data"** at game start: module 2122 is **BCAT** (background
  data delivery), not NEX. The console's `bcat-list …/nx_data_010064800f66a000` and
  `bcat-topics …/010064800f66a000` go through baas-proxy to the real Nintendo CDN and come
  back 304 Not Modified. Not this server's job; revisit if it blocks anything.
- Joiners will need Browse (0x6D/4, /5, /42) or AutoMatchmake (/3, /15, /33, /40) — watch
  for them as NotImplemented in the log.

## References

- kinnay/NintendoClients wiki — NEX protocol method ids and parameters
  (https://github.com/kinnay/NintendoClients/wiki/Matchmake-Extension-Protocol), Switch error modules.
- bl-sdk/unrealsdk (LGPL-3.0) — Borderlands 1 Enhanced (the PC base of this port) struct
  layouts and globals: `src/unrealsdk/game/bl1e/` (UObject/UField 104 bytes, UStruct
  SuperField +120 / Children +128 / PropertyLink +176, UClass ClassDefaultObject +468).
  Offsets are PC x64; confirm on the aarch64 Switch build before use.

Local shallow clones (not committed): `unrealsdk`,
`NintendoClients`, `NintendoClients.wiki` (protocol pages: Matchmake-Extension-Protocol.md,
Secure-Protocol.md, …). `bl1-hack/refs/unrealsdk` keeps the few BL1E files read first.

## Online stack: Nintendo NEX

Found in the binary's symbols (Unreal Engine 3, `WillowGame`, Gearbox framework):

- `nn::nex` is linked into main: `RendezVous::Login/SilentLogin`, `BackEndServices`,
  `StreamManager`, `HttpClient`, `GameStream`, `DataCode64`.
- Unreal side: `OnlineSubsystemSwitch`, `FNEXClient::{Create,Join,Leave,Update,Destroy,Browse}MatchmakeSession`,
  `UpdateMatchmakeSessionBuffer`, `AddOnSessionUnregisteredCallback`.
- Access key: set through `nn::nex::BackEndServices::SetSandboxAccessKey(String const&)`
  and `nn::nex::StreamManager::SetSandboxAccessKey` (static `s_szSandboxAccessKey`).
  Not present as a plain 8-hex string in rodata.
- Name resolution: `nn::nsd::ResolveEx(Fqdn*, Fqdn const&)` — expected
  `g<gameServerId>-lp1.s.n.srv.nintendo.net`, which sni-router routes by the `g<id>` prefix.
- Also imports nn::socket (Connect/SendTo/RecvFrom/GetAddrInfo…), nn::ssl, nn::nifm,
  nn::friends (UserPresence, DeclareOpen/CloseOnlinePlaySession, friend invitations).
- Also present: Gearbox **Spark** (`SparkPB::SparkUpdate`, `GetSparkGuidFromSave`) — a
  separate Gearbox service (SHiFT/telemetry), not needed for co-op; watch what it contacts.

## Still needed

1. NEX access key → `BL1_ACCESS_KEY` (bl1-hack logs `SetSandboxAccessKey`).
2. Game server ID → sni-router route + hosts (bl1-hack logs `nn::nsd::ResolveEx`).
3. NEX / Pia versions → `BL1_NEX_VERSION`, matchmaking/station handling.
4. Which protocols/methods the game calls after login. Every RMC nextendo-nex does
   not implement falls through to the endpoint fallback (empty success + a log
   line) and is now also recorded structurally: `unhandled.go` counts each
   proto/method with a body sample, surfaced on the dashboard under
   `/api/stats` → `unhandled` (most-called first). Watch that list against a live
   console to see exactly what to implement next; `UpdateMatchmakeSessionAttribute`
   (0x6D/12) was the first such gap and is handled in `main.go`.

## Ports (local stack)

| what | port |
|---|---|
| auth (behind sni-router) | 443 → local AUTH_PORT (proposed 8456) |
| secure | 60012 |
| dashboard | 8094 |

## Method

Instrument first: `bl1-hack` (an exlaunch module, not part of this repository) logs name
resolution, sockets and the access key to `sd:/config/bl1-hack/log.txt`.
