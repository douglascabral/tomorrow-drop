# Graphical application (Fyne)

Tasks 045 to 054 (block 06). Implementation in `cmd/tomorrow-drop` (Fyne screens and wiring), with the presentation rules in `internal/ui` and the first-contact flow in `internal/pairing`. The core never imports Fyne; `internal/architecture_test.go` enforces it.

## 1. Architecture

- `internal/ui` turns core state into rows, labels, plans and validated previews. It imports no toolkit and is tested headless.
- `internal/pairing` holds the mutual confirmation flow. The command line and the window both call it, so the rules cannot diverge.
- `cmd/tomorrow-drop` starts the services (identity, trust, discovery, receiver, transfers), asks the person questions through dialogs and shows results. It never implements protocol, trust or transfer rules.

## 2. Screens

| Screen | Content |
|---|---|
| Home | This device's IPv4 address(es), so two open devices can be told apart by sight; a **Paired devices** section (from the trust store, with the last address each one answered at, `home_list.go`); a **Devices found** section, without the paired ones; a status for each paired device (Online, Offline, or Paired until it is checked); Refresh; manual connection by IP and port (the port is filled in); Settings. Tapping a paired device connects to it and opens its actions (send, receive); a device with no known address says so |
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

## 2.5 Usability decisions

Terms. One word for one idea: a device is **paired** (not "trusted"), the action is **Pair** / **Unpair**. A device that is not paired reads "Tap to pair". Counts agree with the number ("1 file", "2 files"; `plural`). A request to send files reads "Receive files?" and "X wants to send you:"; it does not say the files were received.

Who a device is. A device announces only a name and a port, so it is known only after an authenticated connection. From then on its **paired name** replaces the name it announces in the list (`paired.go`); the device screen shows the real pairing date and fingerprint; and Unpair works from the device screen and from the list of paired devices, even when the device is offline (`TestPairedNameReplacesTheAnnouncedNameInTheList`, `TestUnpairRemovesTrustFromTheDevice`). Addresses are shown without the usual port.

Screens.
- Home: an empty list says it is looking and what to do ("Open Tomorrow Drop on the other device, on the same Wi-Fi network"). The logo is smaller on a phone.
- Send: Send says "Send" and stays disabled until there is a recipient and something to send. Files can be dragged onto the window on desktop (`droppedPaths`).
- Pairing and requests: going back declines, so the other device is not left waiting (`TestBackOnADecisionScreenDeclines`). The name of a new device starts from the name it announces. A pairing the person started ends on the device screen, not in an alert.
- Progress: Cancel works on a reception too, and a cancelled or cut reception shows a result instead of leaving the ring on screen.
- Result: the header is neutral ("Transfer"); "Open folder" is only on desktop.
- Manual connection: Enter connects; the card keeps its natural height.
- A device paired by IP:port stays in the device list for the rest of the session, even when it is on another network and does not announce itself (`TestDeviceConnectedByAddressStaysListedDuringTheSession`).
- Paired devices show Online or Offline, from a TCP check of their last address every 30 seconds while the window is open and after Refresh. Online devices are listed first and are green with the check icon. Offline ones are red, without the icon, and keep their name and stay tappable. A device not checked yet is neutral grey, never green (`TestPairedDeviceIsOnlineThenOffline`, `TestQueryAllAlsoChecksPairedDevices`, `TestOnlinePairedDevicesAreListedFirst`, `TestOfflinePairedDeviceHasNoCheckIcon`, `TestUnpairForgetsPresence`). See `docs/DISCOVERY.md` section 7.
- Paired devices are listed in their own section on the home screen, also after a restart, and never twice (`TestHomeListsPairedDevicesInTheirOwnSection`, `TestPairedDeviceIsNotRepeatedAmongFoundDevices`). Tapping one opens its actions (`TestTappingAPairedDeviceOpensItsActions`).
- Settings: languages are named for people (Automatic, Português (Brasil), English).
- Fingerprints are shown in four lines of four groups, so they can be compared piece by piece.
- This device's fingerprint is one tap away, right below the address: a "View my fingerprint" button on the home screen and on the screen of a paired device. It opens the whole fingerprint in four lines, with a button to copy it. No screen shows only the start of it, so nobody is tempted to compare just the beginning (`TestHomeOffersThisDevicesFingerprintBelowItsAddress`, `TestPairedDeviceScreenOffersThisDevicesFingerprintBelowTheAddress`, `TestFingerprintViewShowsTheWholeFingerprint`). The card at the end of the device screen is the other device's fingerprint.

Phone layout. The padding that makes controls easy to tap would make every line of text as high as a button, so single lines of text (names, titles, addresses) are laid out compactly (`gui.compact`, a theme override) and controls keep the large size. Long text that wraps is not compacted, because a theme override reports its height for one line. Names are cut with an ellipsis instead of wrapping.

## 2.6 Renaming and one-sided unpairing

- A paired device can be renamed, from the list of paired devices and from its own screen (Rename). The new name shows at once (`TestRenamingAPairedDeviceUpdatesTheList`).
- The home list shows the paired name at once, also after a restart: the last address of each paired device is kept in `paired-addresses.json` and used only to name a device that discovery finds there. The device is not shown as paired until a connection proves it (`TestRenameReachesADeviceSeenByDiscoveryOnly`).
- If one device unpairs and the other still has it paired, the next pairing used to fail with a generic error. The connection from the device that no longer trusts is recognized: a pairing message on a trusted connection is refused and named (`error.peer_wants_to_pair`, `internal/transport`, `TestPairingMessageOnATrustedConnectionIsNamed`). The device that still has the other paired is asked whether to unpair it too, so the pairing can be done again (`askUnpairOnRequest`, `TestPeerAskingToPairAgainIsOfferedAnUnpair`). Nothing is unpaired without the person's answer.

## 2.7 Drop area

The area where files are dropped or chosen keeps room around its text and its button (`dropCard`, `TestDropCardHasRoomAroundItsContent`).

The clipboard can send pictures (screenshots). The Fyne clipboard API reads text only, so each platform reads the picture with its own code (`clipboard_darwin.go`, `clipboard_android.go`, `clipboard_ios.go`; `clipboard_other.go` reports none). A picture is shown with its size, asked for, and sent as a file named `Captura-<date>.png` (or the picture's own format on Android), staged in `<home>/clipboard` (`imageSpec`, `TestClipboardImageBecomesAFileToSend`). Text is sent when there is no picture. Tested on macOS with a picture on the clipboard; Android and iOS are type-checked against the NDK and the iPhone SDK, not run on a device yet (task 059).

## 2.8 Device screen actions

The screen of a paired device offers Send files, Send text, Send clipboard (the picture or text on the clipboard, with the same confirmation as on the send screen) and Receive, then Rename and Unpair.

## 2.9 Sharing from other apps

On Android, what another app shares is taken when the app comes to the foreground and waits on the home screen ("Choose a device to send: 2 files"). Choosing a paired device sends it (`TestSharedFileWaitsForADevice`, `TestSharedTextWaitsForADevice`, `TestChoosingADeviceSendsTheShareOnce`). Unsafe names are replaced (`TestSharedNamesAreSafe`).

## 2.10 Files copied in the file manager

"Send clipboard" sends the files copied in the file manager first, then a picture, then text.
- macOS: the copied files are read from the pasteboard by their paths and sent from where they are (`clipboardPaths`, `previewClipboardPaths`). A folder is refused (`TestCopiedFilesArePlannedAsFiles`).
- Android: the copied files come as content, through the content resolver, and are copied into the application's folder before sending (`clipboardContents`, `previewClipboardContents`).
- iOS: no files are given to other apps through the pasteboard; only pictures and text are sent.
- Windows and Linux: not implemented yet. The clipboard file list (CF_HDROP on Windows; text/uri-list or x-special/gnome-copied-files on Linux) must be read by native code on each system.

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
