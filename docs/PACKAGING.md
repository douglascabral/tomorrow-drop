# Packaging and distribution

Task 069 and the build parts of task 067. Scripts live in `scripts/`.

## Status

| Target | Status | Evidence |
|---|---|---|
| macOS application bundles (arm64 and amd64) | Built, **unsigned** | `scripts/package-macos.sh`; each binary was checked with `lipo`, and the bundle identifier, name, version, Local Network key and icon with `PlistBuddy` |
| macOS signing and notarization | Not done | Needs a Developer ID certificate and an Apple account, which this project does not have yet |
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

It produces two bundles, one per processor architecture, each with its own binary:

- `dist/TomorrowDrop-macos-arm64.app` for Apple silicon;
- `dist/TomorrowDrop-macos-amd64.app` for Intel Macs.

Each run replaces them. `MACOS_ARCHES=arm64 make package-macos` builds only the listed architectures. The binaries are compiled with `CGO_ENABLED=1` by the Apple toolchain of this machine (Xcode); the amd64 build runs on an Apple silicon Mac without Rosetta, because only the compiler targets that architecture. Each bundle is checked with `lipo` before packaging, so a wrong architecture stops the build.

Both bundles are tested here only as built files; they were not started on Intel hardware.

## Reproducibility

The bundle is built from the committed `go.mod` and `go.sum`. For the core binaries, `CGO_ENABLED=0 go build -trimpath` gives reproducible output (see `docs/BUILD.md`). The graphical binary needs cgo, so its output is not byte-identical across machines.

## Patches of the packaging tool

`scripts/fyne-tools/build.sh` builds `build/tools/fyne-ios15` from `fyne.io/tools v1.7.4` with two patches: `ios-deployment-and-local-network.patch` (iOS deployment target and local-network key) and `android-manifest.patch` (adds `android.permission.WAKE_LOCK`, needed to keep the screen on, and the share targets that let other apps send to Tomorrow Drop, see `docs/MOBILE.md`).

## Linux (.deb) and Windows (.exe)

- `make package-linux` builds `dist/TomorrowDrop-linux-amd64.deb`. The program is compiled for amd64 inside the `fyne-cross` Linux image, then packed with `dpkg-deb` with its menu entry, its icon and the libraries it needs. Install with `sudo apt install ./dist/TomorrowDrop-linux-amd64.deb`.
- `make package-windows` builds `dist/TomorrowDrop-windows-amd64.exe`. It is cross-compiled with MinGW inside the same image, and the icon is embedded in the file. It is a GUI program (no console window).

Both need Docker running (on macOS, Colima or Docker Desktop). The first run downloads the image (about 4 GB) and the cross-compilers.

Limits: the `.deb` is not signed, and the `.exe` is not code-signed, so Windows SmartScreen may warn until it is signed with a certificate. The dependency list of the `.deb` follows the libraries Fyne uses on Linux; it was not checked with `readelf` yet.
