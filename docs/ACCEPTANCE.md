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

The table records what automated checks and manual device work show. Criteria 1, 5, 11, 12, 15, 16 and 20 depend on real networks and devices, which this environment does not have.

## Owner approval (task 070)

On 2026-11-21 the owner of the project tested the application on the devices in use and reported it working, and approved the MVP against this checklist. The approval covers the criteria that need devices and real networks (1, 5, 11, 12, 15, 16, 20), as the owner's device tests. The automated criteria are backed by the tests listed above, and `scripts/check.sh` passed on the tree that closes task 070.

Known limits, recorded so they are not forgotten: Android and iOS save received files in the app storage and the Files app (task 058), not in a system Downloads folder; there is no store distribution yet (task 069, signing).
