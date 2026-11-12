package kickoff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func readSealedKickoff(t *testing.T, changeRoot string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(changeRoot, KickoffFileName))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// REQ-29.13: la instantánea `sdd:` sobrevive a sellar y releer; con iso falso o
// sin bloque, omitempty no escribe ni `sdd:` ni `iso`.
func TestSDDSnapshot(t *testing.T) {
	t.Run("iso true se escribe y se relee", func(t *testing.T) {
		changeRoot := t.TempDir()
		k := sampleKickoff("x")
		k.SDD = &SDDSnapshot{ISO: true}
		sealed, wrote, err := Seal(changeRoot, k)
		if err != nil || !wrote || sealed.SDD == nil || !sealed.SDD.ISO {
			t.Fatalf("Seal() = (%+v, %v, %v)", sealed.SDD, wrote, err)
		}
		data := readSealedKickoff(t, changeRoot)
		if !strings.Contains(data, "sdd:") || !strings.Contains(data, "iso: true") {
			t.Errorf("kickoff.yaml debe llevar el bloque sdd con iso:\n%s", data)
		}
		loaded, err := Load(changeRoot)
		if err != nil || loaded == nil || !loaded.IsISO() {
			t.Fatalf("Load() = (%+v, %v), se esperaba un cambio ISO", loaded, err)
		}
	})

	t.Run("iso false no escribe sdd", func(t *testing.T) {
		changeRoot := t.TempDir()
		k := sampleKickoff("x")
		k.SDD = &SDDSnapshot{ISO: false}
		if _, _, err := Seal(changeRoot, k); err != nil {
			t.Fatal(err)
		}
		data := readSealedKickoff(t, changeRoot)
		if strings.Contains(data, "sdd:") || strings.Contains(data, "iso") {
			t.Errorf("con iso falso no se escribe sdd: ni iso:\n%s", data)
		}
	})

	t.Run("sin bloque se lee sin clave", func(t *testing.T) {
		changeRoot := t.TempDir()
		if _, _, err := Seal(changeRoot, sampleKickoff("x")); err != nil {
			t.Fatal(err)
		}
		if data := readSealedKickoff(t, changeRoot); strings.Contains(data, "sdd:") {
			t.Errorf("sin instantánea no se emite sdd:\n%s", data)
		}
		loaded, err := Load(changeRoot)
		if err != nil || loaded == nil || loaded.SDD != nil || loaded.IsISO() {
			t.Fatalf("Load() = (%+v, %v), se esperaba un kickoff sin instantánea", loaded, err)
		}
	})

	t.Run("un decodificador con el esquema previo lo ignora", func(t *testing.T) {
		k := sampleKickoff("x")
		k.SDD = &SDDSnapshot{ISO: true}
		data, err := yaml.Marshal(k)
		if err != nil {
			t.Fatal(err)
		}
		var previous struct {
			Schema string       `yaml:"schema"`
			Change string       `yaml:"change"`
			Git    *GitSnapshot `yaml:"git,omitempty"`
		}
		if err := yaml.Unmarshal(data, &previous); err != nil || previous.Change != "x" {
			t.Fatalf("un decodificador sin SDD debe ignorar sdd: (%+v, %v)", previous, err)
		}
	})
}

// REQ-29.13: IsISO es verdadero solo con bloque e iso verdadero.
func TestIsISO(t *testing.T) {
	tests := []struct {
		name string
		sdd  *SDDSnapshot
		want bool
	}{
		{"sin bloque", nil, false},
		{"iso false", &SDDSnapshot{}, false},
		{"iso true", &SDDSnapshot{ISO: true}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (Kickoff{SDD: tt.sdd}).IsISO(); got != tt.want {
				t.Errorf("IsISO() = %v, se esperaba %v", got, tt.want)
			}
		})
	}
}
