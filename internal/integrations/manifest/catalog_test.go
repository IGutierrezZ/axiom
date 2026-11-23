package manifest

import (
	"reflect"
	"strings"
	"testing"
)

func fakeManifest(id string) Manifest {
	m := validManifest()
	m.ID = id
	return m
}

func TestCatalogDefaultIsExactlyAzureDevOps(t *testing.T) {
	got := Catalog()
	if len(got) != 1 || got[0].ID != AzureDevOpsID {
		t.Fatalf("el catálogo cerrado de esta versión es exactamente azure-devops, obtuvo %v", got)
	}
	if !reflect.DeepEqual(got[0], AzureDevOps()) {
		t.Fatal("la entrada del catálogo debe ser igual a AzureDevOps()")
	}
	if _, ok := Lookup("fake"); ok {
		t.Fatal("Lookup no debe encontrar un plugin ajeno al catálogo")
	}
	if _, ok := Lookup(AzureDevOpsID); !ok {
		t.Fatal("Lookup debe encontrar azure-devops")
	}
}

func TestSetCatalogForTestInstallsAndRestores(t *testing.T) {
	inner := &fakeT{}
	SetCatalogForTest(inner, fakeManifest("fake"), fakeManifest("other"))

	got := Catalog()
	if len(got) != 2 || got[0].ID != "fake" || got[1].ID != "other" {
		t.Fatalf("Catalog() = %v", got)
	}
	m, ok := Lookup("other")
	if !ok || m.ID != "other" {
		t.Fatalf("Lookup(other) = %v, %v", m, ok)
	}
	if _, ok := Lookup("missing"); ok {
		t.Fatal("Lookup(missing) debe fallar")
	}

	inner.runCleanups()
	if got := Catalog(); len(got) != 1 || got[0].ID != AzureDevOpsID {
		t.Fatalf("Cleanup debe restaurar el catálogo por defecto (azure-devops), obtuvo %v", got)
	}
}

func TestCatalogReturnsDeepCopy(t *testing.T) {
	m := fakeManifest("fake")
	m.Config = []ConfigField{{Key: "k", Type: FieldString, Enum: []string{"a"}}}
	SetCatalogForTest(t, m)

	got := Catalog()
	got[0].Capabilities[0] = CapTags
	got[0].Config[0].Enum[0] = "z"
	got[0].ID = "mutado"

	again := Catalog()
	if again[0].ID != "fake" || again[0].Capabilities[0] != CapItemsRead || again[0].Config[0].Enum[0] != "a" {
		t.Fatal("mutar el resultado de Catalog no debe afectar al catálogo")
	}
	l, _ := Lookup("fake")
	l.Capabilities[0] = CapTags
	if again2 := Catalog(); again2[0].Capabilities[0] != CapItemsRead {
		t.Fatal("Lookup también debe devolver una copia")
	}
}

func TestValidateCatalog(t *testing.T) {
	t.Run("válido", func(t *testing.T) {
		if err := ValidateCatalog([]Manifest{fakeManifest("a"), fakeManifest("b")}); err != nil {
			t.Fatalf("catálogo válido: %v", err)
		}
	})
	t.Run("identificador repetido", func(t *testing.T) {
		err := ValidateCatalog([]Manifest{fakeManifest("a"), fakeManifest("a")})
		if err == nil || !strings.Contains(err.Error(), "repetido") {
			t.Fatalf("identificador repetido: %v", err)
		}
	})
	t.Run("misma variable de entorno para dos pares", func(t *testing.T) {
		// (a-b, c) y (a, b_c) dan AXIOM_INTEGRATION_A_B_C.
		first := fakeManifest("a-b")
		first.Secrets = []SecretSpec{{Name: "c"}}
		second := fakeManifest("a")
		second.Secrets = []SecretSpec{{Name: "b_c"}}
		err := ValidateCatalog([]Manifest{first, second})
		if err == nil || !strings.Contains(err.Error(), "AXIOM_INTEGRATION_A_B_C") {
			t.Fatalf("colisión de variable de entorno: %v", err)
		}
	})
	t.Run("manifiesto inválido", func(t *testing.T) {
		bad := fakeManifest("Mal Id")
		if err := ValidateCatalog([]Manifest{bad}); err == nil {
			t.Fatal("un manifiesto inválido invalida el catálogo")
		}
	})
}

func TestSetCatalogForTestRejectsInvalidCatalog(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("instalar un catálogo inválido debe entrar en pánico para que la prueba falle")
		}
	}()
	SetCatalogForTest(&fakeT{}, fakeManifest("a"), fakeManifest("a"))
}

func TestCatalogEqualsItselfAfterClone(t *testing.T) {
	m := fakeManifest("fake")
	SetCatalogForTest(t, m)
	got, _ := Lookup("fake")
	if !reflect.DeepEqual(got, m) {
		t.Fatalf("Lookup debe devolver un manifiesto igual al instalado:\n%#v\n%#v", got, m)
	}
}

// fakeT implementa la interfaz mínima de SetCatalogForTest para comprobar la restauración.
type fakeT struct{ cleanups []func() }

func (f *fakeT) Helper()           {}
func (f *fakeT) Cleanup(fn func()) { f.cleanups = append(f.cleanups, fn) }
func (f *fakeT) runCleanups() {
	for i := len(f.cleanups) - 1; i >= 0; i-- {
		f.cleanups[i]()
	}
	f.cleanups = nil
}
