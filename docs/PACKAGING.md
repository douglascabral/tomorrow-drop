# Packaging and distribution

Task 069 and the build parts of task 067. Scripts live in `scripts/`.

## Status

| Target | Status | Evidence |
|---|---|---|
| macOS application bundles and disk images (arm64 and amd64) | Built, **signed ad hoc** | `scripts/package-macos.sh`; each binary was checked with `lipo`, and the bundle identifier, name, version, Local Network key and icon with `PlistBuddy`; the disk image is checked by `scripts/package-macos_test.sh` |
| macOS signing with a Developer ID, and notarization | Implemented, **not run with a real certificate** | `scripts/sign-macos.sh`, `scripts/notarize-macos.sh` and the `macos` job of `.github/workflows/release.yml`. They need a Developer ID Application certificate and an App Store Connect API key, which this project does not have yet. The tests use fake `codesign` and `xcrun`, so they prove the arguments and the failure handling, not Apple's acceptance |
| Linux and Windows packages | Not done | Needs their native toolchains (cgo with the graphics libraries of each system); no such machine was used |
| Android and iOS/iPadOS packages | Not done | See `docs/MOBILE.md`; needs the Android SDK/NDK, Xcode and devices |

## Metadata

`FyneApp.toml` (repository root) holds the name, version and reverse-domain ID `org.tomorrowdrop.app`. It must match `app.NewWithID` in `cmd/tomorrow-drop/main.go`. The icon is `assets/icon.png` (512×512), generated from `assets/icon.svg`.

## Tool

The packaging tool is `fyne` from `fyne.io/tools` (v1.7.4 used here; it reports Fyne library v2.8.2, the project's version). The older `fyne.io/fyne/v2/cmd/fyne` is deprecated.

```sh
make package-macos
```

The target builds `build/tools/fyne-ios15` if it is missing (see "Patches of the packaging tool" below), so `go install` is not needed.

It produces, for each processor architecture, a bundle with its own binary and a disk image that holds it:

- `dist/TomorrowDrop-macos-arm64.app` and `dist/TomorrowDrop-macos-arm64.dmg` for Apple silicon;
- `dist/TomorrowDrop-macos-amd64.app` and `dist/TomorrowDrop-macos-amd64.dmg` for Intel Macs.

The disk image is what is published: it opens to `Tomorrow Drop.app` and a link to Applications, to drag the application across. Each run replaces the files. `MACOS_ARCHES=arm64 make package-macos` builds only the listed architectures. The binaries are compiled with `CGO_ENABLED=1` by the Apple toolchain of this machine (Xcode); the amd64 build runs on an Apple silicon Mac without Rosetta, because only the compiler targets that architecture. Each bundle is checked with `lipo` before packaging, so a wrong architecture stops the build.

Both bundles are tested here only as built files; they were not started on Intel hardware.

## macOS signing and notarization

An application that is not signed by a Developer ID, and notarized by Apple, is blocked by Gatekeeper on the first open. The scripts do each step, and fall back to an ad-hoc signature (the bundle is intact and runs, but macOS does not know who made it) when there are no credentials, and say so.

| Variable | Meaning |
|---|---|
| `MACOS_SIGN_IDENTITY` | Name of a **Developer ID Application** certificate in the keychain, for example `Developer ID Application: Name (TEAMID)`. The bundle and the disk image are signed with it, with the hardened runtime and a secure timestamp. Without it: ad hoc. |
| `MACOS_NOTARY_KEY`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_ISSUER` | An App Store Connect API key (the `.p8` file, its key id and the issuer id). When `MACOS_NOTARY_KEY_ID` is set, the application and then the disk image are sent to Apple's notary service, and the ticket is stapled to each. Only an `Accepted` result passes; anything else stops the build and prints Apple's log. Needs `MACOS_SIGN_IDENTITY`, which is checked before anything is built. |

The order matters. The bundle is signed after its `Info.plist` is changed (a later change would break the signature). The application is notarized and stapled **before** it goes into the image, so the copy that people drag out of the image carries its own ticket; then the image is signed, notarized and stapled.

```sh
MACOS_SIGN_IDENTITY="Developer ID Application: Name (TEAMID)" \
MACOS_NOTARY_KEY=~/AuthKey_ABC123.p8 MACOS_NOTARY_KEY_ID=ABC123 MACOS_NOTARY_ISSUER=<issuer-id> \
make package-macos
```

In the release workflow (`.github/workflows/release.yml`, job `macos`) the same variables come from repository secrets: `MACOS_CERTIFICATE` (the `.p12` file, base64), `MACOS_CERTIFICATE_PASSWORD`, `MACOS_NOTARY_KEY` (the `.p8` file, base64), `MACOS_NOTARY_KEY_ID` and `MACOS_NOTARY_ISSUER`. The job imports the certificate into a temporary keychain that is deleted at the end, writes the key to the runner's temporary folder and removes it, and never prints either. Without the secrets the job still builds, signs ad hoc and prints a warning, so the release says what it is. After the build it opens each image and verifies the signature of the application inside.

The command-line programs for macOS (`tdrop-macos-arm64`, `tdrop-macos-amd64`) are cross-compiled on Linux: the Go linker signs them ad hoc, which is what Apple silicon needs to run them, but they are not signed with a Developer ID nor notarized.

## Names of the release files

The files of a release carry the version in their name, after the program name and before the system: `TomorrowDrop-1.2.0-macos-arm64.dmg`, `TomorrowDrop-1.2.0-macos-amd64.dmg`, `TomorrowDrop-1.2.0-linux-amd64.deb`, `TomorrowDrop-1.2.0-windows-amd64.exe`, `tdrop-1.2.0-linux-amd64`, `tdrop-1.2.0-windows-amd64.exe`, `tdrop-1.2.0-macos-arm64` and `tdrop-1.2.0-macos-amd64`. `scripts/collect-release.sh` copies the built files (`bin/`, `dist/`) with those names, reading the version from the `VERSION` file; the built files keep their names without a version, so `make` and the other scripts do not change. In the workflow, a tag that is not `v<VERSION>` stops the release before anything is published, so a file never carries a version that is not the one tagged (`scripts/collect-release_test.sh`).

## Reproducibility

The bundle is built from the committed `go.mod` and `go.sum`. For the core binaries, `CGO_ENABLED=0 go build -trimpath` gives reproducible output (see `docs/BUILD.md`). The graphical binary needs cgo, so its output is not byte-identical across machines.

## Patches of the packaging tool

`scripts/fyne-tools/build.sh` builds `build/tools/fyne-ios15` from `fyne.io/tools v1.7.4` with two patches: `ios-deployment-and-local-network.patch` (iOS deployment target and local-network key) and `android-manifest.patch` (adds `android.permission.WAKE_LOCK`, needed to keep the screen on, and the share targets that let other apps send to Tomorrow Drop, see `docs/MOBILE.md`).

## Linux (.deb) and Windows (.exe)

- `make package-linux` builds `dist/TomorrowDrop-linux-amd64.deb`. The program is compiled for amd64 inside the `fyne-cross` Linux image, then packed with `dpkg-deb` with its menu entry, its icon and the libraries it needs. Install with `sudo apt install ./dist/TomorrowDrop-linux-amd64.deb`.
- `make package-windows` builds `dist/TomorrowDrop-windows-amd64.exe`. It is cross-compiled with MinGW inside the same image, and the icon is embedded in the file. It is a GUI program (no console window).

Both need Docker running (on macOS, Colima or Docker Desktop). The first run downloads the image (about 4 GB) and the cross-compilers.

Limits: the `.deb` is not signed, and the `.exe` is not code-signed, so Windows SmartScreen may warn until it is signed with a certificate. The dependency list of the `.deb` follows the libraries Fyne uses on Linux; it was not checked with `readelf` yet.
