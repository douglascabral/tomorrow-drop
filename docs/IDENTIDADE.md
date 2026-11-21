# Identity, trust and pairing

This document describes tasks 006 to 016 (block 02). The implementation lives in `internal/identity`, `internal/storage`, `internal/trust` and `internal/transport`.

## 1. Identity

- Each installation has an Ed25519 key pair generated with `crypto/rand` (`identity.Generate`).
- The private key is written to `identity.key` (PEM PKCS#8) inside the directory chosen by the caller, with directory mode `0700` and file mode `0600` on POSIX systems. On Windows, Go's permission bits do not control ACLs; this is a known limitation.
- `identity.Store` operations:
  - `Load`: reads and validates. Missing: `os.ErrNotExist`. Corrupted: `apperr.ErrMalformed`.
  - `Create`: creates only if absent (exclusive creation). Never overwrites.
  - `Replace`: deliberate regeneration, with atomic replacement. Requires explicit user confirmation in the interface, because the old identity stops existing.
  - `LoadOrCreate`: reuses or creates. A corrupted file is never replaced.
- The private key has no public accessor. `String`, `GoString` and `Fingerprint` do not expose it.

## 2. Fingerprint

- `SHA-256` of the 32 bytes of the Ed25519 public key.
- Shown as the 64 hexadecimal characters, in groups of 4 separated by spaces (16 groups). No truncation.

## 3. Trust list

- `trust.Store` keeps, in JSON with atomic writes, the friendly name, public key, fingerprint and date.
- On load, the fingerprint is recomputed from the key; a mismatch indicates tampering and is an error.
- Read failures make `IsTrusted` return `false` (fail closed).
- Revocation: `Remove`. Afterwards, a new connection requires new pairing.
- Names: 1 to 64 characters, no control characters, and not only spaces.
- The only way to insert a key is `trust.CommitPairing`, which requires proven mutual confirmation.

## 4. TLS 1.3 transport

- TLS 1.3 is mandatory. Each side presents a self-signed X.509 certificate with the identity's Ed25519 key (`identity.NewCertificate`). Validity is one year; the identity does not change, so the certificate can be renewed on each run.
- Both sides must present a certificate (`RequireAnyClientCert`).
- Verification: `VerifyPeerCertificate` validates the certificate on every handshake. Go's default chain validation does not apply.
- TLS session resumption is disabled, so no connection escapes verification.
- Pinned key (`pin`): when the client already knows the peer's key, a different key results in `apperr.ErrIdentityChanged`. There is no automatic key update.

## 5. Security state

After the handshake, the state is derived from the trust list:

- trusted key: `trusted`;
- any other key: `awaiting_verification` (provisional connection, encrypted, without trust).

An established TLS connection is never treated as trusted by itself.

## 6. Message policy

`protocol.AllowedIn(state, type)` is enforced in `Session.Send` and `Session.Receive`:

| State | Allowed types |
|---|---|
| `connecting` | `HELLO` |
| `awaiting_verification` | `HELLO`, `PAIR_CONFIRM`, `PAIR_CANCEL` |
| `trusted` | `HELLO`, `TRANSFER_OFFER` |
| others | none |

A disallowed incoming type closes the connection with `ErrUnauthorized`. In a provisional connection, transfers are blocked both when sending and when receiving.

## 7. Version negotiation

- `HELLO` is exchanged inside TLS, so an intermediary cannot alter it.
- The version is the highest common one between the lists (`protocol.Negotiate`). No common version: `ErrVersionMismatch`.

## 8. Pairing

Flow, per PRD §9:

1. Provisional connection established.
2. Each side obtains a `Pairing` from the session. The binding is the TLS exporter (`ExportKeyingMaterial`, label `tomorrow-drop pairing v1`), unique per connection.
3. Each user compares the complete fingerprints on both devices and confirms explicitly (`ConfirmLocal`).
4. `ExchangePairing` exchanges the confirmations. Each confirmation is `HMAC-SHA256(binding, label || role || version || confirmer || other)`.
5. `VerifyRemote` recomputes the peer's expected confirmation and compares it in constant time.
6. `CommitPairing` stores the peer's key only if both the local confirmation and the peer's confirmation were verified.
7. `Session.Promote` marks the session as trusted.

Guarantees covered by tests:

- no confirmation is emitted without `ConfirmLocal`;
- a unilateral confirmation persists nothing;
- reflecting one's own confirmation is rejected;
- a confirmation from another session, another peer or another version is rejected;
- retransmission by an intermediary (MITM) makes verification fail on both sides.

## 9. Limits

- Comparing fingerprints is done by people. The protocol guarantees that the confirmation is valid for the session and the displayed keys, but it does not replace the visual comparison.
- Without a compromised operating system, without protection against theft of the private key, and without protection against a user who confirms without comparing (PRD §22.1).
