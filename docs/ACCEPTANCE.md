# MVP acceptance checklist (PRD §26)

Task 070. Each criterion is marked with the evidence that exists today:

- **Automated**: a test or a measurement in this repository covers it.
- **Partial**: part of it is covered; the rest needs a device or a real network.
- **Pending**: it needs hardware or a real network that this environment does not have.

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 1 | Devices on the same permitted network discover each other over UDP | Partial | Loopback tests (`TestScannerFindsListenerOverLoopback`, `TestDiscoverFindsAnnouncedDevice`). Real LAN pending (task 024) |
| 2 | Discovery happens at once on open or refresh | Automated | `TestControllerQueriesImmediatelyThenPeriodically`, `TestForegroundSendsQueryEvenWithinRefreshCooldown` |
| 3 | Periodic queries every 30 seconds while available | Automated | `TestSchedulePeriodicQueryEvery30Seconds`, `TestControllerSuspensionStopsPeriodicQueries` |
| 4 | Manual connection by IP and port works | Automated | `TestDialManualEndpointUsesSameTLSAuthentication`, `TestSendAfterPairingDeliversFile` |
| 5 | First pairing requires cross comparison of complete fingerprints | Automated | `internal/pairing` tests; the GUI and CLI show both complete fingerprints (manual check of the screens pending) |
| 6 | Trust persists until manual revocation | Automated | `TestAddTrustsAndPersistsAcrossOpens`, `TestRevocationRequiresNewPairing` |
| 7 | Changed identities are rejected | Automated | `TestPinRejectsDifferentKey`, `TestMITMRetransmissionOfConfirmationIsRejected` |
| 8 | Transport uses TLS 1.3 with the expected identity | Automated | `TestTLSRequiresVersion13`, `TestPinAcceptsExpectedKey` |
| 9 | Every received transfer requires authorization | Automated | `TestTrustedDevicesStillNeedAcceptance`, `TestDeclinedTransferSavesNothing` |
| 10 | Files, texts and links can be transferred | Automated | `TestSingleFileTransferPublishesIntactFile`, `TestTextTransferDeliversContentWithoutPersisting`, `TestLinkTransferAcceptsHTTPOnly` |
| 11 | The clipboard can be sent manually | Partial | Preview and validation tested (`TestClipboardPreview*`). The system clipboard itself is not exercised in tests |
| 12 | Received files are saved to the appropriate destination | Partial | Desktop Downloads automated; Android and iOS destinations not implemented (task 058) |
| 13 | Incomplete files are never presented as complete | Automated | `TestSenderDropAt99PercentLeavesNothingBehind`, `TestCorruptedHashIsRejectedAndNothingPublished` |
| 14 | Interrupted transfers restart from scratch | Automated | No resume exists; interrupted data is removed (`TestSenderDropAt99PercentLeavesNothingBehind`) |
| 15 | The desktop completes authorized transfers after minimizing | Partial | `TestDesktopBackgroundKeepsRunningOperations`. The real minimize behavior on Linux, Windows and macOS is pending (task 054) |
| 16 | Mobile requires the application in the foreground | Partial | `TestMobileBackgroundCancelsRunningOperations`. Pending on devices (task 061) |
| 17 | No QR Code and no camera | Automated | No QR or camera code or dependency exists (`docs/DEPENDENCIES.md`) |
| 18 | No account, cloud or central server | Automated | Design: devices talk only to each other; no such code exists |
| 19 | CLI and GUI share the same core | Automated | `TestCoreDoesNotDependOnFyne`; pairing is shared (`internal/pairing`) |
| 20 | The application works on the defined platforms | Partial | CLI cross-compiles for Linux and Windows; macOS application bundle built. Linux and Windows GUI, Android and iOS pending |
| 21 | Source code and documentation are under the MIT license | Automated | `LICENSE` (MIT); documentation in the repository |

## Result

Task 070 stays open. Criteria 1, 5, 11, 12, 15, 16, 20 need a real network or device test, and nothing in this environment can run them. The checklist records what has been shown and what has not.
