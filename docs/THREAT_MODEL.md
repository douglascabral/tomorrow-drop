# Threat model review

Task 062. The threats come from PRD §22. For each one, this document names the control, the test that exercises it, and the risk that remains.

## Threats and controls

| Threat (PRD §22) | Control | Test |
|---|---|---|
| Interception of traffic | TLS 1.3 only (`MinVersion`), AEAD encryption | `TestTLSRequiresVersion13`, `TestMutualAuthenticationBindsIdentities` |
| Modification in transit | TLS integrity; per-item SHA-256 checked before publishing | `TestCorruptedHashIsRejectedAndNothingPublished` |
| MITM during first contact | Pairing confirmation bound to the TLS exporter; both fingerprints compared by people | `TestMITMRetransmissionOfConfirmationIsRejected`, `TestConfirmationFromPreviousSessionIsNotReused`, `TestUnilateralConfirmationDoesNotPersist` |
| Forged UDP announcements | Announcements never authorize anything; ANNOUNCE accepted only with a nonce of a recent DISCOVER | `TestScannerIgnoresAnnouncementWithUnknownNonce`, `TestScannerExpiresPendingNonces` |
| Transfers by unauthenticated devices | Transfer messages allowed only in the trusted state; `Receiver.Serve` refuses other sessions | `TestProvisionalBlocksTransfer`, `TestProvisionalRejectsForgedIncomingTransfer`, `TestUntrustedSessionCannotReceive` |
| Unexpected change of a trusted key | Pinned key; mismatch fails the handshake | `TestPinRejectsDifferentKey` |
| Certificate without the private key | TLS CertificateVerify checked against the certificate key | `TestRejectsCertificateWithoutPrivateKeyPossession` |
| Downgrade of the protocol version | Version list exchanged inside TLS; highest common version only | `TestRejectsIncompatibleVersionInHello`, `TestConfirmationCannotBeReusedWithAnotherVersion` |
| Malicious transfer requests | Every offer needs a decision; names, sizes and links validated before anything is written | `TestMaliciousOfferNamesAreRejected`, `TestValidateLinkAcceptsOnlyHTTP`, `TestDataBeyondDeclaredSizeIsRejected` |
| Resource abuse | Frame, datagram, file, offer, text, device-list and concurrency limits; rate limiting per origin | `internal/limits` tests, `TestListenerEnforcesPerOriginRateLimit`, `TestBusyReceiverDeclinesExtraTransfers` |
| Reachability checks of paired devices | A check is a TCP connect with no TLS and no data, to addresses of paired devices only; it authorizes nothing. Whoever sees the traffic learns only that a TCP connection was made to the device's address | `TestProbeIsTrueWhenSomethingAcceptsTCP`, `TestCheckRecordsEveryAddress` |

## Adversarial tests of the first pairing

The first-contact cases are covered by real network exchanges, not only by unit tests:

- a relay that retransmits confirmations unchanged between two sessions (MITM): both sides reject;
- a device that substitutes its key after the user saw the first one: `ErrIdentityChanged`;
- a confirmation replayed from an earlier session or with another version: rejected;
- one person confirms and the other does not: nothing is stored on either side;
- a declined comparison closes the session, so the other side does not wait.

## Fuzzing

The parsers that read data from the network or from files are fuzzed (task 063, see `docs/SECURITY_REVIEW.md`).

## Residual risks

These are accepted, documented, and outside the protocol's control:

- A person who confirms the fingerprints without comparing them is not protected (PRD §22.1). The screens show both complete fingerprints for that reason.
- A compromised operating system can read the private key or the screen (PRD §22.1).
- The device name shown in discovery is chosen by the other device. It is shown as a label only and never used for trust.
- Denial of service on the local network is not prevented (PRD §22.1); the rate limits only bound this device's own cost.
- `daemon --auto-accept-trusted` accepts every transfer from a trusted device without asking. It is an explicit operator choice (`docs/CLI.md`).
- Trust has no expiry and no shared revocation list: removing trust must be done on each device.
- On Windows, file permissions are not enforced by the operating system the way POSIX modes are; the private key relies on the user profile's default ACLs.
- The receive path has no resume (PRD §16); this costs time on large files but does not weaken integrity.
