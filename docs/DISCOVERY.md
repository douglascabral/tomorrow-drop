# Discovery and connectivity

Tasks 017 to 023 (block 03). Implementation in `internal/discovery` and `internal/transport/dial.go`. Task 024 (tests on real Wi-Fi and enterprise networks) is still open: it needs physical networks and is not covered by the automated suite.

## 1. Datagram format (task 017)

UDP datagrams, UTF-8 JSON, at most `limits.MaxDiscoveryDatagram` (1 KiB). Port: `discovery.DefaultPort` (47520).

```json
{"v":1,"type":"DISCOVER","nonce":"0123456789abcdef"}
{"v":1,"type":"ANNOUNCE","nonce":"0123456789abcdef","name":"living room","port":7000}
```

- `v`: must equal `1`.
- `type`: `DISCOVER` or `ANNOUNCE`. Any other value is rejected.
- `nonce`: 8 random bytes in hexadecimal (16 characters), generated with `crypto/rand`. `ANNOUNCE` echoes the nonce of the query it answers.
- `name`: friendly name (1 to 64 characters, no control characters). Only in `ANNOUNCE`. It is the device name and its system, for example `iPad de Douglas · iPadOS 26.1` or `samsung SM-S911B · Android 15`; on desktop it is the computer name. The device name is the one its owner set in the system, so it may contain the owner's first name.
- `port`: TCP port of the TLS listener (1 to 65535). Only in `ANNOUNCE`.

Validation is strict: unknown fields, trailing data, oversized datagrams and fields that do not belong to the type are rejected. Nothing else is carried: no key, fingerprint, IP address or MAC address. The name may include the device name set by its owner, as described above. The sender's address comes from the UDP packet itself.

## 2. Listener (task 018)

- Runs while the application is available. Stopping it is done by cancelling its context.
- Answers only well-formed `DISCOVER` datagrams. Malformed input gets no reply, so the listener does not amplify traffic.
- Answers are unicast to the querier.
- Per-origin rate limit: 4 answers per 10 seconds per source address (`RateLimiter`). The limiter tracks at most 1024 origins, evicting the least recently active one.
- Random delay before each answer: uniform in [0, 200 ms), from `crypto/rand`, so answers are spread out.

## 3. Scanner (tasks 018–019)

- Sends `DISCOVER` to each target. Targets are built by `BroadcastTargets`: the directed broadcast of every IPv4 subnet of the local interfaces, plus `255.255.255.255`. Loopback and IPv6 interfaces are skipped; IPv6 discovery is not part of this block.
- Accepts an `ANNOUNCE` only if its nonce matches a query sent in the last 10 seconds. Unsolicited or stale answers are discarded.
- A query with no targets, or one where every send fails, returns `error.no_route`.

## 4. When queries run (tasks 019–021, PRD §5.2)

`discovery.Schedule` and `discovery.Controller` implement the rules:

| Event | Behavior |
|---|---|
| Application becomes visible | Query immediately |
| Visible and not suspended | Periodic query every 30 seconds |
| Refresh clicked | Query immediately, restart the 30-second counter |
| Refresh within 2 seconds of the last query | Rejected with `error.rate_limited` |
| Hidden (minimized or background) | No queries |
| Transfer in progress | `SetSuspended(true)`: no periodic queries until it ends |

The counter restarts after every query, including Refresh.

## 5. Device list (task 022)

`discovery.Registry` is an in-memory list. It is not persisted (PRD §21).

- Entries are keyed by source address and announced port. An answer from a known device updates its name and last-seen time.
- Entries expire 90 seconds after their last answer.
- The list holds at most `limits.MaxDiscoveredDevices` (256) entries. When it is full after pruning, new devices are rejected with `error.limit_exceeded`.
- `Trusted` is set only through `MarkTrusted`, after an authenticated connection to that address shows the device is trusted. An announcement never marks a device as trusted.

## 6. Manual connection (task 023)

- `discovery.ParseEndpoint` accepts `IPv4:port` and `[IPv6]:port`. Host names are rejected, so no name resolution happens. Port 0, unspecified, multicast and broadcast addresses are rejected.
- `transport.Dial` opens TCP and then runs the same TLS handshake as the automatic flow. No key is pinned, and the state is derived from the trust list afterwards (PRD §5.5: the IP address is never a trusted identity).

## 7. Open item: task 024

Automated tests cover the logic: message validation, rate limits, schedule, nonce matching, subnet computation for multiple interfaces, and loopback exchanges. The real-network matrix (same VLAN, blocked broadcast, client isolation, multiple interfaces, no route) must be run on physical networks and recorded here before task 024 is marked complete.
