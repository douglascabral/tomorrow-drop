# Android and iOS/iPadOS

Tasks 055 to 061 (block 07). This document records the requirements and what was verified. **No device was tested in this environment.** The tasks stay open until they run on real hardware.

## Android build status

- SDK installed with the `android` CLI into `~/Library/Android/sdk`: platform-tools 37.0.3, platform android-36, build-tools 36.0.0 and NDK 29.0.14206865. Java comes from Homebrew (`openjdk`), which the tools need.
- `fyne package --target android` produces `dist/Tomorrow_Drop.apk`: package `org.tomorrowdrop.app`, version 1.1.0, minimum SDK 21, native libraries for arm64-v8a, armeabi-v7a, x86 and x86_64. The APK is signed by the Fyne tool with a debug certificate, so it installs on devices in developer mode. It is about 127 MB because it is a debug build; a release build needs a release keystore.
- The build found a real defect: a limit above 2 GiB passed to a formatting call overflowed on 32-bit ARM. The fix and the regression test `TestCoreBuildsOn32BitPlatforms` are in the repository.
- Not run: the APK on an emulator or a device. No emulator is set up.

## iOS build status

- Xcode 27.2 is installed and selected (`xcode-select -p` points to `Xcode.app/Contents/Developer`). The iPhoneOS and iPhoneSimulator SDKs are present.
- A development signing identity ("Apple Development") exists in the keychain. No provisioning profile is installed.
- `fyne package --target ios` (from `cmd/tomorrow-drop`) produces an arm64 iOS bundle, `dist/Tomorrow-Drop.app`, with `MinimumOSVersion` 15.0 and the bundle ID `org.tomorrowdrop.app`. The bundle is **not signed**.
- The official Fyne tool (v1.7.4) hard-codes `IPHONEOS_DEPLOYMENT_TARGET = 13.0` in its Xcode project template, and Xcode 27.2 rejects it (the minimum it accepts is 15.0). The build used a local copy of the tool with that one value changed to 15.0. This is a deviation from the official tool; it must go away once the official tool is updated or the project template is overridden.
- To install on a device, the bundle needs a provisioning profile for `org.tomorrowdrop.app` that includes the device UDID, and the signing identity must match the team. Those depend on the developer account and must be set by the owner of the account.

## 055 — Build requirements

| | Android | iOS/iPadOS |
|---|---|---|
| Toolchain | Android SDK and NDK; Java; the Fyne packaging tool | Full Xcode (not only the command-line tools); macOS |
| Signing | Keystore for release builds | Apple Developer account, certificate and provisioning profile (`--profile`, `--certificate`) |
| Local network | `INTERNET` (in the generated manifest). Discovery uses UDP broadcast, which does not need a multicast lock; confirm on device | `NSLocalNetworkUsageDescription` (present in the bundle since the patch in `scripts/fyne-tools/`). Without it iOS blocks discovery and pairing |
| Storage | Scoped storage; the system save dialog (`WRITE_EXTERNAL_STORAGE` and `READ_EXTERNAL_STORAGE` are in the generated manifest but the save dialog does not need them) | The Files app through the system export dialog |

The packaging metadata is in `FyneApp.toml`. The Fyne tool only supports the microphone usage key there, so the iOS local-network key comes from a patch to the tool. Build the patched tool with `scripts/fyne-tools/build.sh`; it applies `scripts/fyne-tools/ios-deployment-and-local-network.patch` to `fyne.io/tools v1.7.4` and writes `build/tools/fyne-ios15`. The patch also sets the iOS deployment target to 15.0, which current Xcode requires.

The local-network prompt text is English only: iOS shows it in the system language, and translating it needs a localized `InfoPlist.strings` resource that the tool does not yet produce.

## 056 and 057 — Lifecycle (verified in tests, not on device)

- Leaving the foreground stops discovery and the incoming listener. Covered by `TestBackgroundClosesIncomingListenerAndForegroundReopensIt`.
- On mobile, leaving the foreground also interrupts transfers and pairings in progress. Covered by `TestMobileBackgroundCancelsRunningOperations`, with the platform check `fyne.CurrentDevice().IsMobile()`.
- Whether the operating system delivers these lifecycle events when the screen locks must be confirmed on each device.

## 058 — Files

On phones each received file is saved through the platform's save dialog: `transfer.WriterDestination` (core, tested) asks the application for a writer, and `cmd/tomorrow-drop/save.go` opens `dialog.NewFileSave`, which is the Android storage picker or the iOS Files export.

Deviations from the task text:
- Android does not write straight into Downloads. Writing there needs MediaStore on Android 10 and later; the storage picker keeps the person in control of the location. The user chooses one place per file, so a batch asks once per file.
- The save dialog creates new files and does not check for an existing name on every platform; the destination never overwrites a file it did not create in this run.

Still to verify on device (task 061): the picker appears after accepting an offer, the file is written completely, and cancelling the picker does not leave a partial file.

A regression test (`TestReceiveDestinationUsesPlatformSaveOnPhones`) covers the old fault, where phones saved into `~/Downloads`, which does not exist there.

## 059 — Clipboard

The clipboard is read only after the person presses "Send clipboard", with the same preview as on desktop. Fyne's clipboard API exposes text only, so text and links work and images do not, on any platform. Reading images would need native code per platform; task 059 stays open until that decision is made.

## 060 — Touch and screen sizes

The screens use Fyne containers that resize, and the send and home screens scroll. On phones and tablets `themeFor(true)` (`cmd/tomorrow-drop/theme.go`) enlarges the padding, text and icons, so a button is at least 64 dp high and a device row at least 72 dp, above the 48 dp guideline. `TestMobileTouchTargetsAreLarge` checks this. Text fields and buttons follow the theme; the settings picker is kept at 64 dp by `gui.touch` (`TestMobileControlsAreLargeOnEveryScreen`, `TestTouchKeepsPickersLargeOnlyOnMobile`); desktop keeps the compact theme (`TestDesktopKeepsCompactTheme`). Orientation changes, accessibility and the real size on a device have not been checked.

## 061 — Device tests

Not run. They need a physical Android device, an iPhone or iPad, and a desktop on the same network, including suspension and the enterprise network cases of task 024.

## Device name in the announcement (task 022 follow-up)

The name a phone or tablet announces comes from the platform (`cmd/tomorrow-drop/devicename_*.go`, selected by build tags), formatted by `ui.DeviceLabel`:

- iOS/iPadOS: `UIDevice.name` and `systemName systemVersion`, through cgo with Objective-C. Since iOS 16, Apple may return only the generic model name (for example "iPad") unless the app has the user-assigned device name entitlement; the name shown on the other device will be whatever iOS returns. Not verified on a device.
- Android: `android.os.Build.MANUFACTURER` and `MODEL`, and `Build.VERSION.RELEASE`, read through JNI. `JNI_GetCreatedJavaVMs` is looked up with `dlsym` because the NDK does not export it to applications. Not verified on a device.
- Desktop: the computer name.

Both native builds compile (`GOOS=android` with the NDK clang and `GOOS=ios` with the iPhoneOS SDK). Reading the values on a device is still open.

## Screen on while the app is open

On phones the app keeps the screen on while it is in the foreground, and leaving the foreground lets it turn off, which interrupts transfers (task 057). Covered by `TestPhoneKeepsTheScreenOnWhileTheAppIsOpen`; desktop is unchanged (`TestDesktopDoesNotChangeScreenBlanking`).

- Android: a screen-bright wake lock (`screenawake_android.go`, JNI through `PowerManager`). It needs `android.permission.WAKE_LOCK`, which the Fyne packaging tool does not declare, so `scripts/fyne-tools/android-manifest.patch` adds it. The window flag that Fyne uses (`SetDisableScreenBlanking`) did not take effect on the test phone (`mHoldScreenWindow=null`), so it is not used. Checked on a Motorola G75: the lock is held while the app is open, the screen stayed awake for 75 s with a 60 s timeout, and the lock is released when the app leaves the foreground.
- iOS/iPadOS: `UIApplication.idleTimerDisabled` (`screenawake_ios.go`). Compiles; not checked on a device.
- Desktop: no change.

## Share from other apps (Android)

Tomorrow Drop appears in the share menu of other apps (`ACTION_SEND` and `ACTION_SEND_MULTIPLE`, any type), so a screenshot, a file or a text can be sent straight from the app that shows it. The manifest declares the filter (`scripts/fyne-tools/android-manifest.patch`). When the app comes to the foreground it reads what was shared (`share_android.go`, through the content resolver), copies the files into its folder, and shows on the home screen what waits to be sent: the person then chooses a paired device, and the share is sent (`share.go`). A share is sent once. Checked on a Motorola G75: the app is listed for shared pictures (`cmd package query-activities`). Not yet checked: choosing a device and receiving the file on the other side from the share menu, and iOS (no share extension yet).

## Saving a received file on Android

Saving a received file opens the system's storage picker, which takes the app to the background. Leaving the foreground cancels the running operations (task 057), so a reception was cut before its result reached the sender: the sender reported "the connection ended before the transfer finished". While the picker is open, the cancellation is skipped (`gui.pickingFile`, `TestPickerKeepsTheReceptionRunning`). Still to check on a device: the sender waits 30 s for the result, so a person who takes longer than that in the picker makes the sender fail with a timeout. Asking for the location before the data is sent would remove that limit.

## iPhone and iPad: native interface

The iPhone and iPad app does not use Fyne. The Fyne iOS driver draws with OpenGL ES (GLKit), and on iOS 27.2 the app showed a black screen, also in a minimal Fyne app; with v2.8.2 it closed at launch with SIGTRAP. The iOS app is therefore native:

- `cmd/tomorrow-drop-ios` is the Go core as a static library (`-buildmode=c-archive`), with no user interface. It runs the same packages as the other applications (identity, trust, discovery, transport, pairing, transfer) and exposes a small C API that returns state as JSON.
- `ios/` is the SwiftUI app, generated with XcodeGen from `ios/project.yml`. `TomorrowDrop/Core/Core.swift` reads the state every second and sends the answers; the screens are in `TomorrowDrop/Screens/`.
- `make install-ios` (`scripts/install-ios.sh`) builds the library for the iPhone SDK, generates the project, builds it with the team of the Apple Development certificate (automatic signing) and installs it on the connected iPhone or iPad. The app is copied to `dist/TomorrowDrop-ios-arm64.app`, with the identifier `br.com.douglascabral.tomorrowdrop`.

Screens of the iPhone and iPad app (`ios/TomorrowDrop/Screens`), checked by type-checking against the iPhone Simulator SDK (`swiftc -typecheck`), not run on a device yet:

- Home: a **Paired devices** section (from the trust store, so it survives a restart and needs no announcement), then **Devices found** without the paired ones. The same device paired more than once is listed once, by its latest pairing at that address (`TestPairedListHasOneLinePerAddress`). Each paired device shows Online (green) or Offline (red) and opens its screen in either state, also with no known address (the send actions are then disabled), so it can always be unpaired, as on the other systems; a device not checked yet is offline, and there is no grey state. Paired and found devices use the same line layout; the check is a TCP connect to its last address, every 30 seconds while the app is open (`cmd/tomorrow-drop-ios/presence_test.go`, type-checked in Swift; not run on a device yet).
- Connect by IP: two fields, IP address and port, with the port filled in (47521).
- The screen of a paired device: Send files, Send text, Send clipboard, Receive (a waiting screen: the other device sends by itself while the app is open, and each transfer still needs the person's authorization), Rename, Unpair.

Requirements: Xcode signed in to the Apple account, XcodeGen (`brew install xcodegen`), the device unlocked, paired and in developer mode. A profile of a free Apple account expires after 7 days; the command must be run again then.

On the first launch the app asks for the local network: allow it, or discovery and pairing do not work. Received files are saved in the app's Documents, which the Files app shows (Tomorrow Drop folder).

The iPhone and iPad app keeps no history: an operation in progress is listed, and its outcome is shown once, until it is closed. A received text opens at once, with a button to copy it.

Not yet covered on iOS: the share sheet from other apps (no share extension), reading copied files from the clipboard (iOS does not give them to other apps), and checks on a real network. Pictures and text from the clipboard are sent.
