# Security review: keys, logs, temporary files, errors and overwrites

Task 064. Each item below names what the code does and the test that checks it. A check that was not run is marked as such.

## Private key storage

- Generated with `crypto/rand` and stored as PKCS#8 PEM in `identity.key`. The directory is created with `0700` and the file with `0600` on POSIX systems (`TestCreateAppliesRestrictivePermissions`).
- The key has no public accessor. Formatting (`%v`, `%+v`, `%#v`) never shows it (`TestFormattingDoesNotRevealPrivateKey`).
- A corrupted or foreign key file is rejected and never silently replaced (`TestLoadOrCreateDoesNotOverwriteCorruptedFile`).
- **Gap:** on Windows, Go's permission bits do not control ACLs. The key is protected only by the default ACLs of the user profile. This must be reviewed on Windows before release; it is recorded in `docs/IDENTIDADE.md`.

## Logs

- The `safelog` package redacts fields named as private, secret, token, password, key, content, clipboard, payload, filename and path, and any `[]byte` value (`internal/safelog` tests).
- The CLI and the GUI print fingerprints (public) and never print keys or transferred content. The only text printed for a received text is the text the person chose to receive, shown on the screen of the recipient.
- Error messages shown to people are translations of error keys. Internal messages are not shown (task 004 and `docs/PADROES.md`).

## Temporary files

- Staging files are created with `os.CreateTemp` (unpredictable names) in a directory of the application, and only files matching `tdrop-*.part` older than an hour are removed at startup (`TestCleanupAbandonedRemovesOnlyOldStagingFiles`). Files of other programs are never touched.
- Failed or interrupted transfers remove their staging files (`TestSenderDropAt99PercentLeavesNothingBehind`, `TestCorruptedHashIsRejectedAndNothingPublished`).

## Error handling

- Errors carry sentinels (`internal/apperr`) and are checked with `errors.Is`. Network and storage details stay in the error chain for diagnosis, and are never shown as user text.
- Timeouts are reported as timeouts, not as corrupt data (`TestReadFrameTimeoutIsNotMalformed`). This was a real defect found during this block.

## Overwrites

- Received files are published with `os.Link`, which never overwrites: a name clash produces `name (n).ext` (`TestPublishExclusiveNeverOverwrites`).
- Key replacement is an explicit call (`Store.Replace`) that the interface must confirm. Creation refuses to overwrite (`TestCreateDoesNotOverwriteExistingIdentity`).
- The trust list is rewritten atomically (`storage.WriteFileAtomic`).

## Fuzzing (task 063)

Eight fuzz targets exercise the untrusted-input parsers. Each ran for 15 seconds on the development machine (Apple M4, macOS), with no crash and no accepted invalid input:

| Target | Package | Executions |
|---|---|---|
| `FuzzReadFrame` | protocol | 4.36 million |
| `FuzzDecodeMessage` | protocol | 3.91 million |
| `FuzzDecodeDatagram` | discovery | 0.14 million |
| `FuzzParseEndpoint` | discovery | 3.17 million |
| `FuzzValidateName` | transfer | 2.33 million |
| `FuzzValidateLink` | transfer | 0.11 million |
| `FuzzDecodeIdentity` | identity | 3.42 million |
| `FuzzTrustFile` | trust | 0.07 million |

The seed corpus of each target runs in the normal test suite. Longer runs are recommended before a release:

```sh
go test -run='^$' -fuzz='^FuzzReadFrame$' -fuzztime=10m ./internal/protocol/
```

## Not yet reviewed

- A review by a second person, and the security review that PRD §24.6 requires before a stable release.
- The Android and iOS storage and permissions (task 058 and block 07).
