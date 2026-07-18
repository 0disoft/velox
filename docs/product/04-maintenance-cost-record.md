# Maintenance Cost Record

- Status: Current bounded snapshot for M5 input
- Owner: Project maintainer
- Snapshot: `velox.maintenance-cost/v1`
- Machine-readable record: `docs/product/maintenance-cost-v1.json`

## Question

M5 must decide whether Velox's narrow product value justifies the surface that
the maintainer has to keep working. Commit count, source lines, workflow jobs,
and configured timeout ceilings are imperfect proxies, but they are observable.
Invented person-hours are not.

This record therefore answers a bounded question: how much repository and
recurring CI surface existed from the first complete implementation commit
through the first selected public-preview candidate?

## Observation Window

- Start: `cf3b6e38fd5c7322113e3c49f16743a474582671`, 2026-07-12.
- End: `7d39d89b9ecfff339518b065ba78a13d69737160`, 2026-07-18.
- Included commits: 42, including both endpoints.
- Aggregate diff from the initial scaffold parent: 271 changed files and
  29,416 inserted lines.

The zero deletion count is not a low-churn signal. The comparison starts from
the initial scaffold parent, so files retired during the window can still be
net additions relative to that parent.

## Repository Surface

| Surface | Files | Lines | Boundary |
| --- | ---: | ---: | --- |
| Project-owned production Go | 46 | 6,909 | `cmd/` and `internal/`, excluding `_test.go` |
| Go tests | 39 | 5,145 | `_test.go` under `cmd/`, `internal/`, and `tests/` |
| Vendored Go binding | 42 | 3,859 | `third_party/`, reported separately |
| GitHub Actions workflows | 4 | 825 | `.github/workflows/*.yml` |
| JSON contracts | 20 | not used | `schema/*.json` |
| Markdown documentation | 43 | not used | `docs/**/*.md` |

Line counts use PowerShell `Get-Content | Measure-Object -Line` over the named
snapshot files. They are inventory, not a quality score.

The module graph has two direct dependencies and one indirect dependency. The
product supports one target, seven public CLI commands, and six native IPC
methods. That is a narrow feature boundary carried by a larger-than-feature-
count implementation and evidence surface.

## Recurring CI Ceiling

The only scheduled product workflow runs weekly. Its scheduled path creates
one release job, ten isolated consumer jobs, and one summary job on
`windows-2025`. The completed workflow then creates one bounded warning-monitor
job on `ubuntu-24.04`.

| Weekly scheduled work | Jobs | Configured timeout ceiling |
| --- | ---: | ---: |
| Windows | 12 | 63 job-minutes |
| Ubuntu | 1 | 5 job-minutes |

These are worst-case configured ceilings, not observed duration or a billing
claim. Manual alpha, public-download, and full diagnostic dispatches add work
only when a maintainer explicitly requests them. Consumer workflows continue
to upload zero Actions cache bytes.

## Manual Preview Work

The first public preview requires three maintainer transitions:

1. Push the immutable alpha tag.
2. Dispatch the unsigned-preview publication gate with its exact confirmation.
3. Dispatch the public-download verifier with an independently recorded ZIP
   SHA-256.

An independent external-user attempt is deliberately not counted as maintainer
automation. It is product evidence that cannot be manufactured by this
repository.

## M5 Interpretation

The maintenance burden is not tiny. In seven days the project accumulated 42
implementation and evidence commits, 6,909 production Go lines, 5,145 Go test
lines, a 3,859-line maintained binding fork, four workflows, and 20 JSON
contracts for a one-target, seven-command, six-method product.

That cost can be justified only if external users value the compile-free,
portable, deterministic artifact boundary enough to outweigh a PWA or an
existing compile-free wrapper. M5 must not read fast consumer builds as proof
that the framework itself is cheap to maintain.

## Refresh Rule

Create a new immutable record rather than rewriting this observation window
when any of these occur:

- M5 starts;
- a second target or native capability is proposed;
- the maintained WebView2 fork changes materially;
- scheduled CI shape changes;
- beta or stable promotion is considered.

The next record must name its own commit window and preserve this one for
comparison.
