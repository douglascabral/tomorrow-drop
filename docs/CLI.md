# Command-line interface

Tasks 037 to 044 (block 05). Implementation in `cmd/tdrop`. The CLI uses the same core as the GUI (`internal/*`).

## Commands

| Command | What it does |
|---|---|
| `tdrop identity show` | Shows this device's identity and full fingerprint. Creates the identity on first use. The private key is never printed. |
| `tdrop discover [--target IP:port]...` | Lists devices that answer discovery, by name and address. `--target` queries explicit addresses, for networks where broadcast is blocked. |
| `tdrop pair <destination> [--name NAME]` | Pairs with a device by IP:port or name. Both people compare the fingerprints and confirm. |
| `tdrop trust list` | Lists trusted devices with their identifiers. |
| `tdrop trust remove <id>` | Revokes trust. `<id>` is the identifier from `trust list` or a unique prefix of it. |
| `tdrop send <file>... --to <destination>` | Sends files to a trusted device. |
| `tdrop send --stdin --to <destination>` | Sends the text read from standard input. |
| `tdrop receive [--listen ADDR] [--out DIR] [--name NAME]` | Waits for one connection, pairs or asks to accept each offer, then exits. |
| `tdrop daemon [--listen ADDR] [--out DIR] [--name NAME] [--auto-accept-trusted]` | Same as `receive`, but keeps running until interrupted. |
| `tdrop version` | Prints the version and the protocol version. |
| `tdrop config language [auto\|pt-BR\|en]` | Shows or sets the language. |

Defaults: TLS listener on `0.0.0.0:47521`; discovery on UDP `47520`; received files go to the Downloads folder.

`<destination>` is an `IP:port` (IPv4 or `[IPv6]:port`) or the name of a device found by discovery. Names are looked up with a two-second scan.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Error (network, storage, protocol) |
| 2 | Usage error |
| 3 | Refused: the transfer was declined, the recipient was busy, the device is not trusted, or pairing was not confirmed |
| 130 | Canceled (Ctrl-C) |

## Output

- Results go to standard output.
- Progress and errors go to standard error.
- All text comes from the translation files and follows the language setting.

## Authorization (tasks 042 and 043)

- **`receive`** asks for every offer, including offers from trusted devices. The answer must be `y` or `s`. Anything else, including end of input, declines.
- **`daemon`** asks by default, like `receive`.
- **`daemon --auto-accept-trusted`** is the only non-interactive policy. It accepts offers from trusted devices without asking. It never accepts offers from unknown devices: those always ask. Every automatic acceptance is printed. Without this flag, nothing is accepted automatically.
- Pairing always asks. No device becomes trusted without both sides confirming the fingerprints.

## Headless use

Prompts read from standard input. With no terminal attached, redirect answers (for example `echo y | tdrop receive`) or use `--auto-accept-trusted` for a daemon. An interrupted command (Ctrl-C) closes its sessions and listeners, and the other side discards partial data.

## Tests

`cmd/tdrop/integration_test.go` runs two simulated devices in one process over loopback TCP and UDP. It covers pairing, file and text transfer, decline, untrusted sends, the daemon policy, discovery, cancellation and exit codes.
