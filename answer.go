// Package answer normalizes signature-validation reports into one stable,
// wire-ready answer shape.
//
// A qualified validation service (EU DSS behind a signing API) relays a rich,
// verbatim validation report whose field layout varies between API versions —
// the signer identity may sit under a signerExt sub-object or at the entry's
// top level, timestamps may be flat or nested under an info object, and the
// included-file list may be strings or objects. This package reads the
// variants it knows and maps them onto ONE normalized answer (Validation),
// which is also the JSON wire contract every consumer serves and renders — so
// the answer shape lives in exactly one place instead of being hand-copied
// per service.
//
// The package interprets nothing cryptographic: verdicts are mapped from the
// report's indications, never recomputed.
package answer

// The normalized verdict vocabulary. The provider relays the upstream report
// verbatim; NormalizeReport maps it onto this stable set so consumers never
// parse the raw report.
const (
	VerdictPassed        = "PASSED"
	VerdictIndeterminate = "INDETERMINATE"
	VerdictFailed        = "FAILED"
)

// Signature is the normalized answer for one signature within a signed
// document — a container can hold several (parallel co-signatures). Each
// carries its own party, profile, level, times, and verdict so every
// signature is rendered, not just the first. The identity fields
// (Signer/SignerSerial/Organization) are surfaced to the owner on request;
// keep them out of lean audit events.
type Signature struct {
	Verdict         string   `json:"verdict"`
	Format          string   `json:"format,omitempty"` // signature profile, e.g. XAdES_BASELINE_LT
	Level           string   `json:"level,omitempty"`  // legal-meaning label, e.g. QES
	Signer          string   `json:"signer,omitempty"`
	SignerSerial    string   `json:"signerSerial,omitempty"`
	Organization    string   `json:"organization,omitempty"`
	SigningTime     string   `json:"signingTime,omitempty"`     // when this signature was created (RFC 3339)
	RevocationTime  string   `json:"revocationTime,omitempty"`  // revocation-data (OCSP) time (RFC 3339)
	MaxValidityTime string   `json:"maxValidityTime,omitempty"` // long-term-validity horizon; may be empty
	Warnings        []string `json:"warnings,omitempty"`
	Errors          []string `json:"errors,omitempty"`
}

// Validation is the normalized validation answer for one signed document, and
// the JSON wire shape consumers exchange. The container-level fields
// (Verdict, ContainerForm, SignedFiles, Pass) describe the whole document;
// Signatures lists every signature it holds. The top-level per-signer fields
// mirror the FIRST signature for single-signature callers — Signatures is the
// full set.
//
// SignatureID / DocumentID are caller context (which recorded signature or
// which document was validated — at most one is set) and ReportID is the
// persisted-answer id; all three are set by the serving side, not by
// NormalizeReport.
type Validation struct {
	SignatureID     string      `json:"signatureId,omitempty"`
	DocumentID      string      `json:"documentId,omitempty"`
	Verdict         string      `json:"verdict"`
	Format          string      `json:"format,omitempty"`
	Level           string      `json:"level,omitempty"`
	Signer          string      `json:"signer,omitempty"`
	SignerSerial    string      `json:"signerSerial,omitempty"`
	Organization    string      `json:"organization,omitempty"`
	ContainerForm   string      `json:"containerForm,omitempty"` // "PDF" | "ASiC-E"
	SigningTime     string      `json:"signingTime,omitempty"`
	RevocationTime  string      `json:"revocationTime,omitempty"`
	MaxValidityTime string      `json:"maxValidityTime,omitempty"`
	SignedFiles     []string    `json:"signedFiles,omitempty"` // container data-object names
	Warnings        []string    `json:"warnings,omitempty"`
	Errors          []string    `json:"errors,omitempty"`
	Signatures      []Signature `json:"signatures,omitempty"` // every signature, in report order
	Pass            bool        `json:"pass"`
	ReportID        string      `json:"reportId,omitempty"`

	// In-process fields, deliberately NOT on the wire (kept off it so the wire
	// contract stays exactly what consumers already exchange).

	// SignaturesCount / ValidSignaturesCount are the report's own totals,
	// used to derive the overall verdict.
	SignaturesCount      int `json:"-"`
	ValidSignaturesCount int `json:"-"`
	// ValidationLevel is the depth the validator reported (e.g. ARCHIVAL_DATA
	// for an archive-timestamped document). Captured for consumers that want
	// to surface preservation state; not yet part of the wire contract.
	ValidationLevel string `json:"-"`
}
