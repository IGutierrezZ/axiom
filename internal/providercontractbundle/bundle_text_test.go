package providercontractbundle

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// Digests of the exact bytes published at contract 1.2.0 and before (v3.5.1):
// README.md and each schemas/<role>.schema.json as written into the archive.
// They pin the legacy text so a later edit of the current README or schemas
// can never silently change what a published archive must still verify against.
const (
	legacyREADMESHA256          = "18baab5ee79aefd0bc62a28da0dadcf2162544f57a940c5b859cd6bc4932a085"
	legacyLensSchemaSHA256      = "9b8a2354a570888486015434501b66b51441a3baf5bf858635e4fff56b2e8298"
	legacyRefuterSchemaSHA256   = "36f8267c6c3f04600b8f1cb77df70d90858611af822cc81922efd6b3502fb646"
	legacyValidatorSchemaSHA256 = "40e397195bdfbc6782fc3090f20c5be082e4d14d2855314f99ecb8506133eddc"
)

var textBoundFiles = []string{
	"README.md",
	"schemas/lens.schema.json",
	"schemas/refuter.schema.json",
	"schemas/targeted-validator.schema.json",
}

func TestLegacyBundleTextMatchesThePublishedDigests(t *testing.T) {
	files, err := generatedFiles("1.2.0")
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"README.md":                              legacyREADMESHA256,
		"schemas/lens.schema.json":               legacyLensSchemaSHA256,
		"schemas/refuter.schema.json":            legacyRefuterSchemaSHA256,
		"schemas/targeted-validator.schema.json": legacyValidatorSchemaSHA256,
	} {
		if got := strings.TrimPrefix(hash(files[name]), "sha256:"); got != want {
			t.Errorf("legacy %s sha256 = %s, want published %s", name, got, want)
		}
	}
}

// A semver that matches the pattern but overflows the parser must be refused,
// never read as the oldest contract and answered with the legacy text.
func TestBundleTextRefusesAnUnreadableContractSemver(t *testing.T) {
	if _, err := generatedFiles("1.2.99999999999999999999"); err == nil {
		t.Fatal("generatedFiles accepted a contract semver its parser cannot read")
	}
}

func TestCurrentBundleTextNamesAxiomAndLegacyKeepsGentleAI(t *testing.T) {
	current, err := generatedFiles("1.2.1")
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := generatedFiles("1.2.0")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range textBoundFiles {
		if bytes.Contains(current[name], []byte("Gentle AI")) {
			t.Errorf("current %s still names Gentle AI", name)
		}
		if !bytes.Contains(current[name], []byte("Axiom")) {
			t.Errorf("current %s does not name Axiom", name)
		}
		if !bytes.Contains(legacy[name], []byte("Gentle AI")) || bytes.Contains(legacy[name], []byte("Axiom")) {
			t.Errorf("legacy %s must carry the Gentle AI text and no Axiom text", name)
		}
	}
	// Vectors and orchestration do not depend on the text version.
	for _, name := range []string{"vectors/lens.json", "vectors/refuter.json", "vectors/targeted-validator.json", "orchestration/pi.md"} {
		if !bytes.Equal(current[name], legacy[name]) {
			t.Errorf("%s differs between legacy and current bundles", name)
		}
	}
}

func TestVerifyArchiveAcceptsPublishedLegacyAndCurrentText(t *testing.T) {
	for _, semver := range []string{"1.2.0", "1.2.1", "1.3.0"} {
		t.Run(semver, func(t *testing.T) {
			files, err := generatedFiles(semver)
			if err != nil {
				t.Fatal(err)
			}
			archive := filepath.Join(t.TempDir(), "bundle.tar.gz")
			writeArchive(t, archive, files, nil, false)
			if err := VerifyArchive(archive); err != nil {
				t.Fatalf("VerifyArchive rejected the %s bundle: %v", semver, err)
			}
		})
	}
	directory := t.TempDir()
	if err := Generate(directory, "1.2.0"); err != nil {
		t.Fatal(err)
	}
	if err := VerifyStaging(directory); err != nil {
		t.Fatalf("VerifyStaging rejected the 1.2.0 staging directory: %v", err)
	}
}

// rebindManifest declares semver in files' manifest and re-binds its README and
// schema digests to the bytes files actually carries, so a rejection can only
// come from the text-for-version check and never from a stale digest.
func rebindManifest(t *testing.T, files map[string][]byte, semver string) {
	t.Helper()
	var decoded manifest
	if err := json.Unmarshal(files["manifest.json"], &decoded); err != nil {
		t.Fatal(err)
	}
	decoded.ContractSemver = semver
	decoded.README.SHA256 = hash(files["README.md"])
	for index := range decoded.Roles {
		decoded.Roles[index].Schema.SHA256 = hash(files[decoded.Roles[index].Schema.Path])
	}
	payload, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	files["manifest.json"] = append(payload, '\n')
}

func TestVerifyArchiveRejectsTextThatDoesNotMatchTheDeclaredContract(t *testing.T) {
	current, err := generatedFiles("1.2.1")
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := generatedFiles("1.2.0")
	if err != nil {
		t.Fatal(err)
	}
	mixed := func(base, other map[string][]byte, replaced ...string) func() map[string][]byte {
		return func() map[string][]byte {
			files := cloneFiles(base)
			for _, name := range replaced {
				files[name] = other[name]
			}
			return files
		}
	}
	schemas := textBoundFiles[1:]
	for _, test := range []struct {
		name   string
		semver string
		build  func() map[string][]byte
	}{
		{name: "current text declared as 1.2.0", semver: "1.2.0", build: func() map[string][]byte { return cloneFiles(current) }},
		{name: "legacy text declared as 1.2.1", semver: "1.2.1", build: func() map[string][]byte { return cloneFiles(legacy) }},
		{name: "legacy text declared as 1.3.0", semver: "1.3.0", build: func() map[string][]byte { return cloneFiles(legacy) }},
		{name: "legacy README with current schemas declared as 1.2.1", semver: "1.2.1", build: mixed(current, legacy, "README.md")},
		{name: "legacy README with current schemas declared as 1.2.0", semver: "1.2.0", build: mixed(current, legacy, "README.md")},
		{name: "current README with legacy schemas declared as 1.2.0", semver: "1.2.0", build: mixed(legacy, current, "README.md")},
		{name: "current README with legacy schemas declared as 1.2.1", semver: "1.2.1", build: mixed(legacy, current, "README.md")},
		{name: "legacy schemas with current README declared as 1.2.1", semver: "1.2.1", build: mixed(current, legacy, schemas...)},
		{name: "current schemas with legacy README declared as 1.2.0", semver: "1.2.0", build: mixed(legacy, current, schemas...)},
		{name: "one legacy schema among current ones declared as 1.2.1", semver: "1.2.1", build: mixed(current, legacy, "schemas/refuter.schema.json")},
		{name: "one current schema among legacy ones declared as 1.2.0", semver: "1.2.0", build: mixed(legacy, current, "schemas/lens.schema.json")},
	} {
		t.Run(test.name, func(t *testing.T) {
			files := test.build()
			rebindManifest(t, files, test.semver)
			archive := filepath.Join(t.TempDir(), "mismatch.tar.gz")
			writeArchive(t, archive, files, nil, false)
			if err := VerifyArchive(archive); err == nil {
				t.Fatal("VerifyArchive accepted README/schema text that does not belong to the declared contract_semver")
			}
		})
	}
}

func TestLegacyResultSchemaFailsLoudlyUnlessExactlyOneKnownTitleOccurs(t *testing.T) {
	const lens = `{"title": "Axiom reviewer result"}`
	for _, test := range []struct {
		name    string
		schema  string
		want    string
		wantErr bool
	}{
		{name: "lens title", schema: lens, want: `{"title": "Gentle AI reviewer result"}`},
		{name: "refuter title", schema: `{"title":"Axiom refuter result"}`, want: `{"title":"Gentle AI refuter result"}`},
		{name: "validator title", schema: `{"title":"Axiom targeted validator result"}`, want: `{"title":"Gentle AI targeted validator result"}`},
		{name: "no known title", schema: `{"title":"Something else"}`, wantErr: true},
		{name: "already legacy", schema: `{"title":"Gentle AI refuter result"}`, wantErr: true},
		{name: "duplicated title", schema: lens + lens, wantErr: true},
		{name: "two different known titles", schema: lens + `{"title":"Axiom refuter result"}`, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := legacyResultSchema([]byte(test.schema))
			if test.wantErr {
				if err == nil {
					t.Fatalf("legacyResultSchema(%q) = %q, want an error", test.schema, got)
				}
				return
			}
			if err != nil || string(got) != test.want {
				t.Fatalf("legacyResultSchema(%q) = %q, %v, want %q", test.schema, got, err, test.want)
			}
		})
	}
}

// TestRepositoryContractSemverIsNotBeforeTheAxiomText keeps a forgotten bump
// from shipping the Axiom README and schemas under a contract version that
// promises the legacy bytes: the release would publish an archive that the
// same code then refuses to verify.
func TestRepositoryContractSemverIsNotBeforeTheAxiomText(t *testing.T) {
	semver, err := ReadContractSemver(filepath.Join("..", "..", "contracts", "review-provider-contract", "CONTRACT_SEMVER"))
	if err != nil {
		t.Fatal(err)
	}
	if !contractSemverAtLeast(semver, contractSemverIntroducingAxiomText) {
		t.Fatalf("CONTRACT_SEMVER %s is before the Axiom text contract %+v: bump it", semver, contractSemverIntroducingAxiomText)
	}
	files, err := generatedFiles(semver)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(files["README.md"], []byte("Gentle AI")) {
		t.Fatalf("the repository's own contract %s would publish the legacy README", semver)
	}
}
