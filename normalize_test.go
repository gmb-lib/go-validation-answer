package answer

import (
	"encoding/json"
	"testing"
)

// passingReport is a signerExt-layout report (nested identity, flat times,
// string file list): one qualified personal signature on a container.
const passingReport = `{"data":{"signatureForm":"ASiC-E","validationLevel":"LTA",` +
	`"signaturesExt":[{"id":"S1","indication":"TOTAL-PASSED","signatureLevel":"QESIG",` +
	`"signatureFormat":"XAdES_BASELINE_LT",` +
	`"signerExt":{"signedby":"TEST SIGNER","signerSerialNumber":"PNOLV-111111-11111"},` +
	`"timeStamp":"2026-06-27T07:22:26Z","ocspResponceTime":"2026-06-27T07:22:27Z",` +
	`"maximumValidityTime":"2030-01-01T00:00:00Z","errors":[],"warnings":[]}],` +
	`"signaturesCount":1,"validSignaturesCount":1,` +
	`"validatedDocument":{"fileName":"c.asice","includedFiles":["contract.pdf"]}}}`

// archivalReport is the OTHER layout (top-level identity, times nested under
// info, data-level object file list): a re-validated archive-timestamped
// container, as the validation API family also produces. Mirrors a real
// post-archive report byte-shape with a synthesized signer.
const archivalReport = `{"data":{"includedFiles":[{"filename":"test.txt"}],` +
	`"signatureForm":"ASiC-E","signaturesCount":1,` +
	`"signaturesExt":[{"errors":[],"id":"id-1","indication":"TOTAL-PASSED",` +
	`"info":{"bestSignatureTime":"2026-07-19T08:57:32Z","ocspResponseCreationTime":"2026-07-19T08:57:33Z",` +
	`"signatureProductionPlace":{},"signerRole":[],"timestampCreationTime":"2026-07-19T08:57:32Z"},` +
	`"signatureFormat":"XAdES_BASELINE_LTA","signatureLevel":"QESIG",` +
	`"signedBy":"TEST SIGNER","signerSerialNumber":"PNOLV-111111-11111","subIndication":"",` +
	`"warnings":[{"content":"The trusted list is not considered as fresh!"}]}],` +
	`"validSignaturesCount":1,"validatedDocument":{"filename":"1.extended.edoc"},` +
	`"validationLevel":"ARCHIVAL_DATA"}}`

func TestNormalizePassedQualifiedPerson(t *testing.T) {
	res, err := NormalizeReport([]byte(passingReport))
	if err != nil {
		t.Fatalf("NormalizeReport: %v", err)
	}
	if res.Verdict != VerdictPassed || !res.Pass {
		t.Fatalf("verdict: got %q pass=%v", res.Verdict, res.Pass)
	}
	if res.Format != "XAdES_BASELINE_LT" || res.Level != "QES" {
		t.Fatalf("format/level: got %q/%q", res.Format, res.Level)
	}
	if res.Signer != "TEST SIGNER" || res.SignerSerial != "PNOLV-111111-11111" || res.Organization != "" {
		t.Fatalf("identity: got %q/%q/%q", res.Signer, res.SignerSerial, res.Organization)
	}
	if res.ContainerForm != "ASiC-E" {
		t.Fatalf("containerForm: got %q", res.ContainerForm)
	}
	if res.SigningTime != "2026-06-27T07:22:26Z" || res.RevocationTime != "2026-06-27T07:22:27Z" ||
		res.MaxValidityTime != "2030-01-01T00:00:00Z" {
		t.Fatalf("times: got %q/%q/%q", res.SigningTime, res.RevocationTime, res.MaxValidityTime)
	}
	if len(res.SignedFiles) != 1 || res.SignedFiles[0] != "contract.pdf" {
		t.Fatalf("signedFiles: got %v", res.SignedFiles)
	}
	if res.Warnings == nil || len(res.Warnings) != 0 || res.Errors == nil || len(res.Errors) != 0 {
		t.Fatalf("notes: got %v / %v (want empty non-nil)", res.Warnings, res.Errors)
	}
	if len(res.Signatures) != 1 || res.Signatures[0].Signer != "TEST SIGNER" {
		t.Fatalf("signatures: got %v", res.Signatures)
	}
}

// The archival layout resolves the same facts from its different placements:
// identity from the entry's top level, signing time from info.bestSignatureTime,
// revocation time from info.ocspResponseCreationTime, files from the data-level
// object list — and the validation level is captured.
func TestNormalizeArchivalLayout(t *testing.T) {
	res, err := NormalizeReport([]byte(archivalReport))
	if err != nil {
		t.Fatalf("NormalizeReport: %v", err)
	}
	if res.Verdict != VerdictPassed || !res.Pass {
		t.Fatalf("verdict: got %q pass=%v", res.Verdict, res.Pass)
	}
	if res.Format != "XAdES_BASELINE_LTA" || res.Level != "QES" {
		t.Fatalf("format/level: got %q/%q", res.Format, res.Level)
	}
	if res.Signer != "TEST SIGNER" || res.SignerSerial != "PNOLV-111111-11111" {
		t.Fatalf("identity from the top-level layout: got %q/%q", res.Signer, res.SignerSerial)
	}
	if res.SigningTime != "2026-07-19T08:57:32Z" {
		t.Fatalf("signingTime from info.bestSignatureTime: got %q", res.SigningTime)
	}
	if res.RevocationTime != "2026-07-19T08:57:33Z" {
		t.Fatalf("revocationTime from info.ocspResponseCreationTime: got %q", res.RevocationTime)
	}
	if len(res.SignedFiles) != 1 || res.SignedFiles[0] != "test.txt" {
		t.Fatalf("signedFiles from the data-level object list: got %v", res.SignedFiles)
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("warnings: got %v", res.Warnings)
	}
	if res.ValidationLevel != "ARCHIVAL_DATA" {
		t.Fatalf("validationLevel: got %q", res.ValidationLevel)
	}
}

// The signerExt layout also serves archive-timestamped documents: the LTA
// format, the archival validation level, and a populated maximum-validity
// horizon all normalize from the same fields. Mirrors a real post-archive
// report byte-shape with a synthesized signer.
func TestNormalizeArchivedSignerExtLayout(t *testing.T) {
	const v2Archival = `{"data":{"signatureForm":"ASiC-E","signaturesCount":1,` +
		`"signaturesExt":[{"errors":[],"id":"id-1","indication":"TOTAL-PASSED",` +
		`"ocspResponceTime":"2026-07-19T08:57:33Z","signatureFormat":"XAdES_BASELINE_LTA",` +
		`"signatureLevel":"QESIG",` +
		`"signerExt":{"signatureProductionPlace":{},"signedby":"TEST SIGNER","signerRole":[],` +
		`"signerSerialNumber":"PNOLV-111111-11111"},"subIndication":"",` +
		`"timeStamp":"2026-07-19T08:57:32Z",` +
		`"warnings":[{"content":"The trusted list is not considered as fresh!"}],` +
		`"maximumValidityTime":"2027-12-13T08:36:59Z"}],"validSignaturesCount":1,` +
		`"validatedDocument":{"fileName":"1.extended.edoc","includedFiles":["test.txt"]},` +
		`"validationLevel":"ARCHIVAL_DATA"}}`

	res, err := NormalizeReport([]byte(v2Archival))
	if err != nil {
		t.Fatalf("NormalizeReport: %v", err)
	}
	if res.Verdict != VerdictPassed || res.Format != "XAdES_BASELINE_LTA" || res.Level != "QES" {
		t.Fatalf("got %q/%q/%q", res.Verdict, res.Format, res.Level)
	}
	if res.MaxValidityTime != "2027-12-13T08:36:59Z" {
		t.Fatalf("maxValidityTime: got %q", res.MaxValidityTime)
	}
	if res.ValidationLevel != "ARCHIVAL_DATA" {
		t.Fatalf("validationLevel: got %q", res.ValidationLevel)
	}
	if len(res.SignedFiles) != 1 || res.SignedFiles[0] != "test.txt" {
		t.Fatalf("signedFiles: got %v", res.SignedFiles)
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("warnings: got %v", res.Warnings)
	}
}

func TestNormalizeMultipleSignatures(t *testing.T) {
	const twoSigs = `{"data":{"signatureForm":"ASiC-E",` +
		`"signaturesExt":[` +
		`{"id":"S1","indication":"TOTAL-PASSED","signatureLevel":"QESIG","signatureFormat":"XAdES_BASELINE_LT",` +
		`"signerExt":{"signedby":"FIRST SIGNER","signerSerialNumber":"PNOLV-111111-11111"},` +
		`"timeStamp":"2026-06-28T10:00:00Z","ocspResponceTime":"2026-06-28T10:00:01Z","errors":[],"warnings":[]},` +
		`{"id":"S2","indication":"TOTAL-PASSED","signatureLevel":"QESIG","signatureFormat":"XAdES_BASELINE_LT",` +
		`"signerExt":{"signedby":"SECOND SIGNER","signerSerialNumber":"PNOLV-222222-22222"},` +
		`"timeStamp":"2026-06-28T11:00:00Z","ocspResponceTime":"2026-06-28T11:00:01Z","errors":[],"warnings":[]}],` +
		`"signaturesCount":2,"validSignaturesCount":2,` +
		`"validatedDocument":{"fileName":"c.asice","includedFiles":["contract.pdf"]}}}`

	res, err := NormalizeReport([]byte(twoSigs))
	if err != nil {
		t.Fatalf("NormalizeReport: %v", err)
	}
	if res.Verdict != VerdictPassed || len(res.Signatures) != 2 {
		t.Fatalf("got %q / %d signatures", res.Verdict, len(res.Signatures))
	}
	if res.Signatures[0].Signer != "FIRST SIGNER" || res.Signatures[1].Signer != "SECOND SIGNER" {
		t.Fatalf("signers: got %q, %q", res.Signatures[0].Signer, res.Signatures[1].Signer)
	}
	if res.Signatures[1].SigningTime != "2026-06-28T11:00:00Z" {
		t.Fatalf("second signingTime: got %q", res.Signatures[1].SigningTime)
	}
	// The top-level per-signer fields mirror the first signature (backward compat).
	if res.Signer != "FIRST SIGNER" || res.SignerSerial != "PNOLV-111111-11111" {
		t.Fatalf("top-level mirror: got signer=%q serial=%q", res.Signer, res.SignerSerial)
	}
}

func TestNormalizeOrgSealWithWarnings(t *testing.T) {
	const orgSeal = `{"data":{"signatureForm":"PDF",` +
		`"signaturesExt":[{"id":"S1","indication":"TOTAL-PASSED","signatureLevel":"ADESEAL_QC",` +
		`"signatureFormat":"PAdES_BASELINE_LT",` +
		`"signerExt":{"signedby":"EXAMPLE ORG SEAL","organization":"Example Org",` +
		`"signerSerialNumber":"40000000000","registrationNumber":"NTRLV-40000000000"},` +
		`"timeStamp":"2026-06-27T07:22:26Z","ocspResponceTime":"2026-06-27T07:22:26Z",` +
		`"errors":[],"warnings":[{"content":"The private key does not reside in a QSCD at (best) signing time!"},` +
		`{"content":"The private key does not reside in a QSCD at issuance time!"}]}],` +
		`"signaturesCount":1,"validSignaturesCount":1,` +
		`"validatedDocument":{"fileName":"a.signed.pdf","includedFiles":["Partial PDF"]}}}`

	res, err := NormalizeReport([]byte(orgSeal))
	if err != nil {
		t.Fatalf("NormalizeReport: %v", err)
	}
	if res.Verdict != VerdictPassed || !res.Pass {
		t.Fatalf("verdict: got %q pass=%v (warnings must not fail the verdict)", res.Verdict, res.Pass)
	}
	if res.Level != "AdES" {
		t.Fatalf("level: got %q (want AdES for a QC seal not on a QSCD)", res.Level)
	}
	if res.SignerSerial != "NTRLV-40000000000" || res.Organization != "Example Org" {
		t.Fatalf("org identity: got %q/%q (registration number preferred)", res.SignerSerial, res.Organization)
	}
	if len(res.Warnings) != 2 {
		t.Fatalf("warnings: got %v (want 2)", res.Warnings)
	}
	// A single PDF reports a placeholder, not real file names — the list stays empty.
	if len(res.SignedFiles) != 0 {
		t.Fatalf("signedFiles: got %v (want empty for a PDF)", res.SignedFiles)
	}
	if res.MaxValidityTime != "" {
		t.Fatalf("maxValidityTime: got %q (want empty when absent)", res.MaxValidityTime)
	}
}

func TestNormalizeFailedAndIndeterminate(t *testing.T) {
	failed := `{"data":{"signaturesExt":[{"indication":"TOTAL-FAILED","signatureLevel":"QESIG",` +
		`"signerExt":{"signedby":"X"}}],"signaturesCount":1,"validSignaturesCount":0}}`
	res, err := NormalizeReport([]byte(failed))
	if err != nil {
		t.Fatalf("NormalizeReport: %v", err)
	}
	if res.Verdict != VerdictFailed || res.Pass {
		t.Fatalf("expected FAILED/!pass, got %q pass=%v", res.Verdict, res.Pass)
	}

	// All-passed indications but a count mismatch is INDETERMINATE, not PASSED.
	mismatch := `{"data":{"signaturesExt":[{"indication":"TOTAL-PASSED","signatureLevel":"QESIG"}],` +
		`"signaturesCount":2,"validSignaturesCount":1}}`
	if res, err = NormalizeReport([]byte(mismatch)); err != nil || res.Verdict != VerdictIndeterminate || res.Pass {
		t.Fatalf("expected INDETERMINATE/!pass, got %+v err=%v", res, err)
	}

	// No signatures → INDETERMINATE.
	empty := `{"data":{"signaturesExt":[],"signaturesCount":0,"validSignaturesCount":0}}`
	if res, err = NormalizeReport([]byte(empty)); err != nil || res.Verdict != VerdictIndeterminate {
		t.Fatalf("expected INDETERMINATE for no signatures, got %+v err=%v", res, err)
	}

	// Not JSON at all → error, never a fabricated verdict.
	if _, err := NormalizeReport([]byte("not json")); err == nil {
		t.Fatal("expected an error for unparseable input")
	}
}

func TestLegalLevelMapping(t *testing.T) {
	cases := map[string]string{
		"QESIG":      "QES",
		"QESEAL":     "SEAL",
		"ADESIG_QC":  "AdES",
		"ADESEAL_QC": "AdES",
		"SOMETHING":  "SOMETHING", // unknown codes pass through, not guessed
	}
	for in, want := range cases {
		if got := legalLevel(in); got != want {
			t.Errorf("legalLevel(%q) = %q, want %q", in, got, want)
		}
	}
}

// The wire contract: a fully-populated answer marshals to exactly the agreed
// key set (and the in-process fields stay off the wire). Guards accidental
// tag drift — every consumer serves this shape.
func TestWireContractKeys(t *testing.T) {
	full := Validation{
		SignatureID: "sig", DocumentID: "doc", Verdict: VerdictPassed, Format: "F", Level: "L",
		Signer: "S", SignerSerial: "SS", Organization: "O", ContainerForm: "ASiC-E",
		SigningTime: "t1", RevocationTime: "t2", MaxValidityTime: "t3",
		SignedFiles: []string{"f"}, Warnings: []string{"w"}, Errors: []string{"e"},
		Signatures: []Signature{{Verdict: VerdictPassed}}, Pass: true, ReportID: "r",
		SignaturesCount: 1, ValidSignaturesCount: 1, ValidationLevel: "ARCHIVAL_DATA",
	}
	raw, err := json.Marshal(full)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var keys map[string]any
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := []string{"signatureId", "documentId", "verdict", "format", "level", "signer",
		"signerSerial", "organization", "containerForm", "signingTime", "revocationTime",
		"maxValidityTime", "signedFiles", "warnings", "errors", "signatures", "pass", "reportId"}
	if len(keys) != len(want) {
		t.Fatalf("wire keys: got %d (%v), want %d", len(keys), keys, len(want))
	}
	for _, k := range want {
		if _, ok := keys[k]; !ok {
			t.Errorf("wire key %q missing", k)
		}
	}
}

// FuzzNormalizeReport asserts the untrusted-input invariant: arbitrary bytes
// never panic — they normalize or error.
func FuzzNormalizeReport(f *testing.F) {
	f.Add([]byte(passingReport))
	f.Add([]byte(archivalReport))
	f.Add([]byte("{"))
	f.Add([]byte(`{"data":{}}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = NormalizeReport(data)
	})
}
