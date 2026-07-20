# validation-answer

Normalizes qualified signature-validation reports (EU DSS behind a signing
API) into **one stable, wire-ready answer shape** — so every service and UI
renders the same fields from one place, instead of hand-copying the answer
struct per consumer.

```
go get github.com/gmb-lib/go-validation-answer
```

Zero dependencies (standard library only). Import path package name: `answer`.

## Why

The upstream validation report is relayed **verbatim** by the validation
service, and its layout varies across API versions: the signer identity may be
nested under `signerExt` or sit at the signature entry's top level; timestamps
may be flat (`timeStamp`, `ocspResponceTime` — upstream spelling) or nested
under `info` (`bestSignatureTime`, `ocspResponseCreationTime`); the
included-file list may be `["a.pdf"]` or `[{"filename":"a.pdf"}]`, under
`validatedDocument` or at the data level. `NormalizeReport` reads the known
layouts with fallback resolution and produces one answer.

## The answer (also the JSON wire contract)

```go
res, err := answer.NormalizeReport(rawReport) // the verbatim {data:{…}} bytes
```

`answer.Validation` — document-level verdict (`PASSED` / `INDETERMINATE` /
`FAILED`), `pass`, container form, signed-file names, and `signatures[]`
(`answer.Signature`: per-signature verdict, format profile, legal-meaning
level, signer identity, signing / revocation / max-validity times, warnings,
errors). The top-level per-signer fields mirror the **first** signature for
single-signature callers. `signatureId` / `documentId` / `reportId` /
`validatedAt` are caller context, set by the serving side — `validatedAt` is
when the validation actually ran (RFC 3339): validation is time-anchored
(revocation can post-date it), so an answer served later than it was produced
renders "as of" that moment, never as current.

Rules baked in:

- **Verdicts are mapped, never recomputed**: `TOTAL-PASSED`/`TOTAL-FAILED`
  map directly; anything else is `INDETERMINATE`. The overall verdict fails on
  any failed signature and passes only when every signature passed **and** the
  report's own counts agree.
- **Levels map to legal meaning** (`QESIG`→`QES`, `QESEAL`→`SEAL`,
  `ADESIG*`/`ADESEAL*`→`AdES`); unknown codes pass through visibly rather than
  being guessed.
- An organisation's registration number is preferred over a bare serial; a
  single PDF's placeholder "file list" is dropped rather than surfaced.
- Timestamps pass through as RFC 3339 (a legacy locale format is converted).
- In-process extras (`SignaturesCount`, `ValidSignaturesCount`,
  `ValidationLevel` — e.g. `ARCHIVAL_DATA` after an archive timestamp) are
  available on the struct but deliberately **off the wire**.

The wire key set is pinned by a test (`TestWireContractKeys`) — changing it is
an explicit, reviewed act, never a side effect.

## Scope / non-goals

- No validation calls, no HTTP — bytes in, answer out.
- No cryptographic interpretation; presence/verdict mapping only.
- The verbatim report itself is not reshaped or re-served by this package.
