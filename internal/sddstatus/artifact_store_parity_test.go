package sddstatus

import (
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// R40 y m10: workspace.DeclaredArtifactStore replica a declaredArtifactStore
// porque el segundo vive en el motor del status, código que el fork comparte con
// el upstream y que este incremento no toca (R9). Esta prueba ejecuta los dos
// lectores sobre las mismas fixtures y exige resultados iguales: es la red que
// impide que diverjan. Las fixtures son las de artifact_store_test.go de
// internal/workspace más la forma anidada (`sdd:` / `artifact_store:`) que usan
// las pruebas existentes de este paquete.
func TestArtifactStoreReadersAgree(t *testing.T) {
	fixtures := []struct {
		name  string
		files map[string]string // nombre dentro de openspec/ -> contenido
	}{
		{"sin config", nil},
		{"openspec", map[string]string{"config.yaml": "artifact_store: openspec\n"}},
		{"hybrid", map[string]string{"config.yaml": "artifact_store: hybrid\n"}},
		{"engram", map[string]string{"config.yaml": "artifact_store: engram\n"}},
		{"none", map[string]string{"config.yaml": "artifact_store: none\n"}},
		{"config.yml", map[string]string{"config.yml": "artifact_store: engram\n"}},
		{"clave camelCase", map[string]string{"config.yaml": "artifactStore: hybrid\n"}},
		{"mayusculas", map[string]string{"config.yaml": "artifact_store: HYBRID\n"}},
		{"comillas dobles", map[string]string{"config.yaml": "artifact_store: \"none\"\n"}},
		{"comillas simples", map[string]string{"config.yaml": "artifact_store: 'engram'\n"}},
		{"comentario al final", map[string]string{"config.yaml": "artifact_store: none # sin almacén\n"}},
		{"linea comentada", map[string]string{"config.yaml": "# artifact_store: none\nartifact_store: hybrid\n"}},
		{"solo comentada", map[string]string{"config.yaml": "# artifact_store: none\n"}},
		{"con otras claves", map[string]string{"config.yaml": "strict_tdd: false\nrules:\n  apply: []\nartifact_store: engram\n"}},
		{"sin la clave", map[string]string{"config.yaml": "strict_tdd: false\n"}},
		{"valor desconocido", map[string]string{"config.yaml": "artifact_store: s3\n"}},
		{"valor vacio", map[string]string{"config.yaml": "artifact_store:\n"}},
		{"config.yaml prevalece sobre config.yml", map[string]string{"config.yaml": "artifact_store: none\n", "config.yml": "artifact_store: engram\n"}},
		{"config.yaml sin la clave cede a config.yml", map[string]string{"config.yaml": "strict_tdd: true\n", "config.yml": "artifact_store: engram\n"}},
		{"valor desconocido en config.yaml no cede a config.yml", map[string]string{"config.yaml": "artifact_store: s3\n", "config.yml": "artifact_store: engram\n"}},
		{"primera aparicion de la clave", map[string]string{"config.yaml": "artifact_store: none\nartifact_store: engram\n"}},
		{"terminadores de linea de Windows", map[string]string{"config.yaml": "strict_tdd: false\r\nartifact_store: hybrid\r\n"}},
		{"clave anidada", map[string]string{"config.yaml": "sdd:\n  artifact_store: hybrid\n"}},
		{"clave anidada y comentario", map[string]string{"config.yaml": "sdd:\n  # almacén\n  artifactStore: 'Engram' # engram\n"}},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			root := t.TempDir()
			for name, content := range fixture.files {
				write(t, filepath.Join(root, "openspec", name), content)
			}
			gotStore, gotDeclared := declaredArtifactStore(root)
			wantStore, wantDeclared := workspace.DeclaredArtifactStore(root)
			if string(gotStore) != wantStore || gotDeclared != wantDeclared {
				t.Fatalf("declaredArtifactStore = (%q, %v) pero workspace.DeclaredArtifactStore = (%q, %v): los dos lectores deben coincidir",
					gotStore, gotDeclared, wantStore, wantDeclared)
			}
		})
	}

	t.Run("sin directorio", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "no-existe")
		gotStore, gotDeclared := declaredArtifactStore(root)
		wantStore, wantDeclared := workspace.DeclaredArtifactStore(root)
		if string(gotStore) != wantStore || gotDeclared != wantDeclared {
			t.Fatalf("(%q, %v) frente a (%q, %v)", gotStore, gotDeclared, wantStore, wantDeclared)
		}
	})
}
