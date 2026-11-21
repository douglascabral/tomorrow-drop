# Android and iOS/iPadOS

Tasks 055 to 061 (block 07). This document records the requirements and what was verified. **No device was tested in this environment.** The tasks stay open until they run on real hardware.

## Android build status

- SDK installed with the `android` CLI into `~/Library/Android/sdk`: platform-tools 37.0.3, platform android-36, build-tools 36.0.0 and NDK 29.0.14206865. Java comes from Homebrew (`openjdk`), which the tools need.
- `fyne package --target android` produces `dist/Tomorrow_Drop.apk`: package `org.tomorrowdrop.app`, version 0.1.0, minimum SDK 21, native libraries for arm64-v8a, armeabi-v7a, x86 and x86_64. The APK is signed by the Fyne tool with a debug certificate, so it installs on devices in developer mode. It is about 127 MB because it is a debug build; a release build needs a release keystore.
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

The screens use Fyne containers that resize, and the send and home screens scroll. Touch targets, orientation changes and accessibility have not been checked on a device.

## 061 — Device tests

Not run. They need a physical Android device, an iPhone or iPad, and a desktop on the same network, including suspension and the enterprise network cases of task 024.
