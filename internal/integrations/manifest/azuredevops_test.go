package manifest

import (
	"reflect"
	"testing"
)

func TestAzureDevOpsManifestIsValid(t *testing.T) {
	m := AzureDevOps()
	if err := m.Validate(); err != nil {
		t.Fatalf("el manifiesto de azure-devops debe validar: %v", err)
	}
	if m.ID != AzureDevOpsID || AzureDevOpsID != "azure-devops" {
		t.Fatalf("ID = %q", m.ID)
	}
}

func TestAzureDevOpsManifestDeclaresOnlyDeliveredCapabilities(t *testing.T) {
	m := AzureDevOps()
	want := []Capability{CapItemsRead, CapDiscovery, CapTags, CapAttachments}
	if !reflect.DeepEqual(m.Capabilities, want) {
		t.Fatalf("capacidades = %v, esperaba exactamente %v", m.Capabilities, want)
	}
	for _, forbidden := range []Capability{
		CapItemsTransition, CapItemsCreate, CapRelations, CapAssignment, CapEstimation,
		CapTime, CapVCSLinks,
	} {
		for _, c := range m.Capabilities {
			if c == forbidden {
				t.Errorf("no debe declarar la capacidad %q", forbidden)
			}
		}
	}
}

func TestAzureDevOpsManifestListensOnlyToChangeStart(t *testing.T) {
	m := AzureDevOps()
	if !reflect.DeepEqual(m.Events, []EventType{EventChangeStart}) {
		t.Fatalf("eventos = %v, esperaba solo change.start", m.Events)
	}
}

func TestAzureDevOpsManifestConfigKeys(t *testing.T) {
	m := AzureDevOps()
	got := map[string]ConfigField{}
	for _, f := range m.Config {
		got[f.Key] = f
	}
	if len(got) != 4 {
		t.Fatalf("claves = %v, esperaba organization, project, team e item_keys", m.Config)
	}
	for _, key := range []string{"organization", "project", "team"} {
		f, ok := got[key]
		if !ok {
			t.Fatalf("falta la clave %q", key)
		}
		if f.Type != FieldString || !f.Required {
			t.Errorf("%s: tipo %q, obligatoria %v; esperaba string obligatoria", key, f.Type, f.Required)
		}
	}
}

func TestAzureDevOpsManifestSecretAndIdentity(t *testing.T) {
	m := AzureDevOps()
	if len(m.Secrets) != 1 || m.Secrets[0].Name != "pat" || !m.Secrets[0].Required {
		t.Fatalf("secretos = %v, esperaba un único secreto obligatorio pat", m.Secrets)
	}
	if m.Identity.Source != IdentityPlugin || m.Identity.Secret != "pat" {
		t.Fatalf("identidad = %+v, esperaba {plugin, pat}", m.Identity)
	}
	if got := EnvVarName(m.ID, "pat"); got != "AXIOM_INTEGRATION_AZURE_DEVOPS_PAT" {
		t.Fatalf("variable de entorno = %q", got)
	}
}

func TestAzureDevOpsReturnsFreshValue(t *testing.T) {
	a := AzureDevOps()
	a.Capabilities[0] = CapVCSLinks
	a.Config[0].Key = "mutado"
	a.Secrets[0].Name = "mutado"
	b := AzureDevOps()
	if b.Capabilities[0] != CapItemsRead || b.Config[0].Key == "mutado" || b.Secrets[0].Name != "pat" {
		t.Fatal("cada llamada debe devolver un valor nuevo e independiente")
	}
}

func TestAzureDevOpsManifestCapabilities(t *testing.T) {
	m := AzureDevOps()
	want := []Capability{CapItemsRead, CapDiscovery, CapTags, CapAttachments}
	if !reflect.DeepEqual(m.Capabilities, want) {
		t.Fatalf("capacidades = %v, esperaba exactamente %v", m.Capabilities, want)
	}
}

func TestAzureDevOpsManifestRefAlias(t *testing.T) {
	if got := AzureDevOps().RefAlias; got != "ado" {
		t.Fatalf("ref_alias = %q, esperaba ado", got)
	}
}

func TestAzureDevOpsManifestItemKeys(t *testing.T) {
	var f *ConfigField
	m := AzureDevOps()
	for i := range m.Config {
		if m.Config[i].Key == "item_keys" {
			f = &m.Config[i]
		}
	}
	if f == nil {
		t.Fatal("falta la clave item_keys")
	}
	if f.Type != FieldStringList || f.Format != FormatTypePrefixPairs || f.Required {
		t.Fatalf("item_keys = %+v, esperaba string_list opcional con formato type_prefix_pairs", *f)
	}
	if !reflect.DeepEqual(f.Default, []string{"User Story=US", "Bug=BUG"}) {
		t.Fatalf("default = %v", f.Default)
	}
}
