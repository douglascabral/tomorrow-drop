# Graphical application (Fyne)

Tasks 045 to 054 (block 06). Implementation in `cmd/tomorrow-drop` (Fyne screens and wiring), with the presentation rules in `internal/ui` and the first-contact flow in `internal/pairing`. The core never imports Fyne; `internal/architecture_test.go` enforces it.

## 1. Architecture

- `internal/ui` turns core state into rows, labels, plans and validated previews. It imports no toolkit and is tested headless.
- `internal/pairing` holds the mutual confirmation flow. The command line and the window both call it, so the rules cannot diverge.
- `cmd/tomorrow-drop` starts the services (identity, trust, discovery, receiver, transfers), asks the person questions through dialogs and shows results. It never implements protocol, trust or transfer rules.

## 2. Screens

| Screen | Content |
|---|---|
| Home | This device's IPv4 address(es), so two open devices can be told apart by sight; discovered devices (trusted first, paired ones marked with a check icon; this device is never listed); Refresh; manual connection by IP:port; Settings. Send and receive are offered on the device screen, after a device is chosen |
| Device | Send to this device, or Pair with it |
| Send | Destination (name or IP:port); add files and send them; text; send the clipboard after a preview; progress bar and Cancel |
| Pairing (dialog) | Both complete fingerprints, instructions, Confirm or Cancel, then a name for the device |
| Incoming offer (dialog) | Sender name, kind, item count, total size, file names; Accept or Decline |
| Received text | The text, read only, and a button that copies it to the device clipboard |
| Trusted devices | List with identifier and name; Inspect shows the full fingerprint and can remove trust |
| Settings | Language: auto, pt-BR or en (applies on the next start); showing the current language saves nothing; the "about" text wraps |

## 2.1 Layout on desktop

On desktop every screen is a centered column of at most 960 px (`maxWidthLayout` in `cmd/tomorrow-drop/column.go`, applied in `gui.show`), so a wide window does not stretch the controls. Narrow windows use the whole width. The send screen on desktop does not use the column: the target and the drop area stay on top, the list of files or the text takes the height that is left, and the send action sits at the bottom. Phones keep the full width. Covered by `TestColumnCentersContentWithAMaximumWidth`, `TestColumnUsesTheWholeNarrowWindow` and `TestDesktopScreensUseTheColumnAndPhonesDoNot`.

## 2.2 Phones

- Every label wraps its text (`newLabel`, `newLabelStyled`), so long text never widens a screen beyond the window. `TestScreensFitThePhoneWidth` checks each screen at 420 px.
- Send screen on a phone: the hint speaks of "this phone or tablet", and the send action is a labeled button across the screen (`sendRow`).
- Receive screen title is "Receive" / "Receber"; the incoming-offer dialog title includes the sender's name.
- Device rows: the status stays on one line and the row is laid out again after its texts are set (`TestDeviceRowStatusStaysOnOneLine`).

## 2.3 Choosing files

- On a phone the chosen file is a content address, not a path. `stagePicked` copies it into `<home>/picked/pick-*/<name>` when it is chosen, so the send uses a real file. The copy is not removed yet. `TestPickedFileIsCopiedWithItsContent` covers the copy.
- On desktop the path is used directly, and the file dialog is sized to 90% of the window (`pickerSize`, `TestPickerFillsTheWindow`).

## 2.4 Result of a transfer

- A failed or cancelled transfer shows an error mark; only a completed one shows the success mark (`showOutcome`, `TestFailedTransferShowsAnErrorMark`).
- A connection that ends during a transfer reads "The connection ended before the transfer finished." A folder that cannot be written reads "The app cannot write to this folder. Check its permissions." (`apperr.MessageKey`, `TestMessageKeyNamesConnectionAndPermissionProblems`).

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
