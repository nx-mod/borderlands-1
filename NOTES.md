# Borderlands GOTY (Switch) — Nextendo NEX server: notes

Keep this file updated as you go: it is the map for this server.

- Game: Borderlands: Game of the Year Edition, title `010064800F66A000`
- Binary: `sd:/atmosphere/contents/010064800F66A000/exefs/main` (an Atmosphere exefs
  override: may be a patched main, not stock — confirm before trusting offsets).
  Copy + segments in `nextendo/bl1-hack/capture/nso/` (gitignored).
  - SHA-256 `2051DA55540AEC785619A717B07EC01D7B9616F6789F24D7AA8DFBCCAE950305`,
    NSO build id `1c37c3673e0e4e7aadf7860078d55f63`
  - segments: text @0x0 (31.4 MB), rodata @0x01DFA000, data @0x03178000
  - Ghidra: `bl1-hack/tools/ghidra_scripts/AddNsoSegments <dir> 0x01DFA000 0x03178000`
- Status (2026-09-13): stack identified; **game server ID 0x241c6800** (live); access key
  candidate **018165a5** (static, unverified).

## Identified

- **Game server ID `0x241c6800`**, seen live: the console opened a NEX WebSocket to
  `g241c6800-lp1.s.n.srv.nintendo.net` (`Sec-Websocket-Protocol: NEX`) right after its BaaS
  login, when "searching lobbies". The same u32 is in rodata at VA 0x1FBD6A7. With no
  sni-router route it fell through to baas-proxy → real Nextendo, which answered a plain
  200 (no upgrade) → "server communication error".
- **Access key candidate `018165a5`**: the only 8-hex string in rodata next to the online
  code, immediately before the `UOnlineSubsystemSwitch` exec strings. Verify with the first
  accepted PRUDP CONNECT, or bl1-hack's `SetSandboxAccessKey` log.
- bl1-hack log file: while the game runs, sys-ftpd shows `log.txt` locked at 0 bytes; close
  the game before pulling it.

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
4. Which protocols/methods the game calls after login (the server logs every unhandled RMC).

## Ports (local stack)

| what | port |
|---|---|
| auth (behind sni-router) | 443 → local AUTH_PORT (proposed 8456) |
| secure | 60012 |
| dashboard | 8094 |

## Method

Instrument first (see `nextendo/diablo-3/NOTES.md` §6): `nextendo/bl1-hack` is an exlaunch
module that logs name resolution, sockets and the access key to
`sd:/config/bl1-hack/log.txt`.
