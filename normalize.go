package answer

import (
	"encoding/json"
	"fmt"
	"time"
)

// providerReport is the slice of the verbatim validation report this package
// reads. The provider hands back a top-level {data:{…}} envelope; only the
// fields the normalized answer needs are mapped, keeping the original
// upstream field names/casing exactly (the report shape is the provider's).
// Where API versions lay the same fact out differently, both layouts are
// declared and the fallback order is resolved in NormalizeReport.
type providerReport struct {
	Data providerReportData `json:"data"`
}

type providerReportData struct {
	SignatureForm        string              `json:"signatureForm"` // "PDF" | "ASiC-E"
	ValidationLevel      string              `json:"validationLevel"`
	SignaturesExt        []providerSignature `json:"signaturesExt"`
	SignaturesCount      int                 `json:"signaturesCount"`
	ValidSignaturesCount int                 `json:"validSignaturesCount"`
	ValidatedDocument    validatedDocument   `json:"validatedDocument"`
	// Older report layouts list the included files at the data level, as
	// objects, rather than under validatedDocument.
	IncludedFiles fileList `json:"includedFiles"`
}

// validatedDocument names the document that was validated and the files it
// contains. For a container the included files are the real data-object
// names; for a single PDF the provider returns a placeholder instead.
type validatedDocument struct {
	FileName      string   `json:"fileName"`
	IncludedFiles fileList `json:"includedFiles"`
}

// fileList reads an included-file list in either upstream layout: a plain
// string array, or an array of {filename} objects.
type fileList []string

func (f *fileList) UnmarshalJSON(b []byte) error {
	var plain []string
	if err := json.Unmarshal(b, &plain); err == nil {
		*f = plain
		return nil
	}
	var objs []struct {
		FileName string `json:"filename"`
	}
	if err := json.Unmarshal(b, &objs); err != nil {
		return err
	}
	names := make([]string, 0, len(objs))
	for _, o := range objs {
		if o.FileName != "" {
			names = append(names, o.FileName)
		}
	}
	*f = names
	return nil
}

type providerSignature struct {
	ID              string `json:"id"`
	Indication      string `json:"indication"`
	SubIndication   string `json:"subIndication"`
	SignatureLevel  string `json:"signatureLevel"`
	SignatureFormat string `json:"signatureFormat"`

	// Signer identity — one of two layouts: nested under signerExt, or the
	// same facts at the entry's top level.
	SignerExt          signerExt `json:"signerExt"`
	SignedBy           string    `json:"signedBy"`
	SignerSerialNumber string    `json:"signerSerialNumber"`

	// Times — flat fields in one layout (the upstream "ocspResponceTime"
	// spelling kept exact), nested under info in the other.
	TimeStamp           string        `json:"timeStamp"`
	OCSPResponseTime    string        `json:"ocspResponceTime"`
	MaximumValidityTime string        `json:"maximumValidityTime"`
	Info                signatureInfo `json:"info"`

	Errors   []reportNote `json:"errors"`
	Warnings []reportNote `json:"warnings"`
}

// signatureInfo is the nested time/context sub-object of the older layout.
type signatureInfo struct {
	BestSignatureTime        string `json:"bestSignatureTime"`
	TimestampCreationTime    string `json:"timestampCreationTime"`
	OCSPResponseCreationTime string `json:"ocspResponseCreationTime"`
}

// signerExt is the signing party's identity sub-object. The signed-by name is
// nested here (lowercase "signedby" in that upstream layout). A natural
// person carries a personal serial number; an organisation carries a
// registration number and the organisation name.
type signerExt struct {
	SignedBy           string `json:"signedby"`
	Organization       string `json:"organization"`
	SignerSerialNumber string `json:"signerSerialNumber"`
	RegistrationNumber string `json:"registrationNumber"`
}

// reportNote is one warning or error entry from the report.
type reportNote struct {
	Content string `json:"content"`
}

// containerFormASiCE is the container form whose signed-file list is the real
// set of data-object names.
const containerFormASiCE = "ASiC-E"

// NormalizeReport maps a verbatim provider validation report onto the
// normalized answer. The overall verdict is derived across all signatures
// (any failure → FAILED; all passed and all accounted-for → PASSED; otherwise
// INDETERMINATE). Every signature is normalized into Signatures (a container
// can hold several parallel co-signatures); the top-level per-signer fields
// mirror the first one for single-signature callers. Known level codes map to
// a legal-meaning label; an unknown code passes through rather than being
// guessed. Facts that different report layouts place differently (signer
// identity, times, file lists) are resolved by fallback across the known
// layouts.
func NormalizeReport(raw []byte) (*Validation, error) {
	var rep providerReport
	if err := json.Unmarshal(raw, &rep); err != nil {
		return nil, fmt.Errorf("answer: parse validation report: %w", err)
	}

	d := rep.Data
	res := &Validation{
		Verdict:              overallVerdict(d),
		ContainerForm:        d.SignatureForm,
		SignaturesCount:      d.SignaturesCount,
		ValidSignaturesCount: d.ValidSignaturesCount,
		ValidationLevel:      d.ValidationLevel,
		Warnings:             []string{},
		Errors:               []string{},
	}
	res.Pass = res.Verdict == VerdictPassed

	// The signed-file list is meaningful only for container forms; a single
	// PDF reports a placeholder rather than real file names. Prefer the
	// validatedDocument list, fall back to the data-level one.
	res.SignedFiles = []string{}
	if d.SignatureForm == containerFormASiCE {
		files := d.ValidatedDocument.IncludedFiles
		if len(files) == 0 {
			files = d.IncludedFiles
		}
		res.SignedFiles = append(res.SignedFiles, files...)
	}

	// Normalize every signature the report holds (a container may carry several).
	res.Signatures = make([]Signature, 0, len(d.SignaturesExt))
	for _, s := range d.SignaturesExt {
		res.Signatures = append(res.Signatures, Signature{
			Verdict:         mapIndication(s.Indication),
			Format:          s.SignatureFormat,
			Level:           legalLevel(s.SignatureLevel),
			Signer:          firstOf(s.SignerExt.SignedBy, s.SignedBy),
			Organization:    s.SignerExt.Organization,
			SignerSerial:    signerSerial(s),
			SigningTime:     normalizeTimestamp(firstOf(s.TimeStamp, s.Info.BestSignatureTime, s.Info.TimestampCreationTime)),
			RevocationTime:  normalizeTimestamp(firstOf(s.OCSPResponseTime, s.Info.OCSPResponseCreationTime)),
			MaxValidityTime: normalizeTimestamp(s.MaximumValidityTime),
			Warnings:        notesToStrings(s.Warnings),
			Errors:          notesToStrings(s.Errors),
		})
	}

	// The top-level per-signer fields mirror the first signature so
	// single-signature callers keep working; Signatures carries the full set.
	if len(res.Signatures) > 0 {
		f := res.Signatures[0]
		res.Format = f.Format
		res.Level = f.Level
		res.Signer = f.Signer
		res.Organization = f.Organization
		res.SignerSerial = f.SignerSerial
		res.SigningTime = f.SigningTime
		res.RevocationTime = f.RevocationTime
		res.MaxValidityTime = f.MaxValidityTime
		res.Warnings = f.Warnings
		res.Errors = f.Errors
	}

	return res, nil
}

// firstOf returns the first non-empty value.
func firstOf(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}

// signerSerial picks the serial number to display for the signing party: an
// organisation's registration number when present, otherwise the natural
// person's personal serial number (from whichever layout carries it).
func signerSerial(s providerSignature) string {
	if s.SignerExt.RegistrationNumber != "" {
		return s.SignerExt.RegistrationNumber
	}

	return firstOf(s.SignerExt.SignerSerialNumber, s.SignerSerialNumber)
}

// notesToStrings flattens the report's warning/error entries to their text,
// always returning a non-nil slice so an empty set is an explicit "none"
// rather than a missing field.
func notesToStrings(notes []reportNote) []string {
	out := make([]string, 0, len(notes))
	for _, n := range notes {
		if n.Content != "" {
			out = append(out, n.Content)
		}
	}

	return out
}

// overallVerdict collapses the per-signature indications into one verdict.
func overallVerdict(d providerReportData) string {
	if len(d.SignaturesExt) == 0 {
		return VerdictIndeterminate
	}

	allPassed := true
	for _, s := range d.SignaturesExt {
		switch mapIndication(s.Indication) {
		case VerdictFailed:
			return VerdictFailed
		case VerdictPassed:
			// keep checking
		default:
			allPassed = false
		}
	}

	if allPassed && d.ValidSignaturesCount == d.SignaturesCount && d.SignaturesCount > 0 {
		return VerdictPassed
	}

	return VerdictIndeterminate
}

// mapIndication maps a report indication to the verdict vocabulary.
func mapIndication(indication string) string {
	switch indication {
	case "TOTAL-PASSED":
		return VerdictPassed
	case "TOTAL-FAILED":
		return VerdictFailed
	default:
		// INDETERMINATE and any unrecognized indication are treated conservatively.
		return VerdictIndeterminate
	}
}

// legalLevel maps a report signature-level code to a legal-meaning label
// (qualified signature / qualified seal / advanced), passing an unrecognized
// code through unchanged so it stays visible rather than mislabeled.
func legalLevel(level string) string {
	switch level {
	case "QESIG":
		return "QES"
	case "QESEAL":
		return "SEAL"
	case "ADESIG_QC", "ADESIG", "ADESEAL_QC", "ADESEAL":
		return "AdES"
	default:
		return level
	}
}

// reportTimeLayout is a legacy locale timestamp format kept as a fallback.
// The report's times are RFC 3339, which this layout does not match, so they
// pass through unchanged; the fallback future-proofs a format change.
const reportTimeLayout = "02.01.2006. 15:04"

// normalizeTimestamp returns an RFC 3339 timestamp unchanged, converts a
// legacy locale timestamp to RFC 3339 when it matches, and otherwise returns
// the input as-is (the report format is not contractual). An empty input
// stays empty.
func normalizeTimestamp(s string) string {
	if s == "" {
		return ""
	}
	if t, err := time.Parse(reportTimeLayout, s); err == nil {
		return t.UTC().Format(time.RFC3339)
	}

	return s
}
