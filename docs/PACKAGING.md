# Packaging and distribution

Task 069 and the build parts of task 067. Scripts live in `scripts/`.

## Status

| Target | Status | Evidence |
|---|---|---|
| macOS (arm64) application bundle | Built, **unsigned** | `scripts/package-macos.sh`; the bundle identifier, name, version and icon were checked with `PlistBuddy` |
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

The target builds `build/tools/fyne-ios15` if it is missing (see "Patches of the packaging tool" below), so `go install` is not needed. The bundle goes to `dist/Tomorrow Drop.app`; it is replaced on each run.

## Reproducibility

The bundle is built from the committed `go.mod` and `go.sum`. For the core binaries, `CGO_ENABLED=0 go build -trimpath` gives reproducible output (see `docs/COMPILACAO.md`). The graphical binary needs cgo, so its output is not byte-identical across machines.

## Patches of the packaging tool

`scripts/fyne-tools/build.sh` builds `build/tools/fyne-ios15` from `fyne.io/tools v1.7.4` with two patches: `ios-deployment-and-local-network.patch` (iOS deployment target and local-network key) and `android-keep-screen-on.patch` (adds `android.permission.WAKE_LOCK` to the Android manifest, needed to keep the screen on, see `docs/MOBILE.md`).
