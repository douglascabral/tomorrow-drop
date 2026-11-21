
# Tomorrow Drop — Implementation tasks

> Mandatory execution order: complete and validate one task before starting the next, unless a task explicitly declares independence. Mark `[x]` only after implementation, tests and corresponding documentation. Do not change PRD v1.9 decisions without approval. Do not invent security guarantees or replace identity verification with TOFU.

## 01 — Project foundation

[x] 001. Create the Go repository with `go.mod`, modular structure (`cmd/tdrop`, `cmd/tomorrow-drop`, `internal/identity`, `internal/trust`, `internal/discovery`, `internal/transport`, `internal/protocol`, `internal/transfer`, `internal/storage`, `internal/lifecycle`, `internal/ui`), README and `.gitignore`.
[x] 002. Add the MIT license, dependency notices and initial build documentation for Linux, Windows and macOS.
[   ] 003. Configure lint, formatting (`gofmt`), tests (`go test ./...`), static analysis (`go vet ./...`) and a multi-platform CI pipeline.
[x] 004. Define error standards, logging without sensitive data, cancellation with `context.Context`, resource limits and unit tests.
[x] 005. Define and document the versioned protocol: message formats, framing, size limits, timeouts, session identifiers, states and version negotiation.

## 02 — Identity and trust

[x] 006. Implement Ed25519 identity generation with `crypto/rand`, persistent storage and restrictive permissions; do not log the private key.
[x] 007. Implement loading and validation of an existing identity; handle corruption, absence and deliberate regeneration without silent overwrite.
[x] 008. Implement the SHA-256 fingerprint of the 32 canonical bytes of the Ed25519 public key, displaying all 64 hexadecimal characters in readable groups.
[x] 009. Implement local storage of trusted public keys, friendly names and manual revocation, with safe writes and persistence tests.
[x] 010. Implement issuance and validation of TLS certificates bound to the Ed25519 identity, using native Go APIs.
[x] 011. Implement TLS 1.3 with cryptographic authentication of both identities, certificate validation and pinning; forbid silent insecure acceptance on trusted connections.
[x] 012. Implement a provisional TLS state for first contact: encrypted but untrusted, with a message surface strictly limited to pairing.
[x] 013. Implement the mutual pairing flow with cross comparison of complete fingerprints and explicit confirmations on both devices, bound to the same session and the same keys.
[x] 014. Block transfers and privileged commands before pairing completes; persist trust only after proven mutual confirmation.
[x] 015. Reject unexpected identity changes and prevent downgrade, confirmation reuse or session confusion.
[x] 016. Create security tests for MITM, key/certificate substitution, provisional sessions, unilateral confirmation, revocation and reconnection.

## 03 — Discovery and connectivity

[x] 017. Define a versioned UDP format for `DISCOVER` and `ANNOUNCE`, with strict validation, maximum size and no sensitive information.
[x] 018. Implement a UDP discovery listener while the application is available, with preferably unicast responses, per-origin rate limiting and a short random delay.
[x] 019. Send `DISCOVER` immediately when the application opens/activates and when Refresh is clicked.
[x] 020. Implement periodic queries every 30 seconds only while the application is visible and available; restart the counter after Refresh and limit repeated clicks.
[x] 021. Suspend periodic queries during transfers and stop discovery when the application is minimized or in the background.
[x] 022. Implement an ephemeral list of discovered devices, entry update/expiration and visual distinction between discovered and trusted devices.
[x] 023. Implement manual connection by IP:port with input validation and the same TLS authentication as the automatic flow.
[   ] 024. Test discovery and connectivity on home Wi-Fi and enterprise networks: same VLAN, blocked broadcast, client isolation, multiple interfaces and no route.

## 04 — Protocol and transfer

[   ] 025. Implement the connection and transfer state machine, with cancellation, timeouts, concurrency limits and deterministic cleanup.
[   ] 026. Define offer metadata (type, names, sizes and count), path validation and limits to prevent traversal and resource exhaustion.
[   ] 027. Implement receive requests with authenticated identity and mandatory accept/decline for **each** transfer, including between trusted devices.
[   ] 028. Implement text and link transfer over authenticated TLS 1.3, without persistent history.
[   ] 029. Implement single-file sending in streaming mode, without loading all content into memory.
[   ] 030. Implement multiple-file sending, with individual and total progress and defined behavior for partial failure.
[   ] 031. Implement writing received data to temporary files with unpredictable names, size validation, content hash and completion confirmation.
[   ] 032. Implement secure publication of the completed file, avoiding silent overwrite and name collisions; use an atomic operation when supported.
[   ] 033. Implement cancellation, error, timeout and connection drop: invalidate the transfer, delete partial data and require resending from the first byte.
[   ] 034. Implement cleanup of temporary files abandoned by unexpected shutdowns, without deleting files not belonging to the application.
[   ] 035. Implement default saving to Downloads on desktop and a storage abstraction for native Android and iOS mechanisms.
[   ] 036. Create tests for integrity, large files, empty files, multiple files, failures at 99%, malicious names, limits and concurrency.

## 05 — CLI

[   ] 037. Implement `tdrop identity show`, displaying the local identity and complete fingerprint without revealing the private key.
[   ] 038. Implement `tdrop discover` and manual connection by IP:port.
[   ] 039. Implement `tdrop pair <destination>` with interactive confirmation and mutual verification.
[   ] 040. Implement `tdrop trust list` and `tdrop trust remove <id>`.
[   ] 041. Implement `tdrop send <file> --to <destination>` and `tdrop send --stdin --to <destination>`.
[   ] 042. Implement `tdrop receive` with mandatory interactive authorization by default.
[   ] 043. Implement `tdrop daemon` as an explicitly started continuous mode, documenting non-interactive authorization rules when configured.
[   ] 044. Create CLI↔CLI integration tests, exit codes, stdout/stderr, cancellation and headless execution.

## 06 — Fyne desktop interface

[   ] 045. Create the Fyne application with a minimalist visual identity: two-mountains-and-sun logo, dark theme, simple hierarchy and accessible components.
[   ] 046. Implement the home screen with Send, Receive, discovered devices, Refresh, manual connection and settings.
[   ] 047. Implement the pairing screen with the complete fingerprints of both devices, cross-comparison guidance, explicit confirmation and cancellation.
[   ] 048. Implement the trusted devices screen, fingerprint inspection and trust removal.
[   ] 049. Implement file selection, text composition and content preview before sending.
[   ] 050. Implement the manual “Send clipboard” action for text and links; add copied images only where the APIs support it, without automatic monitoring.
[   ] 051. Implement the receive dialog with sender identity, names, sizes and Accept/Decline buttons.
[   ] 052. Implement progress, cancellation, clear errors and a completion screen, without persistent history.
[   ] 053. Implement desktop behavior: visible allows discovery/listener; minimized closes the listener and discovery but keeps only previously authorized transfers until they finish, fail or are cancelled; closing the app cancels everything.
[   ] 054. Test the desktop GUI on Linux, Windows and macOS, including minimizing during a transfer, returning to the window, shutdown and file permissions.

## 07 — Android and iOS/iPadOS

[   ] 055. Prepare Fyne builds for Android and iOS/iPadOS, identifying toolchain, signing and minimum local-network permission requirements.
[   ] 056. Integrate UDP discovery and TLS with foreground/background events on Android and iOS; do not create a background transfer service.
[   ] 057. Implement interruption of discovery, pairing and transfers when entering the background or locking the screen, with resource cleanup and a fresh start.
[   ] 058. Integrate file selection and saving through native mechanisms: Downloads where feasible on Android, and a location accessible through the Files app/export on iOS.
[   ] 059. Integrate manual clipboard reading on mobile, without monitoring and without camera permission; validate real support for text, links and images.
[   ] 060. Adapt Fyne screens for touch, different screen sizes, accessibility and orientation.
[   ] 061. Run tests on real Android↔desktop, iOS↔desktop and Android↔iOS devices, including permissions, suspension and enterprise networks.

## 08 — Security, quality and release

[   ] 062. Review the threat model, authentication states, TLS validation and pinning; run adversarial tests of first pairing.
[   ] 063. Apply fuzzing to the UDP parsers, protocol messages, metadata and framing; verify memory and CPU limits.
[   ] 064. Review private key storage, permissions, logs, temporary files, error handling and overwrite protection.
[   ] 065. Validate full interoperability GUI↔GUI, GUI↔CLI and CLI↔CLI on supported platforms.
[   ] 066. Run performance and stability tests for large files, slow connections, network failures and many devices on one VLAN.
[   ] 067. Review dependencies, licenses and the build chain; generate a dependency list and reproducible build instructions.
[   ] 068. Document installation, secure pairing, manual connection, the 30-second behavior, clipboard, Downloads, lifecycle and application limits.
[   ] 069. Prepare packaging and distribution for Linux, Windows, macOS, Android and iOS/iPadOS, including signing when required.
[   ] 070. Run the final PRD v1.9 checklist and record evidence of approval for all MVP acceptance criteria.
