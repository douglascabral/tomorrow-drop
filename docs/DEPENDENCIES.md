# Dependencies and licenses

Task 067. Generated with `scripts/dependency-report.sh` and reviewed by hand. The binaries contain 26 third-party modules. The rest of the build graph contains tools that are not compiled into the binaries.

## Shipped modules (in the CLI and the GUI)

| Module | Version | License | Note |
|---|---|---|---|
| fyne.io/fyne/v2 | v2.8.2 | BSD-3-Clause | GUI toolkit |
| fyne.io/systray | v1.12.3-0.20260810170012-af4e8e793ec4 | Apache-2.0 | System tray |
| github.com/BurntSushi/toml | v1.6.0 | MIT | |
| github.com/FyshOS/fancyfs | v0.0.1 | BSD-3-Clause | |
| github.com/anthonynsimon/bild | v0.14.0 | MIT | |
| github.com/clipperhouse/uax29/v2 | v2.2.0 | MIT | |
| github.com/fsnotify/fsnotify | v1.9.0 | BSD-3-Clause | |
| github.com/fyne-io/image | v0.1.1 | BSD-3-Clause | |
| github.com/fyne-io/oksvg | v0.2.1-0.20260918172519-a9af55fa95d9 | BSD-3-Clause | |
| github.com/go-gl/gl | v0.0.0-20260331235117-4566fea9a276 | MIT | OpenGL bindings |
| github.com/go-gl/glfw/v3.4/glfw | v0.1.0-pre.2 | BSD-3-Clause | Windowing |
| github.com/go-text/render | v0.2.1 | BSD-3-Clause | |
| github.com/go-text/typesetting | v0.3.5 | BSD-3-Clause | |
| github.com/godbus/dbus/v5 | v5.2.2 | BSD-2-Clause | Read from the license text: two conditions, no endorsement clause |
| github.com/jeandeaual/go-locale | v0.0.0-20250612000132-0ef82f21eade | MIT | System locale |
| github.com/jsummers/gobmp | v0.0.0-20230614200233-a9de23ed2e25 | MIT | |
| github.com/mattn/go-runewidth | v0.0.24 | MIT | |
| github.com/nfnt/resize | v0.0.0-20180221191011-83c6a9932646 | ISC | |
| github.com/nicksnyder/go-i18n/v2 | v2.5.1 | MIT | |
| github.com/srwiley/oksvg | v0.0.0-20221011165216-be6e8873101c | BSD-3-Clause | |
| github.com/srwiley/rasterx | v0.0.0-20220730225603-2ab79fcdd4ef | BSD-3-Clause | |
| github.com/yuin/goldmark | v1.8.2 | MIT | |
| golang.org/x/image | v0.45.0 | BSD-3-Clause | Upgraded for a fixed advisory |
| golang.org/x/net | v0.62.0 | BSD-3-Clause | Upgraded for fixed advisories |
| golang.org/x/sys | v0.49.0 | BSD-3-Clause | Upgraded for a fixed advisory |
| golang.org/x/text | v0.43.0 | BSD-3-Clause | Upgraded for fixed advisories |

Every shipped license is permissive and compatible with the MIT license of the project. The graphics stack (OpenGL, GLFW, typesetting) is the largest part; it is needed by Fyne.

## Build-graph modules that are not shipped

These appear in `go.sum` because of tooling and transitive test dependencies, and are not linked into the binaries: for example `github.com/spf13/cobra`, `github.com/urfave/cli/v2`, `github.com/josephspurrier/goversioninfo`, `github.com/akavel/rsrc`, `golang.org/x/tools` and `golang.org/x/mod`. Their licenses were checked where a license file is present (MIT, BSD and Apache-2.0 families). Modules without a license file in the module cache are listed by `scripts/dependency-report.sh` as UNKNOWN; none of them is shipped.

## Reproducible build

```sh
CGO_ENABLED=0 go build -trimpath -o bin/tdrop ./cmd/tdrop   # CLI: no cgo
go build -trimpath -o bin/tomorrow-drop ./cmd/tomorrow-drop  # GUI: needs cgo and the graphics libraries
```

The module versions are pinned by `go.mod` and `go.sum`. `go mod verify` checks the downloaded modules against `go.sum`.

## Vulnerability scan

`trivy fs .` reports no HIGH or CRITICAL advisory on the current `go.mod`, after the upgrade of the `golang.org/x/*` modules listed above.
