// Command borderlands-1 runs the Borderlands: Game of the Year Edition (Switch,
// 010064800F66A000) online servers on the Nextendo NEX stack.
//
// Borderlands' online layer is Nintendo NEX: nn::nex is linked into the game's main
// (RendezVous login, MatchmakeExtension sessions driven by Unreal's FNEXClient /
// OnlineSubsystemSwitch). So this is a regular NEX game server:
//   - auth   (:443 behind sni-router)  TicketGranting — LoginEx issues the Kerberos ticket.
//   - secure (:60012)                  SecureConnection + matchmaking + NAT traversal + ranking + utility.
//
// Known (see NOTES.md): game server 0x241c6800 (g241c6800-lp1.s.n.srv.nintendo.net, routed
// by sni-router) and access key 018165a5. Still unknown: the exact NEX/Pia versions.
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"os"
	"strconv"
	"strings"
	"time"

	nex "github.com/NextendoNetwork/nextendo-nex"
)

const (
	securePID     = 2
	sessionKeyLen = 32
	bl1AppID      = "010064800f66a000"
)

var (
	// accessKey is Borderlands' NEX access key: the literal the game passes to
	// nn::nex::BackEndServices::SetSandboxAccessKey (logged by bl1-hack on a CFW Switch,
	// 2026-09-13). It sits in rodata right before the UOnlineSubsystemSwitch strings.
	accessKey  = envOr("BL1_ACCESS_KEY", "018165a5")
	nexVersion = envOrInt("BL1_NEX_VERSION", 40000)

	nextendoHost   = envOr("NEXTENDO_HOST", "127.0.0.1")
	authPort       = envOrInt("AUTH_PORT", 443)
	securePort     = envOrInt("SECURE_PORT", 60012) // LM3=60009 ACNH=60010/60011 -> BL1=60012
	securePassword = envOr("NEXTENDO_SECURE_PASSWORD", "")
	certFile       = envOr("CERT_FILE", "cert.pem")
	keyFile        = envOr("KEY_FILE", "key.pem")

	nextendoSecret = loadNextendoSecret()
	requireAccount = os.Getenv("NEXTENDO_REQUIRE_ACCOUNT") == "1"
)

func main() {
	// Refuse to start with a guessed key: PRUDP signatures would fail silently and only
	// confuse captures.
	if len(accessKey) != 8 {
		fmt.Println("[BL1] FATAL: BL1_ACCESS_KEY is not set (8 hex chars).")
		fmt.Println("[BL1] Recover it with bl1-hack (SetSandboxAccessKey is logged), then set BL1_ACCESS_KEY.")
		os.Exit(1)
	}
	// No public default Kerberos password: anyone knowing it could forge tickets.
	if securePassword == "" {
		fmt.Println("[BL1] FATAL: NEXTENDO_SECURE_PASSWORD is not set.")
		os.Exit(1)
	}

	settings := nex.NewSwitchSettings(accessKey, nexVersion)

	// --- Auth server ---
	secureURL := nex.NewStationURL("prudps")
	secureURL.Set("address", nextendoHost)
	secureURL.SetInt("port", securePort)
	secureURL.SetInt("CID", 1)
	secureURL.SetInt("PID", securePID)
	secureURL.SetInt("sid", 1)
	secureURL.SetInt("stream", 10)
	secureURL.SetInt("type", 2)

	authEndpoint := nex.NewEndpoint(settings)
	authCfg := &nex.AuthConfig{
		Settings:         settings,
		SecurePID:        securePID,
		SecurePassword:   securePassword,
		SecureStationURL: secureURL,
		ServerName:       "Nextendo",
		SessionKeyLength: sessionKeyLen,
		ResolveUser:      resolveUser,
	}
	authEndpoint.Register(nex.ProtocolTicketGranting, authCfg.Handler())
	authEndpoint.RegisterFallback(func(c *nex.Connection, req *nex.RMCMessage) *nex.RMCMessage {
		// Unknown auth calls are logged in full: this server is still being mapped.
		fmt.Printf("[BL1 Auth] UNHANDLED pid=%d proto=%#x method=%d call=%d body=%x\n", c.PID, req.Protocol, req.Method, req.CallID, req.Body)
		return nex.NewRMCSuccess(c.Settings, req.Protocol, req.Method, req.CallID, nil)
	})
	authEndpoint.OnRMC = logRMC("Auth")
	authServer := nex.NewServer(authEndpoint)

	// --- Secure server ---
	secureEndpoint := nex.NewEndpoint(settings)
	secureEndpoint.SetSecureAccount(securePassword, securePID)

	mm := nex.NewMatchmaking()
	secureEndpoint.Register(nex.ProtocolSecureConnection, nex.SecureConnectionHandler())
	secureEndpoint.Register(nex.ProtocolMatchmakeExtension, mm.ExtensionHandler())
	secureEndpoint.Register(nex.ProtocolMatchMaking, mm.MatchMakingHandler())
	secureEndpoint.Register(nex.ProtocolMatchMakingExt, mm.MatchMakingExtHandler())
	secureEndpoint.Register(nex.ProtocolNATTraversal, nex.NATTraversalHandler())
	secureEndpoint.Register(nex.ProtocolRanking, nex.RankingHandler())
	secureEndpoint.Register(nex.ProtocolUtility, nex.UtilityHandler())
	secureEndpoint.RegisterFallback(func(c *nex.Connection, req *nex.RMCMessage) *nex.RMCMessage {
		fmt.Printf("[BL1 Secure] UNHANDLED pid=%d proto=%#x method=%d call=%d body=%x\n", c.PID, req.Protocol, req.Method, req.CallID, req.Body)
		return nex.NewRMCSuccess(c.Settings, req.Protocol, req.Method, req.CallID, nil)
	})
	logSecure := logRMC("Secure")
	secureEndpoint.OnRMC = func(c *nex.Connection, req *nex.RMCMessage) {
		logSecure(c, req)
		noteRMC(c, req)
	}
	secureEndpoint.OnConnect = func(c *nex.Connection) {
		fmt.Printf("[BL1 Secure] connected pid=%d id=%d addr=%s\n", c.PID, c.ID, c.RemoteAddr)
	}
	// A crashed client never unregisters its gathering: drop it with the connection.
	secureEndpoint.OnDisconnect = func(c *nex.Connection) {
		mm.RemovePlayer(c.PID)
	}
	secureServer := nex.NewServer(secureEndpoint)

	secureEndpoint.StartReaper()
	go startDashboard(secureEndpoint, mm)

	// Behind sni-router the PROXY header carries the console's real address; the secure
	// connection arrives direct, and the two must agree.
	proxyProto := os.Getenv("NEXTENDO_PROXY_PROTOCOL") == "1"
	go func() {
		fmt.Printf("[BL1 Auth] listening WSS :%d (proxyProto=%v, secure URL -> %s)\n", authPort, proxyProto, secureURL.String())
		var err error
		if proxyProto {
			err = authServer.ListenSecureProxy(authPort, certFile, keyFile)
		} else {
			err = authServer.ListenSecure(authPort, certFile, keyFile)
		}
		if err != nil {
			fmt.Printf("[BL1 Auth] stopped: %v\n", err)
		}
	}()

	fmt.Printf("[BL1 Secure] listening WSS :%d (title %s, nex %d)\n", securePort, bl1AppID, nexVersion)
	if err := secureServer.ListenSecure(securePort, certFile, keyFile); err != nil {
		fmt.Printf("[BL1 Secure] stopped: %v\n", err)
	}
}

// resolveUser maps a LoginEx username to an account, as the other Nextendo NEX servers do:
// a signed nx2 token or a proven emulator PID keeps its account PID, a console NSA id is
// resolved through nextendo-account, anything else is anonymous (unless an account is required).
func resolveUser(username string, extraData []byte) (uint64, []byte, bool) {
	// The source key encrypts the client ticket; the console expects 32 bytes.
	sk := sha256.Sum256([]byte("nextendo-src:" + username))
	sourceKey := sk[:]

	if pid, ok := nextendoPIDFromToken(username); ok {
		if allow, reason := nextendoOnlineCheck(pid, "ryujinx"); !allow {
			fmt.Printf("[BL1 Auth] pid=%d online refused (%s)\n", pid, reason)
			return 0, nil, false
		}
		return pid, sourceKey, true
	}

	if n, err := strconv.ParseUint(username, 10, 64); err == nil && n >= 1800000000 {
		provenPID, proven := uint64(0), false
		if tok, ok := nex.NexTokenFromLoginExtraData(extraData); ok {
			provenPID, proven = nextendoPIDFromToken(tok)
		}
		if n < 1810000000 {
			switch {
			case proven && provenPID == n:
				fmt.Printf("[BL1 Auth][bind] pid=%d OK: nx2 proves the PID\n", n)
			case proven && provenPID != n:
				fmt.Printf("[BL1 Auth][bind] pid=%d IMPERSONATION: nx2 proves %d\n", n, provenPID)
			default:
				fmt.Printf("[BL1 Auth][bind] pid=%d NO PROOF: no nx2 in extraData\n", n)
			}
			if requireSignedToken() && !(proven && provenPID == n) {
				fmt.Printf("[BL1 Auth] pid=%d refused: identity not proven\n", n)
				return 0, nil, false
			}
		}
		pid, kind := n, "ryujinx"
		if n >= 1810000000 {
			kind = "switch"
			rp, st := resolveNSAtoPID(n)
			switch st {
			case nsaOK:
				pid = rp
				fmt.Printf("[BL1 Auth] NSA %d -> account pid=%d\n", n, pid)
			case nsaUnknown, nsaUnreachable:
				fmt.Printf("[BL1 Auth] NSA %d refused (%v)\n", n, st)
				return 0, nil, false
			}
		}
		if allow, reason := nextendoOnlineCheck(pid, kind); !allow {
			fmt.Printf("[BL1 Auth] pid=%d online refused (%s)\n", pid, reason)
			return 0, nil, false
		}
		return pid, sourceKey, true
	}

	if requireAccount {
		fmt.Printf("[BL1 Auth] anonymous refused: %q\n", username)
		return 0, nil, false
	}
	return anonymousPID(username), sourceKey, true
}

// revokedNexPayloads lists leaked nex_token payloads ("pid.username.expiry") rejected despite
// a valid HMAC. Keep in sync with nextendo-account and the sibling servers.
var revokedNexPayloads = map[string]bool{
	"1800000006.Kazuu.1787343209": true, // leaked in the 1.6.5-win release
}

func nextendoPIDFromToken(s string) (uint64, bool) {
	if len(nextendoSecret) == 0 || !strings.HasPrefix(s, "nx2.") {
		return 0, false
	}
	parts := strings.Split(s[len("nx2."):], ".")
	if len(parts) != 2 {
		return 0, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, false
	}
	mac := hmac.New(sha256.New, nextendoSecret)
	mac.Write([]byte("nex:" + string(raw)))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(parts[1])) {
		return 0, false
	}
	if revokedNexPayloads[string(raw)] {
		return 0, false
	}
	f := strings.SplitN(string(raw), ".", 3)
	if len(f) != 3 {
		return 0, false
	}
	pid, err := strconv.ParseUint(f[0], 10, 64)
	if err != nil {
		return 0, false
	}
	if exp, err := strconv.ParseInt(f[2], 10, 64); err != nil || time.Now().Unix() > exp {
		return 0, false
	}
	return pid, true
}

func loadNextendoSecret() []byte {
	if v := os.Getenv("NEXTENDO_SECRET"); v != "" {
		return []byte(v)
	}
	path := envOr("NEXTENDO_SECRET_FILE", "nextendo_secret.key")
	if b, err := os.ReadFile(path); err == nil {
		if dec, derr := hex.DecodeString(strings.TrimSpace(string(b))); derr == nil && len(dec) >= 16 {
			return dec
		}
	}
	return nil
}

func anonymousPID(username string) uint64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(username))
	return 1800000000 + uint64(h.Sum32()%100000000)
}

func logRMC(tag string) func(*nex.Connection, *nex.RMCMessage) {
	return func(c *nex.Connection, req *nex.RMCMessage) {
		fmt.Printf("[BL1 %s] pid=%d proto=%#x method=%d call=%d\n", tag, c.PID, req.Protocol, req.Method, req.CallID)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envOrInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func requireSignedToken() bool {
	v := os.Getenv("NEXTENDO_REQUIRE_SIGNED_TOKEN")
	return v == "1" || v == "true"
}
