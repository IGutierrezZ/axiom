package livespec

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// writeTree escribe los ficheros de un árbol de prueba (rutas con barras) bajo root.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

const facturacionSpec = `# Especificación: facturacion

## Requirements

### Requirement: Alta de factura (REQ-31.1)

El sistema DEBE dar de alta una factura.

### Requirement: Baja de factura (REQ-31.2)

El sistema DEBE dar de baja una factura.
`

func registryFixture(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	writeTree(t, home, map[string]string{
		"specs/facturacion/spec.md": facturacionSpec,
		"specs/pagos/spec.md":       "# Especificación: pagos\n\n## Requirements\n\n### Requirement: Cobro (REQ-32.7)\n\nCuerpo.\n",
		"changes/inc-1/specs/facturacion/spec.md": "## ADDED Requirements\n\n### Requirement: Nota de abono (REQ-31.5)\n\nCuerpo.\n\n" +
			"## MODIFIED Requirements\n\n### Requirement: Baja de factura (REQ-31.2)\n\nOtro cuerpo.\n\n" +
			"## REMOVED Requirements\n\n### Requirement: Retirada (REQ-31.9)\n\n(Reason: obsoleta)\n\n" +
			"## RENAMED Requirements\n\n### Requirement: Antigua → Nueva (REQ-31.6)\n\n(Reason: nombre)\n",
		"changes/archive/2026-01-01-viejo/specs/facturacion/spec.md": "## ADDED Requirements\n\n### Requirement: Archivada (REQ-31.40)\n\nCuerpo.\n",
		"changes/inc-2/proposal.md":                                  "sin specs\n",
	})
	return home
}

func TestLoadRegistryIndexesSpecsAndActiveDeltasButNotTheArchive(t *testing.T) {
	reg, err := LoadRegistry(registryFixture(t), RegistryOptions{})
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}

	tests := []struct {
		id         string
		wantKind   RefKind
		wantCap    string
		wantChange string
	}{
		{"REQ-31.1", RefSpec, "facturacion", ""},
		{"REQ-32.7", RefSpec, "pagos", ""},
		{"REQ-31.5", RefChange, "facturacion", "inc-1"}, // ADDED
		{"REQ-31.6", RefChange, "facturacion", "inc-1"}, // RENAMED, con el identificador del nombre nuevo
	}
	for _, tt := range tests {
		refs := reg.Lookup(tt.id)
		if len(refs) != 1 {
			t.Errorf("Lookup(%s) = %+v, se esperaba una referencia", tt.id, refs)
			continue
		}
		if refs[0].Kind != tt.wantKind || refs[0].Capability != tt.wantCap || refs[0].Change != tt.wantChange || refs[0].Path == "" {
			t.Errorf("Lookup(%s) = %+v", tt.id, refs[0])
		}
	}
	if refs := reg.Lookup("REQ-31.2"); len(refs) != 2 {
		t.Errorf("REQ-31.2 está en la spec y en el MODIFIED de inc-1: %+v", refs)
	}
	for _, absent := range []string{"REQ-31.40", "REQ-31.9", "REQ-99.1"} {
		if refs := reg.Lookup(absent); len(refs) != 0 {
			t.Errorf("Lookup(%s) = %+v: el archivo y los REMOVED no cuentan", absent, refs)
		}
	}
}

func TestLoadRegistryOptions(t *testing.T) {
	home := registryFixture(t)

	reg, err := LoadRegistry(home, RegistryOptions{ExcludeChange: "inc-1"})
	if err != nil {
		t.Fatal(err)
	}
	if refs := reg.Lookup("REQ-31.5"); len(refs) != 0 {
		t.Errorf("ExcludeChange debe dejar fuera la delta de inc-1: %+v", refs)
	}
	if refs := reg.Lookup("REQ-31.1"); len(refs) != 1 {
		t.Errorf("la spec viva sigue indexada: %+v", refs)
	}

	reg, err = LoadRegistry(home, RegistryOptions{SpecsOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if refs := reg.Lookup("REQ-31.5"); len(refs) != 0 {
		t.Errorf("SpecsOnly no indexa los cambios activos: %+v", refs)
	}
}

func TestLoadRegistryToleratesMissingFolders(t *testing.T) {
	reg, err := LoadRegistry(t.TempDir(), RegistryOptions{})
	if err != nil {
		t.Fatalf("LoadRegistry sobre un planning home vacío: %v", err)
	}
	if got := reg.NextFree(defaultSeries(5)); got != "REQ-5.1" {
		t.Errorf("NextFree(5) = %q, se esperaba REQ-5.1", got)
	}
}

func TestRegistryNextFreeIsTheMaximumOfTheSeriesPlusOne(t *testing.T) {
	reg, err := LoadRegistry(registryFixture(t), RegistryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		series int
		want   string
	}{
		{31, "REQ-31.7"}, // el máximo es REQ-31.6 (RENAMED de inc-1); REQ-31.9 es REMOVED y REQ-31.40 está archivado
		{32, "REQ-32.8"},
		{33, "REQ-33.1"},
	}
	for _, tt := range tests {
		if got := reg.NextFree(defaultSeries(tt.series)); got != tt.want {
			t.Errorf("NextFree(%d) = %q, want %q", tt.series, got, tt.want)
		}
	}
}

func TestRegistryDoesNotConfuseSeriesWithSharedPrefixes(t *testing.T) {
	home := t.TempDir()
	writeTree(t, home, map[string]string{
		"specs/a/spec.md": "# a\n\n### Requirement: Uno (REQ-3.5)\n\ncuerpo\n\n### Requirement: Dos (REQ-30.2)\n\ncuerpo\n",
	})
	reg, err := LoadRegistry(home, RegistryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := reg.NextFree(defaultSeries(3)); got != "REQ-3.6" {
		t.Errorf("NextFree(3) = %q, la serie 30 no es la 3", got)
	}
}

// Con el patrón por defecto la serie es su valor entero, como antes de reqid:
// `REQ-09.4` cuenta en la serie 9 y aparece entre sus referencias.
func TestRegistryTreatsLeadingZerosInTheSeriesAsTheSameSeries(t *testing.T) {
	home := t.TempDir()
	writeTree(t, home, map[string]string{
		"specs/a/spec.md": "# a\n\n### Requirement: Uno (REQ-09.4)\n\ncuerpo\n\n### Requirement: Dos (REQ-9.2)\n\ncuerpo\n",
	})
	reg, err := LoadRegistry(home, RegistryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := reg.NextFree(defaultSeries(9)); got != "REQ-9.5" {
		t.Errorf("NextFree(9) = %q, se esperaba REQ-9.5", got)
	}
	if got := len(reg.LookupIdentity("REQ-9.4")); got != 1 {
		t.Errorf("LookupIdentity(REQ-9.4) tiene %d referencias, se esperaba 1", got)
	}
	if got := len(reg.LookupIdentity("REQ-09.2")); got != 1 {
		t.Errorf("LookupIdentity(REQ-09.2) tiene %d referencias, se esperaba 1: la serie 09 es la 9", got)
	}
}

func TestIDRefWhereNamesTheCapabilityOrTheChange(t *testing.T) {
	spec := IDRef{ID: "REQ-1.1", Kind: RefSpec, Capability: "facturacion"}
	change := IDRef{ID: "REQ-1.2", Kind: RefChange, Capability: "pagos", Change: "inc-7"}
	if got := spec.Where(); got != "la capacidad facturacion" {
		t.Errorf("Where() = %q", got)
	}
	if got := change.Where(); got != "el cambio activo inc-7 (capacidad pagos)" {
		t.Errorf("Where() = %q", got)
	}
}

// defaultSeries es la serie `REQ-n.*` del patrón por defecto.
func defaultSeries(n int) reqid.Series {
	series, _ := reqid.Default().Series(fmt.Sprintf("REQ-%d.0", n))
	return series
}

// REQ-47.1 / D11: un provisional en el título de un requisito de un cambio activo no es un
// identificador: no se indexa, no colisiona y no ocupa números.
func TestRegistryIgnoresAProvisionalOfAnActiveChange(t *testing.T) {
	home := t.TempDir()
	writeTree(t, home, map[string]string{
		"specs/pagos/spec.md":               "# E\n\n## Requirements\n\n### Requirement: Cobro (REQ-30.1)\n\nCuerpo.\n",
		"changes/inc-a/specs/pagos/spec.md": "## ADDED Requirements\n\n### Requirement: Alta (prov:30:1)\n\nCuerpo.\n\n### Requirement: Baja (prov:30:2)\n\nCuerpo.\n",
		"changes/inc-b/specs/pagos/spec.md": "## ADDED Requirements\n\n### Requirement: Otra (prov:30:1)\n\nCuerpo.\n",
	})
	reg, err := LoadRegistry(home, RegistryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"prov:30:1", "prov:30:2", ""} {
		if refs := reg.Lookup(token); len(refs) != 0 {
			t.Errorf("Lookup(%q) = %v: un provisional no se indexa", token, refs)
		}
	}
	if got := reg.NextFree(defaultSeries(30)); got != "REQ-30.2" {
		t.Errorf("NextFree = %s: un provisional no ocupa números", got)
	}
}
