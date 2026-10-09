package livingdoc

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	baselineCanonicalConfig = "workspace:\n  name: ejemplo-canonico\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: Backend\n    repositories:\n      - path: repo-backend\n"
	baselineDeclared        = "baseline:\n  documents:\n    - domain: modelo-de-datos\n      path: 03_Modelo_Datos.md\n    - domain: flujos-de-negocio\n      path: docs/04_Flujos_Negocio.md\n"
	baselineInvalid         = "baseline:\n  documents: []\n"
)

// canonicalWithBaseline monta el espacio canónico con la baseline dada en el
// axiom.yaml del repositorio de especificaciones y un cambio con delta spec.
func canonicalWithBaseline(t *testing.T, baseline string) workspacetest.Workspace {
	t.Helper()
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), baselineCanonicalConfig+baseline)
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "inc-99-ejemplo", "spec.md"), sampleSpec)
	return ws
}

// snapshotAll devuelve los nombres y el contenido de todo lo que hay bajo root.
func snapshotAll(t *testing.T, root string) map[string]string {
	t.Helper()
	tree := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			tree[path] = "<dir>"
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		tree[path] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("recorrer %s: %v", root, err)
	}
	return tree
}

// REQ-1.1, REQ-24.25: con baseline del proyecto el catálogo no escanea, devuelve
// Specs vacío y la baseline externa con los dominios y las rutas absolutas.
func TestGetCatalogWithAProjectBaselineReportsItAsExternal(t *testing.T) {
	ws := canonicalWithBaseline(t, baselineDeclared)
	// Una spec viva en el planning home no se indexa: la baseline es del proyecto.
	writeLivingSpec(t, filepath.Join(ws.Home, "specs"), "dominio-heredado")

	for name, start := range map[string]string{
		"carpeta maestra":                 ws.Master,
		"repositorio de especificaciones": ws.Specs,
		"repositorio de código":           filepath.Join(ws.Backend, "src"),
	} {
		t.Run(name, func(t *testing.T) {
			catalog, err := NewService(start, nil, nil).GetCatalog(context.Background())
			if err != nil {
				t.Fatalf("GetCatalog = %v", err)
			}
			if catalog.Specs == nil || len(catalog.Specs) != 0 {
				t.Errorf("Specs = %#v, se esperaba una lista vacía", catalog.Specs)
			}
			want := &ExternalBaseline{Documents: []ExternalBaselineDocument{
				{Domain: "modelo-de-datos", Path: filepath.Join(ws.Specs, "03_Modelo_Datos.md")},
				{Domain: "flujos-de-negocio", Path: filepath.Join(ws.Specs, "docs", "04_Flujos_Negocio.md")},
			}}
			if !reflect.DeepEqual(catalog.ExternalBaseline, want) {
				t.Errorf("ExternalBaseline = %+v, se esperaba %+v", catalog.ExternalBaseline, want)
			}
			if catalog.TotalRequirements != 0 || catalog.TotalScenarios != 0 {
				t.Errorf("no se indexa nada: %+v", catalog)
			}

			raw, err := json.Marshal(catalog)
			if err != nil {
				t.Fatal(err)
			}
			for _, fragment := range []string{`"specs":[]`, `"external_baseline":{"documents":[`, `"domain":"modelo-de-datos"`} {
				if !strings.Contains(string(raw), fragment) {
					t.Errorf("el JSON no contiene %s: %s", fragment, raw)
				}
			}
		})
	}
}

// Sin baseline del proyecto el catálogo no incluye la clave aditiva.
func TestGetCatalogWithoutAProjectBaselineHasNoExternalBaseline(t *testing.T) {
	ws := workspacetest.Canonical(t)
	writeLivingSpec(t, filepath.Join(ws.Home, "specs"), "workspace-topology")
	catalog, err := NewService(ws.Master, nil, nil).GetCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if catalog.ExternalBaseline != nil {
		t.Errorf("ExternalBaseline = %+v, se esperaba nil", catalog.ExternalBaseline)
	}
	raw, _ := json.Marshal(catalog)
	if strings.Contains(string(raw), "external_baseline") {
		t.Errorf("el JSON no debe llevar external_baseline: %s", raw)
	}
	if got := catalogDomains(catalog); len(got) != 1 {
		t.Errorf("dominios = %v", got)
	}
}

// REQ-24.25: el detalle de un dominio declarado nombra la baseline externa y su
// ruta; un dominio no declarado se resuelve como siempre.
func TestGetSpecDetailOfADeclaredDomain(t *testing.T) {
	ws := canonicalWithBaseline(t, baselineDeclared)
	writeLivingSpec(t, filepath.Join(ws.Home, "specs"), "dominio-heredado")
	svc := NewService(ws.Master, nil, nil)

	_, _, err := svc.GetSpecDetail("modelo-de-datos")
	var external *ExternalBaselineError
	if !errors.As(err, &external) {
		t.Fatalf("GetSpecDetail = %v, se esperaba un *ExternalBaselineError", err)
	}
	if external.Domain != "modelo-de-datos" || external.Path != filepath.Join(ws.Specs, "03_Modelo_Datos.md") || external.ConfigPath != filepath.Join(ws.Specs, "axiom.yaml") {
		t.Errorf("campos = %+v", external)
	}
	for _, want := range []string{"baseline externa", "modelo-de-datos", external.Path, external.ConfigPath} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el mensaje no contiene %q:\n%v", want, err)
		}
	}

	if _, _, err := svc.GetSpecDetail("dominio-no-declarado"); err == nil || errors.As(err, &external) {
		t.Errorf("un dominio no declarado se resuelve como siempre (no encontrado), no como baseline externa: %v", err)
	}
	if entry, content, err := svc.GetSpecDetail("dominio-heredado"); err != nil || entry == nil || content == "" {
		t.Errorf("un dominio presente en el planning home sigue leyéndose: (%v, %q, %v)", entry, content, err)
	}
}

// REQ-1.2, REQ-2.1, REQ-3.1, REQ-3.4, REQ-24.23: Sync y ColdStart se rechazan con
// la baseline del proyecto, sin escribir nada.
func TestSyncAndColdStartRejectAProjectBaseline(t *testing.T) {
	ws := canonicalWithBaseline(t, baselineDeclared)
	before := snapshotAll(t, ws.Master)
	svc := NewService(ws.Master, nil, nil)

	calls := map[string]func() error{
		"Sync": func() error {
			_, err := svc.Sync(context.Background())
			return err
		},
		"ColdStart": func() error {
			_, err := svc.ColdStart("inc-99-ejemplo", "")
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			err := call()
			var second *workspace.SecondBaselineError
			if !errors.As(err, &second) {
				t.Fatalf("%s = %v, se esperaba un *workspace.SecondBaselineError", name, err)
			}
			if second.Ownership != workspace.BaselineProject || !reflect.DeepEqual(second.Domains, []string{"modelo-de-datos", "flujos-de-negocio"}) {
				t.Errorf("campos = %+v", second)
			}
			for _, want := range []string{"baseline de especificaciones es del proyecto", "modelo-de-datos, flujos-de-negocio", "retira el bloque `baseline:`"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("el mensaje no contiene %q:\n%v", want, err)
				}
			}
			if after := snapshotAll(t, ws.Master); !reflect.DeepEqual(before, after) {
				t.Errorf("el rechazo no debe escribir nada")
			}
			for _, absent := range []string{filepath.Join(ws.Home, "specs"), filepath.Join(ws.Home, "INDEX.md")} {
				if _, statErr := os.Stat(absent); !errors.Is(statErr, fs.ErrNotExist) {
					t.Errorf("no debe existir %s (stat = %v)", absent, statErr)
				}
			}
		})
	}
}

// REQ-24.18: con una baseline indeterminada, Sync y ColdStart rechazan con la
// causa, y el catálogo se lee como siempre con un aviso que la nombra.
func TestAnIndeterminateBaselineRejectsWritesAndWarnsOnReads(t *testing.T) {
	ws := canonicalWithBaseline(t, baselineInvalid)
	writeLivingSpec(t, filepath.Join(ws.Home, "specs"), "dominio-heredado")
	before := snapshotAll(t, ws.Master)
	svc := NewService(ws.Master, nil, nil)

	if _, err := svc.Sync(context.Background()); err == nil {
		t.Fatal("Sync debe rechazarse con una baseline indeterminada")
	} else {
		var second *workspace.SecondBaselineError
		if !errors.As(err, &second) || second.Ownership != workspace.BaselineIndeterminate || second.Cause == nil {
			t.Fatalf("Sync = %v, se esperaba un *workspace.SecondBaselineError indeterminado con causa", err)
		}
		for _, want := range []string{"al menos un documento", "axiom workspace validate --path"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("el mensaje no contiene %q:\n%v", want, err)
			}
		}
	}
	if _, err := svc.ColdStart("inc-99-ejemplo", ""); err == nil {
		t.Fatal("ColdStart debe rechazarse con una baseline indeterminada")
	}
	if after := snapshotAll(t, ws.Master); !reflect.DeepEqual(before, after) {
		t.Errorf("los rechazos no deben escribir nada")
	}

	catalog, err := svc.GetCatalog(context.Background())
	if err != nil {
		t.Fatalf("GetCatalog = %v", err)
	}
	if got := catalogDomains(catalog); len(got) != 1 || got[0] != "dominio-heredado" || catalog.ExternalBaseline != nil {
		t.Errorf("con baseline indeterminada el catálogo se lee como siempre: dominios %v, externa %+v", got, catalog.ExternalBaseline)
	}
	if len(catalog.Warnings) != 1 || !strings.Contains(catalog.Warnings[0], "No se puede determinar") || !strings.Contains(catalog.Warnings[0], "al menos un documento") {
		t.Errorf("Warnings = %q, se esperaba el aviso con la causa", catalog.Warnings)
	}
}

// Sin baseline del proyecto, Sync y ColdStart operan como hasta ahora.
func TestSyncAndColdStartWithoutAProjectBaselineStillWrite(t *testing.T) {
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "inc-99-ejemplo", "spec.md"), sampleSpec)
	svc := NewService(ws.Master, nil, nil)

	if _, err := svc.ColdStart("inc-99-ejemplo", "ejemplo"); err != nil {
		t.Fatalf("ColdStart = %v", err)
	}
	for _, present := range []string{filepath.Join(ws.Home, "specs", "ejemplo", "spec.md"), filepath.Join(ws.Home, "INDEX.md")} {
		if _, err := os.Stat(present); err != nil {
			t.Errorf("debe existir %s: %v", present, err)
		}
	}
	if _, err := svc.Sync(context.Background()); err != nil {
		t.Fatalf("Sync = %v", err)
	}
}

// Un rechazo de la resolución se propaga: no hay respuesta de baseline.
func TestCatalogAndGuardsPropagateALocateRejection(t *testing.T) {
	ws := workspacetest.Canonical(t, workspacetest.WithBrokenPointer())
	svc := NewService(ws.Master, nil, nil)
	var locateErr *workspace.LocateError
	if _, err := svc.GetCatalog(context.Background()); !errors.As(err, &locateErr) {
		t.Errorf("GetCatalog = %v, se esperaba un *workspace.LocateError", err)
	}
	if _, err := svc.Sync(context.Background()); !errors.As(err, &locateErr) {
		t.Errorf("Sync = %v, se esperaba un *workspace.LocateError", err)
	}
	if _, err := svc.ColdStart("inc-99-ejemplo", ""); !errors.As(err, &locateErr) {
		t.Errorf("ColdStart = %v, se esperaba un *workspace.LocateError", err)
	}
}
