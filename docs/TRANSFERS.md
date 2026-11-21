# Transfers

Tasks 025 to 036 (block 04). Implementation in `internal/transfer` and the transfer message types in `internal/protocol`.

## 1. Who may transfer

Transfers run only on trusted sessions (PRD §8.1). The policy in `protocol.AllowedIn` allows the transfer message types only in the `trusted` state, and `Receiver.Serve` refuses any other session. A provisional connection can carry pairing messages and nothing else.

## 2. Messages

All transfer messages are control frames (JSON, at most 64 KiB). File data travels in `TRANSFER_DATA` messages of at most 32 KiB of raw content, so the base64 encoding stays under the frame limit.

| Message | Direction | Body | Meaning |
|---|---|---|---|
| `TRANSFER_OFFER` | sender → recipient | `id`, `kind`, `items[name,size]` | Describes the content |
| `TRANSFER_ACCEPT` | recipient → sender | `id` | The recipient accepted |
| `TRANSFER_DECLINE` | recipient → sender | `id`, `reason` | Declined: `declined`, `busy` or `invalid` |
| `TRANSFER_DATA` | sender → recipient | `id`, `index`, `data` | One chunk of an item |
| `TRANSFER_END` | sender → recipient | `id`, `index`, `size`, `sha256` | Item finished, with its size and SHA-256 |
| `TRANSFER_DONE` | sender → recipient | `id` | Every item was sent |
| `TRANSFER_RESULT` | recipient → sender | `id`, `ok`, `reason` | Published, or failed: `integrity`, `size`, `canceled` |
| `TRANSFER_CANCEL` | either | `id` | Explicit cancellation |

`kind` is `file`, `text` or `link`. Text and links carry exactly one item with no name.

## 3. Offer rules (task 026)

- File names must be plain names: no `/`, `\` or `:`, no control characters, not `.` or `..`, not starting with `.`, at most 255 bytes.
- At most 1000 files per offer; 16 GiB per file; 64 GiB per offer (`limits`).
- Text is at most 1 MiB and must be valid UTF-8.
- Links must be absolute `http` or `https` URLs. Other schemes (`file`, `javascript`, `mailto`, ...) are refused.
- Invalid offers are declined before anything is written.

## 4. Decision (task 027)

Every offer goes to the `Decision` function, including offers from trusted devices. The decision has `DecisionTimeout` (60 s) to answer; without an answer the offer is declined automatically. Nothing is saved before acceptance.

## 5. Staging and publishing (tasks 031 and 032)

- Data is written to temporary files created with `os.CreateTemp` in the staging directory. Names are random and match `tdrop-*.part`.
- Each item is checked against its declared size and its SHA-256, computed while the data arrives.
- Files are published only after `TRANSFER_DONE` and only when every item is intact. Publishing uses `os.Link`, which is atomic and never overwrites: a name clash produces `name (1).ext`, `name (2).ext`, and so on.
- Text and links are read from staging into memory, validated, and the staging file is removed. Nothing is persisted (PRD §21).

## 6. Partial failures (task 030)

Transfers are all-or-nothing. The sender opens every file before sending the offer, so a missing or unreadable file stops the transfer before the recipient is involved. If the transfer fails during sending, or one publish fails, nothing is kept: staged files are removed and the recipient publishes nothing. There is no resume (PRD §16): a new attempt starts from the first byte.

## 7. Cancellation, errors, timeouts and drops (tasks 025 and 033)

- Cancelling the context of `Send` or `Serve` closes the session. The other side sees the connection end and discards its staging data.
- Waiting for an answer to the offer uses `DecisionTimeout`; waiting for data or the result uses `IdleTimeout` (30 s) per message (`transport.Session.ReceiveWithin`).
- A timeout surfaces as `error.timeout`. A checksum mismatch is `error.integrity`. A declined offer is `error.declined`. A busy recipient is `error.busy`.
- A `Machine` enforces the lifecycle `idle → offered → accepted → transferring → finishing → done`. Declined, done, failed and canceled are terminal.

## 8. Concurrency (task 025)

The `Receiver` admits at most `MaxConcurrent` transfers (default 4, `limits.MaxConcurrentTransfers`). An extra offer receives `TRANSFER_DECLINE` with reason `busy`.

## 9. Abandoned staging files (task 034)

`NewReceiver` removes files matching `tdrop-*.part` that are older than one hour in the staging directory. Other files are never touched.

## 10. Destinations (task 035)

`transfer.Destination` is the storage abstraction. `DesktopDestination` saves into a folder; `DefaultDownloadsDir` returns the user's Downloads folder. Android and iOS will implement the same interface with native storage (tasks 035 and 058, block 07).

## 11. Limits of this block

- Hard links (`os.Link`) are required for atomic publishing. Filesystems that do not support them (some FAT and exFAT volumes) fail with an explicit error instead of falling back to a non-atomic copy.
- The transfer uses base64 inside JSON for data chunks: simple and fully covered by the same policy, but about a third more traffic than raw binary frames. A raw binary frame type is a possible future optimization that would require a protocol version change.
