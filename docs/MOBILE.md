# Android and iOS/iPadOS

Tasks 055 to 061 (block 07). This document records the requirements and what was verified. **No mobile build was produced and no device was tested in this environment.** The tasks stay open until they are run on real hardware.

## 055 — Build requirements (to confirm on a machine with the SDKs)

| | Android | iOS/iPadOS |
|---|---|---|
| Toolchain | Android SDK and NDK; Java; the Fyne packaging tool | Full Xcode (not only the command-line tools); macOS |
| Signing | Keystore for release builds | Apple Developer account, certificate and provisioning profile (`--profile`, `--certificate`) |
| Local network | `INTERNET`; multicast needs a `WifiManager.MulticastLock` to receive UDP broadcast on some devices | Local Network permission: `NSLocalNetworkUsageDescription` must explain the use |
| Storage | Scoped storage; saving to Downloads through the system save dialog | The Files app through the system export mechanism |

The packaging metadata is in `FyneApp.toml`. Permissions must be added to the generated manifests and verified on device.

## 056 and 057 — Lifecycle (verified in tests, not on device)

- Leaving the foreground stops discovery and the incoming listener. Covered by `TestBackgroundClosesIncomingListenerAndForegroundReopensIt`.
- On mobile, leaving the foreground also interrupts transfers and pairings in progress. Covered by `TestMobileBackgroundCancelsRunningOperations`, with the platform check `fyne.CurrentDevice().IsMobile()`.
- Whether the operating system delivers these lifecycle events when the screen locks must be confirmed on each device.

## 058 — Files

The core already separates the saving step behind `transfer.Destination`. The Android and iOS implementations (native save dialogs and storage APIs) are not written yet, so receiving on mobile still uses the desktop destination. This is a gap that must be closed before mobile receiving can be considered working.

## 059 — Clipboard

The clipboard is read only after the person presses "Send clipboard", with the same preview as on desktop. Fyne's clipboard API exposes text only; images are not supported on any platform yet.

## 060 — Touch and screen sizes

The screens use Fyne containers that resize, and the send and home screens scroll. Touch targets, orientation changes and accessibility have not been checked on a device.

## 061 — Device tests

Not run. They need a physical Android device, an iPhone or iPad, and a desktop on the same network, including suspension and the enterprise network cases of task 024.
