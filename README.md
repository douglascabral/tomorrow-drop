# Tomorrow Drop

**Transferiu. Usou. Pronto.** (roughly "Transferred. Used. Done.")

Free, minimalist application to transfer files, texts, links and clipboard content between devices on the same local network. It uses no accounts, cloud, central servers or internet connection.

> **Status:** identity, trust and pairing implemented (blocks 01 and 02). Discovery, transfer and interfaces are not implemented yet. See [TOMORROW_DROP_TASKS.md](TOMORROW_DROP_TASKS.md).

## Documentation

- [PRD v1.9](docs/PRD.md): requirements, product decisions and acceptance criteria.
- [Build](docs/BUILD.md): build for Linux, Windows and macOS.
- [Protocol](docs/PROTOCOL.md): framing, version, states, limits and timeouts.
- [Discovery and connectivity](docs/DISCOVERY.md): UDP datagrams, rate limits, query schedule, device list and manual connection.
- [User guide](docs/USER_GUIDE.md): installation, pairing, sending, receiving, the 30-second rule and the limits.
- [Threat model](docs/THREAT_MODEL.md) and [security review](docs/SECURITY_REVIEW.md).
- [Performance](docs/PERFORMANCE.md), [dependencies](docs/DEPENDENCIES.md) and [acceptance checklist](docs/ACCEPTANCE.md).
- [Packaging](docs/PACKAGING.md): macOS bundle, metadata and what is still missing.
- [Mobile](docs/MOBILE.md): Android and iOS/iPadOS requirements and verification status.
- [Graphical application](docs/GUI.md): screens, availability with the window, safety rules and known limits.
- [Command-line interface](docs/CLI.md): commands, exit codes, authorization rules and headless use.
- [Transfers](docs/TRANSFERS.md): offers, decisions, staging, all-or-nothing publishing and failure handling.
- [Identity and trust](docs/IDENTITY.md): keys, fingerprint, TLS, pairing and message policy.
- [Languages](docs/LANGUAGES.md): pt-BR as the initial language, English as fallback, system detection and user choice.
- [Core standards](docs/STANDARDS.md): errors, logging, cancellation and tests.
- [Quality](docs/QUALITY.md): checks and tools.
- [Third-party notices](THIRD_PARTY_NOTICES.md): dependencies and licenses.

## Structure

```
cmd/tdrop/            CLI
cmd/tomorrow-drop/    graphical application (Fyne, to be integrated in task 045)
internal/             core shared by the GUI and the CLI
assets/               visual resources
docs/                 PRD and technical documentation
scripts/              quality and security check automation
```

Core packages: `identity`, `trust`, `discovery`, `transport`, `protocol`, `transfer`, `storage`, `lifecycle`, `ui`, plus `apperr`, `safelog`, `limits`, `i18n` and `settings` (shared base).

Architecture rule: the core never imports Fyne. Platform-specific integrations stay behind interfaces and build tags.

## Development

Requirements: Go 1.24 or later.

The `Makefile` wraps the common tasks. Run `make help` to list them:

```sh
make cli       # bin/tdrop, without cgo
make test      # go test ./...
make check     # the full validation required by AGENTS.md
```

The plain commands also work:

```sh
go test ./...
go test -race ./...
```

To run all quality and security checks, see [docs/QUALITY.md](docs/QUALITY.md).

## License

[MIT](LICENSE).

## Command-line tutorial (`tdrop`)

`tdrop` sends and receives files and text on the local network from a terminal. It needs no account, and both devices must be on the same network.

### 1. Build it

```sh
make cli            # creates bin/tdrop
bin/tdrop help      # lists the commands
```

### 2. Know your device

```sh
bin/tdrop identity show
```

This prints the full fingerprint of this device. The private key is never shown. The identity is created the first time you run it.

### 3. Find the other device

On the other computer, start listening (see step 5 for the difference between the two modes):

```sh
bin/tdrop daemon
```

Then, on this computer:

```sh
bin/tdrop discover
```

You get a list with each device's name and address. If the network blocks broadcast, query the address directly:

```sh
bin/tdrop discover --target 192.168.1.42:47521
```

### 4. Pair the two devices (once)

Pairing makes a device trusted. Run this on the device that starts it:

```sh
bin/tdrop pair 192.168.1.42:47521 --name "Living room laptop"
```

Both computers show their fingerprints. Compare every character on both screens. Answer `s` on both only if they match; anything else (including just pressing Enter) cancels, and nothing is saved. The `--name` is how you will recognise the device later.

Pairing can also start from the other side: a device that is not paired yet and connects to a listening `receive` or `daemon` is asked the same question.

### 5. Send files and text

Sending needs a paired device. Use its name from `trust list`, or its address:

```sh
bin/tdrop send report.pdf photo.jpg --to "Living room laptop"
bin/tdrop send notes.txt --to 192.168.1.42:47521
echo "a link to share" | bin/tdrop send --stdin --to "Living room laptop"
```

The receiving device must be running `receive` or `daemon`. Each offer is shown with who sends it, how many files, their sizes and their names, and asks `Accept? [y/N]`.

### 6. Receive

```sh
bin/tdrop receive                 # waits for one transfer, then exits
bin/tdrop daemon                  # keeps running until you press Ctrl-C
bin/tdrop daemon --out ~/Inbox    # saves received files in another folder
```

By default, received files go to the Downloads folder. Nothing is accepted without your answer, not even from paired devices.

### 7. Manage trusted devices

```sh
bin/tdrop trust list              # paired devices and their identifiers
bin/tdrop trust remove 3f9a       # unpair; a unique prefix of the identifier is enough
```

### Automation

`daemon --auto-accept-trusted` accepts offers from paired devices without asking. Offers from unknown devices still ask. Every automatic acceptance is printed. Use it only on a network you control.

### Options and exit codes

- Options can go before, between or after the arguments; `--` ends the options.
- `--name` (pairing and listening) sets the name this device announces.
- `--listen ADDR` changes the address the listener uses (default `0.0.0.0:47521`).
- Discovery uses UDP port `47520`. Allow both ports (`47520` and `47521`) through the firewall.

| Exit code | Meaning |
|---|---|
| 0 | Success |
| 1 | Error (network, storage or protocol) |
| 2 | Usage error: the command or its arguments are wrong, and the usage of that command is printed |
| 3 | Refused: declined, busy, not trusted, or pairing not confirmed |
| 130 | Cancelled with Ctrl-C |

For the full reference, see [docs/CLI.md](docs/CLI.md).
