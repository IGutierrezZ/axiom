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
	want := []Capability{
		CapItemsRead, CapDiscovery, CapTags, CapAttachments,
		CapItemsCreate, CapRelations, CapEstimation, CapTime, CapItemsTransition, CapAssignment,
	}
	if !reflect.DeepEqual(m.Capabilities, want) {
		t.Fatalf("capacidades = %v, esperaba exactamente %v", m.Capabilities, want)
	}
	for _, c := range m.Capabilities {
		if c == CapVCSLinks {
			t.Errorf("no debe declarar la capacidad %q", c)
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
	if len(got) != 12 {
		t.Fatalf("claves = %v, esperaba 12 (organization, project, team, item_keys y las ocho del ciclo de vida)", m.Config)
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

func TestAzureDevOpsManifestLifecycleKeys(t *testing.T) {
	got := map[string]ConfigField{}
	for _, f := range AzureDevOps().Config {
		got[f.Key] = f
	}
	for key, want := range map[string]struct {
		typ    FieldType
		format ValueFormat
		def    any
	}{
		"task_state_start":          {FieldString, FormatNonEmpty, "Active"},
		"task_state_apply_approved": {FieldString, FormatNonEmpty, "Tech Review"},
		"task_state_close":          {FieldString, FormatNonEmpty, nil},
		"work_item_state_archive":   {FieldString, FormatNonEmpty, nil},
		"req_tag_prefix":            {FieldString, FormatTagPrefix, "req:"},
		"req_tag_max":               {FieldInt, "", 50},
		"workday_hours":             {FieldInt, "", 8},
		"workdays":                  {FieldStringList, FormatWeekdaySet, []string{"mon", "tue", "wed", "thu", "fri"}},
	} {
		f, ok := got[key]
		if !ok {
			t.Errorf("falta la clave %q", key)
			continue
		}
		if f.Type != want.typ || f.Format != want.format || f.Required || !reflect.DeepEqual(f.Default, want.def) {
			t.Errorf("%s = %+v, esperaba tipo %s, formato %q, opcional y por defecto %v", key, f, want.typ, want.format, want.def)
		}
	}
	for key, rng := range map[string][2]int{"req_tag_max": {1, 100}, "workday_hours": {1, 24}} {
		f := got[key]
		if f.Min == nil || f.Max == nil || *f.Min != rng[0] || *f.Max != rng[1] {
			t.Errorf("%s: rango = %v..%v, esperaba %v", key, f.Min, f.Max, rng)
		}
	}
}

func TestAzureDevOpsManifestCapabilities(t *testing.T) {
	m := AzureDevOps()
	for _, c := range []Capability{CapItemsCreate, CapRelations, CapEstimation, CapTime, CapItemsTransition, CapAssignment} {
		found := false
		for _, have := range m.Capabilities {
			found = found || have == c
		}
		if !found {
			t.Errorf("falta la capacidad %q", c)
		}
	}
}

func TestAzureDevOpsCloneKeepsBounds(t *testing.T) {
	a := AzureDevOps()
	c := a.Clone()
	for i := range c.Config {
		if c.Config[i].Max != nil {
			*c.Config[i].Max = 999
		}
	}
	if !reflect.DeepEqual(a, AzureDevOps()) {
		t.Fatal("mutar la copia no debe alterar el original")
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
