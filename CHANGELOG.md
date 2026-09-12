# Changelog

Notable changes to this library, newest first. Versions are git tags; this file is written
for whoever bumps the dependency — what changed, and what it means for code that already
uses it.

## v1.1.2

No library code changed since v1.1.1, so the answer shape and the JSON wire contract are exactly
what they were. There is one thing to act on before you bump: **it now needs Go 1.27**.

### Changed

- **The module declares `go 1.27.0`** (was `1.26.6`), so your own module has to be on Go 1.27
  before it can build against this one. A dependency's `go` line does **not** make the go command
  fetch a newer toolchain for you — measured both ways: a consumer whose own `go` directive is
  lower stops with a `requires go >= 1.27.0 (running go 1.26.6)` error, and it stops there with
  `GOTOOLCHAIN` on its `auto` default just as it does under `local`. Raise your own `go` directive
  to `1.27.0` first; from there the go command downloads and uses the 1.27 toolchain by itself, so
  nobody has to install Go by hand. CI that reads `go-version-file: go.mod` follows the bump with
  no workflow edit — a workflow naming a Go version in the YAML needs that line changed.

  This library still has **no module dependencies at all**: the standard library is the whole
  dependency surface, as it has been since v1.0.0.

### Notes

- The gate is green on Go 1.27: `go mod verify`, `go mod tidy -diff`, build, vet, `gofmt`, and
  `go test -race` with **0 races**; `govulncheck` finds nothing.

---

The entries below were **reconstructed from git history** rather than written at the time, so they
say what each tag contains, not why it was decided. They cover what a consumer would have to act on;
releases that only moved build plumbing are named as such.

## v1.1.1

Housekeeping. No library code changed, so a bump from v1.1.0 asks nothing of you — the `go`
directive moved to 1.26.6, the CI workflows were reworked, and the repository gained the
open-source kit it was missing (`SECURITY.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, a
secret-scan configuration and the README sections pointing at them). The advisory DCO workflow was
removed now that the sign-off is enforced by the organisation's app together with a branch ruleset;
what a contribution has to carry is unchanged.

One thing worth a line for anyone reading the tests: the identity codes and the organisation
registration number in them are **assembled at test time rather than written as literals**, so a
secret scanner has no personal-number-shaped string to flag. The values they stand for are
unchanged.

The copyright holder was also named in full — `SIA "Go Make Bytes"` instead of `go-make-bytes`.
Same MIT licence, same terms; only the holder's legal name is spelled out.

## v1.1.0

Additive on the wire: a consumer that ignores unknown fields is unaffected, and nothing existing
moved or changed meaning.

### Added

- **`Validation.ValidatedAt`** (`validatedAt` on the wire, RFC 3339, omitted when empty) — when the
  validation actually ran. Validation is time-anchored: revocation can post-date it, so an answer
  served later than it was produced has to be presented *as of* that moment and never as current.
  The field is set by the serving side, like `SignatureID`, `DocumentID` and `ReportID` —
  `NormalizeReport` does not fill it.

## v1.0.0

Initial code. `NormalizeReport` reads the verbatim `{data:{…}}` report a qualified validation
service relays and produces one stable answer shape, resolving the layouts that vary across API
versions — signer identity nested under `signerExt` or flat, timestamps flat (`timeStamp`,
`ocspResponceTime` — upstream spelling) or nested under `info`, the included-file list as strings
or as objects, under `validatedDocument` or at the data level. Standard library only; the import
path's package name is `answer`.
