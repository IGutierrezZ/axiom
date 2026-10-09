package workspace

import (
	"path/filepath"
	"testing"
)

// artifactStoreFixture es un caso del lector de artifact_store. La prueba de
// paridad de internal/sddstatus repite estas mismas fixtures contra
// declaredArtifactStore: si se añade un caso aquí, hay que añadirlo allí.
type artifactStoreFixture struct {
	name    string
	files   map[string]string // nombre dentro de openspec/ -> contenido
	want    string
	wantSet bool
}

func artifactStoreFixtures() []artifactStoreFixture {
	return []artifactStoreFixture{
		{name: "sin config", files: nil},
		{name: "openspec", files: map[string]string{"config.yaml": "artifact_store: openspec\n"}, want: "openspec", wantSet: true},
		{name: "hybrid", files: map[string]string{"config.yaml": "artifact_store: hybrid\n"}, want: "hybrid", wantSet: true},
		{name: "engram", files: map[string]string{"config.yaml": "artifact_store: engram\n"}, want: "engram", wantSet: true},
		{name: "none", files: map[string]string{"config.yaml": "artifact_store: none\n"}, want: "none", wantSet: true},
		{name: "config.yml", files: map[string]string{"config.yml": "artifact_store: engram\n"}, want: "engram", wantSet: true},
		{name: "clave camelCase", files: map[string]string{"config.yaml": "artifactStore: hybrid\n"}, want: "hybrid", wantSet: true},
		{name: "mayusculas", files: map[string]string{"config.yaml": "artifact_store: HYBRID\n"}, want: "hybrid", wantSet: true},
		{name: "comillas dobles", files: map[string]string{"config.yaml": "artifact_store: \"none\"\n"}, want: "none", wantSet: true},
		{name: "comillas simples", files: map[string]string{"config.yaml": "artifact_store: 'engram'\n"}, want: "engram", wantSet: true},
		{name: "comentario al final", files: map[string]string{"config.yaml": "artifact_store: none # sin almacén\n"}, want: "none", wantSet: true},
		{name: "linea comentada", files: map[string]string{"config.yaml": "# artifact_store: none\nartifact_store: hybrid\n"}, want: "hybrid", wantSet: true},
		{name: "solo comentada", files: map[string]string{"config.yaml": "# artifact_store: none\n"}},
		{name: "con otras claves", files: map[string]string{"config.yaml": "strict_tdd: false\nrules:\n  apply: []\nartifact_store: engram\n"}, want: "engram", wantSet: true},
		{name: "sin la clave", files: map[string]string{"config.yaml": "strict_tdd: false\n"}},
		{name: "valor desconocido", files: map[string]string{"config.yaml": "artifact_store: s3\n"}},
		{name: "valor vacio", files: map[string]string{"config.yaml": "artifact_store:\n"}},
		{name: "config.yaml prevalece sobre config.yml", files: map[string]string{"config.yaml": "artifact_store: none\n", "config.yml": "artifact_store: engram\n"}, want: "none", wantSet: true},
		{name: "config.yaml sin la clave cede a config.yml", files: map[string]string{"config.yaml": "strict_tdd: true\n", "config.yml": "artifact_store: engram\n"}, want: "engram", wantSet: true},
		{name: "valor desconocido en config.yaml no cede a config.yml", files: map[string]string{"config.yaml": "artifact_store: s3\n", "config.yml": "artifact_store: engram\n"}},
		{name: "primera aparicion de la clave", files: map[string]string{"config.yaml": "artifact_store: none\nartifact_store: engram\n"}, want: "none", wantSet: true},
		{name: "terminadores de linea de Windows", files: map[string]string{"config.yaml": "strict_tdd: false\r\nartifact_store: hybrid\r\n"}, want: "hybrid", wantSet: true},
	}
}

// writeArtifactStoreFixture materializa los ficheros de una fixture bajo root/openspec.
func writeArtifactStoreFixture(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		writeTestFile(t, filepath.Join(root, "openspec", name), content)
	}
}

func TestArtifactStoreDeclared(t *testing.T) {
	for _, fixture := range artifactStoreFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			root := t.TempDir()
			writeArtifactStoreFixture(t, root, fixture.files)
			got, declared := DeclaredArtifactStore(root)
			if got != fixture.want || declared != fixture.wantSet {
				t.Fatalf("DeclaredArtifactStore = (%q, %v), se esperaba (%q, %v)", got, declared, fixture.want, fixture.wantSet)
			}
		})
	}
}

func TestArtifactStoreDeclaredWithoutPlanningHome(t *testing.T) {
	if got, declared := DeclaredArtifactStore(filepath.Join(t.TempDir(), "no-existe")); got != "" || declared {
		t.Fatalf("DeclaredArtifactStore = (%q, %v), sin directorio no hay declaración", got, declared)
	}
}
