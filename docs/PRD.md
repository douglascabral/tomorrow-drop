# Tomorrow Drop
## Product Requirements Document (PRD)

**Version:** 1.9  
**Status:** Approved for implementation  
**Slogan:** *Transferiu. Usou. Pronto.* (Portuguese brand slogan; roughly "Transferred. Used. Done.")  
**License:** MIT  
**Language:** Go  
**Graphical interface:** Fyne  
**Platforms:** Linux, Windows, macOS, Android and iOS/iPadOS

---

# 1. Overview

Tomorrow Drop is a free, minimalist, cross-platform application for direct transfer of files, texts, links, images and clipboard content between devices connected to the same local network.

The application does not depend on accounts, cloud services, central servers or an internet connection.

The main flow is simple:

1. Open Tomorrow Drop.
2. Find another device on the local network.
3. Perform secure pairing, if trust does not exist yet.
4. Choose the content.
5. Send.
6. The recipient accepts or declines.
7. The transfer is completed.

**Product principle:** Transferiu. Usou. Pronto.

# 2. Goals

Tomorrow Drop shall:

- Allow direct transfers between computers and mobile devices.
- Work without external infrastructure.
- Use cryptography and identity authentication.
- Keep a simple and consistent visual interface.
- Minimize operating system permissions.
- Reduce external dependencies.
- Avoid unnecessary resident processes.
- Operate on home and enterprise networks.
- Offer a CLI for automation and environments without a graphical interface.
- Respect user privacy.

## 2.1 Out of scope

The following will not be implemented in the MVP:

- User accounts.
- Cloud synchronization.
- Relay servers.
- Transfers over the internet.
- Persistent transfer history.
- Automatic clipboard synchronization.
- Resuming interrupted transfers.
- QR Code.
- Pairing by camera.
- Background transfer services on mobile.
- Automatic discovery across different VLANs.
- Automatic update of identity keys.

# 3. Platforms and technologies

## 3.1 Operating systems

The application will support:

- Linux.
- Windows.
- macOS.
- Android.
- iOS/iPadOS.

## 3.2 Language and interface

**Go** will be the main language.

**Fyne** will be used for the cross-platform graphical interface.

The command-line interface, named `tdrop`, will also be implemented in Go.

The application core shall be shared between the GUI and the CLI.

## 3.3 Native libraries

Whenever possible, use the Go standard library:

- `crypto/tls`
- `crypto/ed25519`
- `crypto/sha256`
- `crypto/rand`
- `crypto/x509`
- `net`
- `context`
- `io`
- `os`

External dependencies will be used only when necessary and shall be evaluated for maintenance, security and compatibility with the MIT license.

# 4. Architecture

Tomorrow Drop will have a modular architecture.

## 4.1 Shared core

The core will be responsible for:

1. Cryptographic identity.
2. UDP discovery.
3. TCP/TLS connections.
4. Pairing.
5. Trust storage.
6. Transfer protocol.
7. Session management.
8. Lifecycle control.
9. Error handling.
10. Resource cleanup.

## 4.2 Interfaces

The Fyne GUI and the CLI will use the same core.

The authentication, transfer and security rules must not diverge between the interfaces.

## 4.3 Communication

Communication will be direct between devices.

There will be no central server, account, cloud, relay or mandatory dependency on external services.

# 5. Device discovery

## 5.1 Protocol

Discovery will use UDP broadcast on the local network.

The protocol will have two logical message types:

**DISCOVER**

Requests information about available devices.

**ANNOUNCE**

Responds to the discovery request, preferably by UDP unicast to the requester.

There is no need for periodic presence announcements sent by every device.

## 5.2 Discovery interval

**Approved interval: 30 seconds.**

| Event | Behavior |
|---|---|
| Application opened | Send DISCOVER immediately |
| Application visible and available | Send DISCOVER every 30 seconds |
| User clicks Refresh | Send DISCOVER immediately |
| Return to foreground | Send DISCOVER immediately |
| Desktop minimized | Suspend discovery |
| Mobile in background | Suspend discovery |
| Transfer in progress | Suspend periodic queries |
| Application closed | No discovery activity |

When Refresh is clicked, the 30-second counter is restarted.

Repeated clicks will be rate-limited to avoid excessive queries.

## 5.3 Responses

Available devices shall respond to incoming queries, respecting:

- Rate limiting.
- A small random delay to spread responses.
- Maximum message size.
- Format validation.
- Rejection of malformed messages.

The implementation shall avoid unnecessary UDP traffic amplification.

## 5.4 Enterprise networks

Discovery shall respect the local broadcast domain.

The application will not attempt to cross VLANs, bypass firewalls or circumvent client isolation.

Enterprise networks may prevent discovery or direct communication between devices.

In these situations, manual connection will be available.

## 5.5 Manual connection

The user may provide:

- IP address.
- TCP port.

Manual connection is subject to the same cryptographic checks as automatic discovery.

The IP address will not be used as a trusted identity.

## 5.6 Discovery security

UDP announcements are considered untrusted.

No discovery information is sufficient to authorize a transfer.

The device must be authenticated by its cryptographic identity before it receives trust.

# 6. Cryptographic identity

Each installation will have a persistent Ed25519 key pair.

The public key will cryptographically identify the device.

The private key must remain local and be protected by the appropriate mechanisms of each platform.

Identity does not depend on:

- Device name.
- IP address.
- MAC address.
- Network interface.
- Operating system user name.

A reinstallation that generates a new key will be treated as a new identity.

# 7. Fingerprint

The fingerprint will be computed over the canonical 32-byte representation of the Ed25519 public key.

**Algorithm:**

`SHA-256(canonical Ed25519 public key)`

The result has 32 bytes, presented as 64 hexadecimal characters.

The interface shall:

- Display the complete fingerprint.
- Group the characters for readability.
- Clearly identify which device each fingerprint belongs to.
- Avoid truncation.
- Allow visual comparison.

Short codes shall not be used as substitutes for the complete verification.

# 8. Secure transport

Transfer connections will use **TLS 1.3**.

The implementation must guarantee:

- Traffic encryption.
- Message integrity.
- Authentication of the cryptographic identity.
- Verification of possession of the private key.
- Binding between the Ed25519 identity and the credential presented in TLS.
- Verification of the previously trusted public key on reconnections.

## 8.1 Provisional connection

On first contact, a TLS connection may be established before out-of-band authentication.

This connection is considered provisional.

Although encrypted, it is not considered authenticated until pairing is confirmed.

No content transfer is authorized in this state.

## 8.2 Reconnections

Devices that are already paired must be authenticated by comparison with the persisted public key.

Keys different from the expected ones will result in blocking.

There will be no automatic acceptance of identity changes.

# 9. Initial pairing

Pairing will be performed exclusively by **manual comparison of complete fingerprints**.

There will be no QR Code, camera, PAKE or short SAS codes.

## 9.1 Flow

1. Two devices establish a provisional TLS connection.
2. Each device identifies the public key presented by the other.
3. Each device computes its local fingerprint.
4. Each device computes the fingerprint of the remote identity.
5. Both screens display the local and remote identities.
6. The user compares the fingerprints cross-wise.
7. Each device requires explicit confirmation.
8. The confirmations are bound to the same session and the same identities.
9. After mutual confirmation, the public keys are stored as trusted.
10. The connection then allows authorized operations.

## 9.2 Cross comparison

The local fingerprint of device A must match the remote fingerprint shown on device B.

The local fingerprint of device B must match the remote fingerprint shown on device A.

Both comparisons are mandatory.

## 9.3 Confirmation

The interface must present clear instructions.

Confirmation must not be interpreted as automatic or implicit.

The protocol must prevent confirmations from different sessions from being reused.

## 9.4 Persistence

**Approved decision:** trust remains valid until manually removed.

There is no automatic expiration by time or inactivity.

## 9.5 Revocation

The user may remove a trusted device.

After revocation, a new connection requires new pairing.

# 10. Security states

The application shall distinguish:

**Discovered:** device found, not authenticated.

**Connecting:** attempt to establish a connection.

**Awaiting verification:** provisional connection, waiting for pairing.

**Trusted:** identity authenticated and persisted.

**Identity changed:** public key different from the stored one; connection blocked.

**Disconnected:** no active session.

The application must not present a connection as trusted merely because TLS was established.

# 11. Sending content

The user may send:

- Individual files.
- Multiple files.
- Texts.
- Links.
- Clipboard content.
- Copied images, when supported by the platform.

Every transfer will be started by an explicit user action.

There will be no automatic sending of content.

# 12. Clipboard

Tomorrow Drop will allow manual sending of the clipboard.

## 12.1 Content

- Text.
- Links.
- Images, when available through the platform APIs.

## 12.2 Privacy

The application must not read the clipboard automatically at startup.

Reading occurs only after the user selects **Send clipboard** (Enviar área de transferência).

A preview of the content is shown before sending.

## 12.3 Receiving

Received content will not automatically replace the recipient's clipboard.

The user may copy the received text manually.

There will be no continuous synchronization or persistent history.

# 13. Receive authorization

**Approved decision:** every transfer requires explicit acceptance by the recipient.

This also applies to already trusted devices.

The request must present:

- Authenticated identity of the sender.
- Content type.
- File name, when applicable.
- File or file set size.
- Accept and Decline actions.

Declining must end the request without saving the content.

Unanswered requests will time out.

# 14. File destination

**Approved decision:** Downloads will be the default destination.

## 14.1 Desktop

Use the Downloads folder of the operating system user.

## 14.2 Android

Use the appropriate Android storage mechanisms, respecting the available permissions and APIs.

## 14.3 iOS/iPadOS

Use a location accessible through the Files app or the native export mechanism.

The application must not assume that a universally writable Downloads folder exists, as on desktop.

## 14.4 Conflicts

Existing files must not be silently overwritten.

The implementation shall adopt a safe renaming policy or request an explicit decision from the user.

# 15. Integrity and temporary files

Received files will initially be written to temporary storage.

On completion, the application must:

1. Verify the completion of the protocol.
2. Validate the expected size.
3. Verify the content integrity according to the mechanism defined by the protocol.
4. Make the final file available.
5. Remove unneeded temporary data.

The implementation must detect truncation and transmission failures.

When supported, the final availability must use atomic renaming.

Incomplete files must never be presented as completed.

# 16. Interrupted transfers

**Approved decision: there is no resume.**

An interrupted transfer will be invalidated.

A new attempt must start from the first byte.

There will be no checkpoints, persistent progress storage or state synchronization between attempts.

The following are considered interruptions:

- Network failure.
- Cancellation.
- Timeout.
- Application shutdown.
- State change incompatible with the transfer.

Partial files must be removed.

# 17. Lifecycle — Desktop

Applies to Linux, Windows and macOS.

## 17.1 Visible application

While the window is visible:

- UDP discovery may operate.
- The TCP/TLS listener is available.
- New connections may be accepted.
- Transfers may be started and received.

## 17.2 Minimized application

When minimized:

- Suspend UDP queries.
- Close the listener for new connections.
- Do not accept new requests.
- Keep only transfers already authorized and in progress.
- Close the kept connections when they finish, are cancelled, fail or time out.

If there is no active transfer, network resources are released immediately.

## 17.3 Shutdown

Explicitly closing the application must:

- Cancel transfers.
- Close connections.
- Close listeners.
- Stop discovery.
- Release resources.

Minimizing and closing are distinct actions.

# 18. Lifecycle — Mobile

Applies to Android and iOS/iPadOS.

The application must remain in the foreground during:

- Discovery.
- Pairing.
- Sending.
- Receiving.

When entering the background:

- Stop discovery.
- Close listeners.
- Cancel transfers.
- Close connections.
- Invalidate incomplete operations.
- Release network resources.

There will be no background transfer service in the MVP.

When returning to the foreground, the application will send a new discovery query immediately.

# 19. Graphical interface

## 19.1 Guidelines

The interface must be:

- Minimalist.
- Clear.
- Consistent.
- Responsive.
- Suitable for small screens.
- Free of excessive decorative elements.
- Oriented to one main action per screen.

Detailed cryptographic information will be presented mainly during pairing, identity inspection and security troubleshooting.

## 19.2 Visual identity

The logo will use **two mountains and a sun**.

The dark theme is the initial visual reference.

The palette should favor neutral colors, with discreet purple accents and success or error indicators when needed.

## 19.3 Screens

### Home

- Send.
- Receive.
- Discovered devices.
- Refresh.
- Manual connection.
- Settings.

### Pairing

- Local device name.
- Remote device name.
- Complete fingerprints.
- Instructions for cross comparison.
- Confirm.
- Cancel.

### Send

- Select files.
- Send clipboard.
- Write text.

### Receive request

- Sending device.
- Content information.
- Accept.
- Decline.

### Transfer

- Content name.
- Progress.
- Cancel.

### Completion

- Confirmation.
- Open content.
- Back.

### Trusted devices

- Device list.
- Fingerprint.
- Remove trust.

### Settings

- Local identity.
- Essential preferences.
- Application information.

There will be no QR Code screens.

# 20. CLI

The application will include the `tdrop` CLI.

Planned commands:

- `tdrop discover`
- `tdrop identity show`
- `tdrop pair <destination>`
- `tdrop trust list`
- `tdrop trust remove <id>`
- `tdrop send <file> --to <destination>`
- `tdrop send --stdin --to <destination>`
- `tdrop receive`
- `tdrop daemon`

The CLI shall use the same core as the GUI.

## 20.1 Receiving through the CLI

Receiving requires explicit authorization.

By default, authorization is interactive.

Any non-interactive policy must be consciously configured by the operator and documented, without weakening identity authentication.

## 20.2 Daemon

`tdrop daemon` allows continuous operation when explicitly started.

This mode is an exception to the availability policy tied to the graphical window.

# 21. Local storage and privacy

The application may persist:

- Local private key.
- Trusted public keys.
- Identifiers and friendly names.
- Essential preferences.

The application must not persist as history:

- Transferred texts.
- Clipboard content.
- Lists of sent or received files.
- Session keys.
- Progress of interrupted transfers.

Files intentionally saved by the user remain in the destination chosen by the user.

Temporary data must be removed after use or during the cleanup of incomplete sessions.

# 22. Threat model

Tomorrow Drop must protect against:

- Traffic interception.
- Modification of data in transit.
- MITM attacks during first contact, when the user correctly performs the out-of-band comparison.
- UDP announcement spoofing.
- Transfer attempts by unauthenticated devices.
- Unexpected changes to trusted public keys.
- Malicious transfer requests.
- Resource abuse through unauthorized messages or connections.

## 22.1 Security limits

The project will not promise protection against:

- A compromised operating system.
- Theft of the private key.
- Physical or logical tampering with the screens.
- A user who confirms fingerprints without comparing them.
- Denial of service on the local network.
- Blocks imposed by network infrastructure.

TLS alone will not be treated as identity authentication.

# 23. Test plan

## 23.1 Security

- Ed25519 identities.
- SHA-256 fingerprints.
- TLS authentication.
- Mutual confirmation.
- MITM rejection.
- Blocking of changed identities.
- Trust revocation.
- Rejection of unauthorized transfers.

## 23.2 Discovery

- Immediate query on open.
- Periodic 30-second query.
- Manual refresh.
- Counter restart after refresh.
- Rate limiting.
- UDP unicast responses.
- Suspension during transfer.
- Operation on enterprise networks.
- Manual connection by IP and port.

## 23.3 Transfers

- Small and large files.
- Multiple files.
- Texts and links.
- Clipboard.
- Acceptance and decline.
- Cancellation.
- Network failures.
- Timeout.
- Restart from the first byte.
- Integrity.
- Temporary file cleanup.
- Name collisions.

## 23.4 Lifecycle

- Desktop visible.
- Desktop minimized during transfer.
- Desktop minimized without transfer.
- Explicit shutdown.
- Mobile in foreground.
- Mobile in background.
- Screen lock.
- Return to the application.

## 23.5 Compatibility

Run tests on:

- Linux.
- Windows.
- macOS.
- Android.
- iOS/iPadOS.

# 24. Approved technical guidelines

All product and architecture decisions are approved for implementation.

## 24.1 Cross-platform interface

- Go as the main language.
- Fyne as the graphical framework.
- Core shared with the CLI.
- Support for the five defined platforms.

## 24.2 Security

- TLS 1.3.
- Ed25519.
- SHA-256.
- Complete fingerprints.
- Manual pairing with mutual confirmation.
- Persistent trust until revocation.
- Identity pinning.
- Rejection of changed keys.

## 24.3 Storage

- Downloads as the default destination, according to the mechanisms of each platform.
- Temporary files.
- Integrity verification.
- Cleanup of incomplete files.
- No transfer resume.

## 24.4 Discovery

- Local UDP broadcast.
- Immediate search on open.
- Immediate search on refresh.
- Periodic interval of 30 seconds.
- Responses preferably by UDP unicast.
- Rate limiting.
- Random response delay.
- Manual connection by IP and port.

## 24.5 Permissions and lifecycle

- No QR Code.
- No camera permission.
- Clipboard accessed only by explicit action.
- Desktop keeps authorized transfers after minimizing.
- Mobile requires the foreground.
- No resident services on mobile.

## 24.6 Distribution and quality

- MIT license.
- Minimal external dependencies.
- Security tests.
- Interoperability tests.
- Packaging for each platform.
- Security review before stable publication.

**Status:** approved.

Tests, integrations and reviews are execution and verification activities, not pending product decisions.

# 25. Implementation plan

## Phase 1 — Secure core in Go

Implement:

1. Ed25519 identities.
2. SHA-256 fingerprints.
3. TLS 1.3.
4. Manual pairing.
5. Persistent trust.
6. UDP discovery.
7. Manual connection.
8. Basic file and text transfer.

## Phase 2 — Desktop and CLI

Implement:

1. Fyne interface.
2. Discovery every 30 seconds.
3. Manual refresh.
4. Clipboard.
5. Receive authorization.
6. Desktop lifecycle control.
7. CLI sharing the core.

## Phase 3 — Mobile

Implement:

1. Android.
2. iOS/iPadOS.
3. Storage integration.
4. Clipboard under explicit action.
5. Foreground control.
6. Desktop-to-mobile tests.

## Phase 4 — Stabilization

Execute:

1. Security tests.
2. Protocol review.
3. Interoperability tests.
4. Dependency review.
5. Documentation.
6. Packaging.
7. Publication under the MIT license.

# 26. MVP acceptance criteria

The MVP will be considered complete when:

1. Devices on the same permitted network can discover each other via UDP.
2. Discovery happens immediately on open or refresh.
3. Periodic queries happen every 30 seconds while the application is available.
4. Manual connection by IP and port works.
5. First pairing requires cross comparison of complete fingerprints.
6. Trust persists until manual revocation.
7. Changed identities are rejected.
8. The transport uses TLS 1.3 with authentication of the expected identity.
9. Every received transfer requires authorization.
10. Files, texts and links can be transferred.
11. The clipboard can be sent manually.
12. Received files are saved to the appropriate destination.
13. Incomplete files are not presented as completed.
14. Interrupted transfers restart from scratch.
15. The desktop allows completing already authorized transfers after minimizing.
16. Mobile requires the application in the foreground.
17. There is no QR Code and no need for a camera.
18. There is no account, cloud or central server.
19. The CLI and the GUI share the same core.
20. The application works on the defined platforms.
21. The source code and documentation are available under the MIT license.

---

# 27. Final specification status

**Version:** 1.9  
**Product decisions:** approved  
**Architecture:** approved  
**Security:** approach approved; implementation subject to tests and review  
**Discovery:** on-demand UDP, with a 30-second interval  
**Pairing:** complete fingerprint, no QR Code  
**Interface:** Go + Fyne  
**License:** MIT  
**Next step:** implementation

---

# Tomorrow Drop

**Transferiu. Usou. Pronto.** (roughly "Transferred. Used. Done.")

A local, direct and minimalist application for sharing content between devices, with encryption, explicit authentication and user control.
