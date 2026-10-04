# External Linux Common-Go Verification: 2026-10-05

## Source And Receipt

The owner supplied an external Linux execution receipt for the source-only
handoff archive. This checkout reviewed its bytes, logs and exit-code records;
it did not rerun the Linux commands or execute the receipt's helper scripts.
This is external execution evidence, not a GitHub Actions or signed attestation.

- Implementation baseline: `1e1be215b0cc979d46c0af5e3704a97f273c742d`.
- Source ZIP commit: `5daf7ae3aa58a9c1e19ae11e048083765961a141` (baseline plus
  the handoff prompt).
- Source ZIP SHA-256:
  `f38a62343b3de6f42055ee615f8d6e9cd8bd15721f99cd2abedf3e2434a4f8d9`.
- [Unmodified result ZIP](evidence/linux-common-go-20261005.zip): 147,593 bytes,
  SHA-256
  `21e437b33fe28f27105cdce74a2f870daaeb1fafa251f9e4acd3eab359c807ca`.

Local receipt inspection verified ZIP read/CRC integrity for all 105 files,
104 `SHA256SUMS` entries covering every file except the checksum list itself,
and all 690 source hashes against the original Git archive. Included source
references also matched. The source identity reports no changes or patch.
All 22 command ledger entries matched their individual exit-code files and
had command text and logs. The detailed Go JSON log was independently counted
and contained no failure event.

## Recorded Execution

The environment log identifies Debian 13 (trixie), Linux amd64 and Go 1.26.7,
with `CGO_ENABLED=1`. Commands started on 2026-10-04 at 17:53 UTC (2026-10-05
in Korea). The prior Windows environment used Go 1.27.1; this receipt is not
an equal-toolchain performance comparison.

| Command | Recorded Exit | Result |
| --- | --- | --- |
| `go test -count=1 ./...` | 0 | Root-module tests passed |
| `go vet ./...` | 0 | No diagnostics |
| `go test -count=1 -v ./cmd/velox-consumer-summary` | 0 | Summary tests passed |
| `go test -count=1 -json ./...` | 0 | Detailed results below |
| `go mod verify` | 0 | All modules verified |

The detailed log covers 50 root-module packages: 39 passed test packages and
11 packages without tests. Top-level test/fuzz items: 323 PASS, 2 SKIP, 0 FAIL.
Including subtests and fuzz seeds: 514 PASS and 2 SKIP. Parent and child events
are counted together in the latter count, not as independent top-level tests.

The two runtime skips match existing source conditions:

- `TestContainedPathRejectsWindowsDriveRelativePath`: non-Windows platform.
- `TestDryRunSchemaFixture`: `VELOX_SIGNING_RECORD_RESULT` not configured.

Actual summary CLI invocations recorded exit 0 and a summary for valid input;
expected samples 2 versus observed 1 recorded exit 1 and `missingCount=1`.
Malformed JSON, duplicate IDs and schema violations returned 1 without a
summary. Failed samples, mixed acquisition archive hashes, hosted fail and
hosted unverified states returned 1 with diagnostic summaries. Hosted pass and
local unverified states returned 0. Successful stdout JSON matched its saved
output. All inputs are synthetic; their Windows metadata, hashes and duration
values are fixtures, not Windows measurements or hosted evidence.

## Scope And Follow-Up

No application source, dependency, API, DB or CI change was needed. Local
review checked the existing skip conditions and did not repeat the tests.
The preserved receipt includes the full external report, logs, source hash
manifest, command ledger, and synthetic inputs/outputs.

Windows native behavior, the separate `third_party/go-webview2` module's tests,
race/extended fuzz runs, other Linux distributions/architectures and a separate
`CGO_ENABLED=0` run remain outside this receipt. Linux application support is
unchanged. Alpha remains active; beta is not approved. No tag, release,
publication, signing or remote push accompanies this record.
