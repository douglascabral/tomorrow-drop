# Graphical application (Fyne)

Tasks 045 to 054 (block 06). Implementation in `cmd/tomorrow-drop` (Fyne screens and wiring), with the presentation rules in `internal/ui` and the first-contact flow in `internal/pairing`. The core never imports Fyne; `internal/architecture_test.go` enforces it.

## 1. Architecture

- `internal/ui` turns core state into rows, labels, plans and validated previews. It imports no toolkit and is tested headless.
- `internal/pairing` holds the mutual confirmation flow. The command line and the window both call it, so the rules cannot diverge.
- `cmd/tomorrow-drop` starts the services (identity, trust, discovery, receiver, transfers), asks the person questions through dialogs and shows results. It never implements protocol, trust or transfer rules.

## 2. Screens

| Screen | Content |
|---|---|
| Home | Send, Trusted, Settings; discovered devices (trusted first); Refresh; manual connection by IP:port; the receiving port |
| Device | Send to this device, or Pair with it |
| Send | Destination (name or IP:port); add files and send them; text; send the clipboard after a preview; progress bar and Cancel |
| Pairing (dialog) | Both complete fingerprints, instructions, Confirm or Cancel, then a name for the device |
| Incoming offer (dialog) | Sender name, kind, item count, total size, file names; Accept or Decline |
| Trusted devices | List with identifier and name; Inspect shows the full fingerprint and can remove trust |
| Settings | Language: auto, pt-BR or en (applies on the next start) |

## 3. Availability (task 053)

The window drives availability through its lifecycle events:

- **Foreground**: discovery answers, the scanner runs, the device list refreshes and the 30-second schedule runs (`discovery.Controller`).
- **Background**: discovery and the incoming listener stop, so no new connections are accepted. Transfers already running are not interrupted.

Whether a desktop minimize produces these lifecycle events depends on the Fyne driver of each operating system. This must be verified on Linux, Windows and macOS before task 053 is marked complete.

## 4. Safety rules enforced by the GUI

- A device that is not trusted is never sent to. The send path checks the trust state of the session and offers pairing instead (PRD §8.1). Covered by `TestSendRefusesDeviceThatIsNotTrusted`.
- The clipboard is read only when the person presses "Send clipboard", and a preview is shown before anything leaves the device (PRD §12.2).
- Every offer, including from trusted devices, needs an explicit answer (PRD §13).
- Pairing shows both fingerprints in full and stores trust only after both sides confirm.

## 5. Known limits (to verify in task 054)

- Clipboard images are not sent: the Fyne clipboard API exposes text only. Task 050 asks for images only where the API supports them.
- The receive dialog shows the total size and the file names, not the size of each file.
- The receiving port is shown on the home screen; the GUI does not have a separate "Receive" mode, because receiving is always on while the window is in the foreground.
- Pairing and offers appear as dialogs over the current screen rather than as separate screens.
- Accessibility (screen readers, contrast) and real-desktop behavior were not verified in this environment.
- On macOS, Fyne prints cache-directory errors when the app runs with a HOME that has no `Library/Caches`; the application still starts.
