# User guide

Task 068. Installation, pairing, sending and receiving, and the limits of the application.

## Installation

- **CLI (all desktop systems):** build from source with Go 1.24 or later. See `docs/BUILD.md`.
- **macOS application:** the release has `TomorrowDrop-macos-arm64.dmg` (Apple silicon) and `TomorrowDrop-macos-amd64.dmg` (Intel). Open it and drag Tomorrow Drop to Applications. A release signed with a Developer ID and notarized opens normally; one signed ad hoc (the log of the release says which) makes macOS ask you to confirm opening it (Control-click, Open). `make package-macos` builds them from source. See `docs/PACKAGING.md`.
- **Linux, Windows, Android and iOS/iPadOS:** no packages yet. See `docs/PACKAGING.md` and `docs/MOBILE.md`.

## Pairing a device (first time only)

1. Open the application on both devices, on the same network.
2. Pick the other device in the list, or type its IP:port in the manual connection field.
3. Choose **Pair**. Both screens show two complete fingerprints, one per device.
4. Compare them character by character on both devices. Confirm only if every character matches. If anything differs, cancel: nothing is stored.
5. Name the other device. It appears in the trusted devices list.

Trust stays until you remove it under **Trusted**. There is no automatic expiry.

## Sending

- Choose the recipient from the device list or type its name or IP:port.
- Add files, write text or a link, or use **Send clipboard**. The clipboard is read only when you press that button, and you see a preview before anything is sent.
- Only trusted devices can receive. If the recipient is not trusted, the application offers to pair first.
- You can cancel a transfer while it runs.

## Receiving

- Every incoming request asks you to accept or decline, including requests from trusted devices. Nothing is saved before you accept.
- Accepted files go to your **Downloads** folder. A name already in use gets a number: `photo (1).jpg`. Existing files are never overwritten.
- Received text is shown on screen and is not saved anywhere.

## Discovery and the 30-second rule

- While the window is in the foreground, the application looks for devices at once when it opens, when you press **Refresh**, and every 30 seconds after that.
- **Refresh** can be pressed again only after two seconds, to avoid flooding the network.
- During a transfer, the periodic search pauses.
- Devices that stop answering disappear from the list after 90 seconds.

## Minimized and background

- On desktop, minimizing stops the search and stops accepting new connections. Transfers already accepted keep running until they finish, fail or are cancelled.
- Closing the application cancels everything.
- On Android and iOS/iPadOS, leaving the application stops the search, closes the listener and cancels any transfer in progress. The application does not run in the background. (Not yet verified on devices.)

## Limits

- Interrupted transfers cannot be resumed. A new attempt starts from the first byte.
- A single file may be up to 16 GiB; one transfer may contain up to 1000 files and 64 GiB in total.
- Text is limited to 1 MiB. Links must be `http` or `https`.
- Up to four transfers run at the same time per device. Additional requests are declined as busy.
- Clipboard images are not sent yet: the toolkit exposes text only.
- Trust must be removed on each device separately.
- The application does not reach devices on other networks or the internet, by design.
