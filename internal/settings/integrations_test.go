package settings

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

const adoPrefix = "integrations[" + manifest.AzureDevOpsID + "]."

// RF-TOP-21: las filas del plugin salen del manifiesto, en su orden, con `enabled` primero,
// y el selector conserva los guiones del identificador.
func TestIntegrationsCatalogIsDerivedFromTheManifest(t *testing.T) {
	m, ok := manifest.Lookup(manifest.AzureDevOpsID)
	if !ok {
		t.Fatal("el catálogo de manifiestos no declara azure-devops")
	}
	want := []string{adoPrefix + "enabled"}
	for _, f := range m.Config {
		want = append(want, adoPrefix+f.Key)
	}
	for _, doc := range []string{monoDoc, multiDoc} {
		snap := readSnapshot(t, doc)
		var got []string
		for _, e := range snap.Entries {
			if e.Plugin != "" {
				got = append(got, e.Key)
				if e.Section != "integrations" || !e.Editable || e.Plugin != manifest.AzureDevOpsID {
					t.Errorf("fila de integración mal formada: %+v", e)
				}
				if len(e.Topologies) != 2 {
					t.Errorf("%s debe aplicar a ambas topologías: %v", e.Key, e.Topologies)
				}
			}
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("claves = %v, quiero %v", got, want)
		}
	}
}

// RF-TOP-21: tipos, vocabulario, formato, cotas y obligatoriedad salen del manifiesto.
func TestIntegrationsCatalogMapsTypesAndConstraints(t *testing.T) {
	snap := readSnapshot(t, monoDoc)
	cases := map[string]ValueType{
		"enabled": TypeBool, "organization": TypeString, "item_keys": TypeStringList,
		"req_tag_max": TypeInt, "workdays": TypeStringList, "req_tag_prefix": TypeString,
	}
	for field, typ := range cases {
		if got := mustEntry(t, snap, adoPrefix+field).Type; got != typ {
			t.Errorf("%s: tipo %s, quiero %s", field, got, typ)
		}
	}
	if e := mustEntry(t, snap, adoPrefix+"organization"); !e.Required || e.Description == "" {
		t.Errorf("organization debe ser obligatoria y llevar descripción: %+v", e)
	}
	if e := mustEntry(t, snap, adoPrefix+"task_state_start"); e.Required || e.Format != string(manifest.FormatNonEmpty) || e.Default != "Active" {
		t.Errorf("task_state_start: %+v", e)
	}
	e := mustEntry(t, snap, adoPrefix+"req_tag_max")
	if e.Min == nil || e.Max == nil || *e.Min != 1 || *e.Max != 100 || e.Default != "50" {
		t.Errorf("req_tag_max debe publicar sus cotas: %+v", e)
	}
	if got := mustEntry(t, snap, adoPrefix+"workdays").Allowed; len(got) != 7 || got[0] != "mon" {
		t.Errorf("workdays debe publicar la enumeración del manifiesto: %v", got)
	}
}

// RF-WEB-18: ningún secreto del manifiesto es una clave editable ni aparece en el catálogo.
func TestIntegrationsCatalogNeverExposesSecrets(t *testing.T) {
	snap := readSnapshot(t, monoDoc+"integrations:\n  azure-devops:\n    enabled: false\n    pat: nunca-visible\n")
	m, _ := manifest.Lookup(manifest.AzureDevOpsID)
	for _, e := range snap.Entries {
		for _, s := range m.Secrets {
			if strings.HasSuffix(e.Key, "]."+s.Name) {
				t.Errorf("el secreto %q figura en el catálogo: %s", s.Name, e.Key)
			}
		}
		if v, ok := e.Value.(string); ok && strings.Contains(v, "nunca-visible") {
			t.Errorf("%s expone el valor de un secreto", e.Key)
		}
	}
	p := newProject(t, monoDoc)
	p.rejects(t, Patch{adoPrefix + "pat": "x"}, CodeUnknownKey)
}

// RF-TOP-22: los valores declarados se leen tal como están escritos.
func TestIntegrationsCatalogReadsDeclaredValues(t *testing.T) {
	snap := readSnapshot(t, monoDoc+"integrations:\n  azure-devops:\n    enabled: true\n    organization: acme\n    req_tag_max: 7\n    workdays:\n      - mon\n      - tue\n")
	if e := mustEntry(t, snap, adoPrefix+"enabled"); !e.Declared || e.Value != true {
		t.Errorf("enabled: %+v", e)
	}
	if e := mustEntry(t, snap, adoPrefix+"organization"); e.Value != "acme" {
		t.Errorf("organization: %+v", e)
	}
	if e := mustEntry(t, snap, adoPrefix+"req_tag_max"); e.Value != 7 {
		t.Errorf("req_tag_max: %+v", e)
	}
	if e := mustEntry(t, snap, adoPrefix+"workdays"); len(e.Value.([]string)) != 2 {
		t.Errorf("workdays: %+v", e)
	}
	if e := mustEntry(t, snap, adoPrefix+"team"); e.Declared || e.Value != nil {
		t.Errorf("team no está declarada: %+v", e)
	}
}

// Un plugin fuera del catálogo no genera filas y su valor no llega a la lectura.
func TestIntegrationsCatalogIgnoresUnknownPlugins(t *testing.T) {
	snap := readSnapshot(t, monoDoc+"integrations:\n  jira:\n    token: nunca-visible\n")
	for _, e := range snap.Entries {
		if strings.Contains(e.Key, "jira") {
			t.Errorf("fila de un plugin desconocido: %s", e.Key)
		}
	}
}

// RF-TOP-22: plugin desconocido, campo ajeno al manifiesto y claves sin selector son unknown_key.
func TestIntegrationsPatchRejectsUnknownKeys(t *testing.T) {
	for _, key := range []string{
		"integrations[jira].enabled", adoPrefix + "inventada", "integrations[azure-devops]", "integrations[].enabled",
		"integrations[azure-devops]enabled", "integrations", "integrations.enabled",
	} {
		if _, err := patched(t, monoDoc, Patch{key: "x"}); err == nil || err.Code != CodeUnknownKey {
			t.Errorf("%q debe ser unknown_key: %v", key, err)
		}
	}
}

// RF-TOP-22: el valor se convierte al tipo del manifiesto.
func TestIntegrationsPatchChecksValueTypes(t *testing.T) {
	cases := []Patch{
		{adoPrefix + "enabled": "true"},
		{adoPrefix + "req_tag_max": "5"},
		{adoPrefix + "req_tag_max": 2.5},
		{adoPrefix + "organization": true},
		{adoPrefix + "workdays": []any{"mon", 3}},
	}
	for _, patch := range cases {
		if _, err := patched(t, monoDoc, patch); err == nil || err.Code != CodeInvalidValue {
			t.Errorf("%v debe ser invalid_value: %v", patch, err)
		}
	}
}

// RF-TOP-21: el bloque y el plugin se crean tras acceptance_tests con estilos tipados.
func TestIntegrationsPatchCreatesTheBlockAfterAcceptanceTests(t *testing.T) {
	doc := multiDoc + "acceptance_tests:\n  enabled: true\n"
	// El bloque va tras acceptance_tests aunque `roles` ya figure antes.
	got := mustPatch(t, doc, Patch{adoPrefix + "enabled": true, adoPrefix + "organization": "acme", adoPrefix + "workdays": []string{"mon"}})
	want := "acceptance_tests:\n  enabled: true\n\nintegrations:\n  azure-devops:\n    enabled: true\n    organization: \"acme\"\n    workdays:\n      - \"mon\""
	if !strings.Contains(got, want) {
		t.Errorf("resultado:\n%s", got)
	}
	if got := mustPatch(t, doc, Patch{adoPrefix + "req_tag_max": 9}); !strings.Contains(got, "req_tag_max: 9\n") {
		t.Errorf("el entero va sin comillas:\n%s", got)
	}
}

// RF-TOP-21: un plugin en estilo de flujo pasa a estilo de bloque al añadirle claves.
func TestIntegrationsPatchTurnsFlowStyleIntoBlockStyle(t *testing.T) {
	got := mustPatch(t, monoDoc+"integrations:\n  azure-devops: {}\n", Patch{adoPrefix + "enabled": false})
	if strings.Contains(got, "{") || !strings.Contains(got, "  azure-devops:\n    enabled: false\n") {
		t.Errorf("resultado:\n%s", got)
	}
	got = mustPatch(t, monoDoc+"integrations:\n  azure-devops:\n", Patch{adoPrefix + "team": "t"})
	if !strings.Contains(got, "  azure-devops:\n    team: \"t\"\n") {
		t.Errorf("un plugin nulo se convierte en mapeo:\n%s", got)
	}
}

// RF-TOP-21: retirar todas las claves de un plugin borra su entrada y el bloque vacío.
func TestIntegrationsPatchRemovesThePluginAndTheEmptyBlock(t *testing.T) {
	doc := monoDoc + "integrations:\n  azure-devops:\n    enabled: true\n    team: t\n"
	got := mustPatch(t, doc, Patch{adoPrefix + "team": nil})
	if !strings.Contains(got, "enabled: true") || strings.Contains(got, "team") {
		t.Errorf("retirar una clave:\n%s", got)
	}
	got = mustPatch(t, doc, Patch{adoPrefix + "enabled": nil, adoPrefix + "team": nil})
	if strings.Contains(got, "integrations") || strings.Contains(got, "azure-devops") {
		t.Errorf("retirar el plugin entero debe borrar el bloque:\n%s", got)
	}
	if same := mustPatch(t, monoDoc, Patch{adoPrefix + "team": nil}); same != monoDoc {
		t.Errorf("retirar lo que no existe no cambia nada:\n%s", same)
	}
}

// RF-TOP-22: ancla, alias y `<<` dentro de integrations hacen el documento no escribible.
func TestIntegrationsConstructsBlockWriting(t *testing.T) {
	for name, block := range map[string]string{
		"ancla":  "integrations:\n  azure-devops: &a\n    enabled: false\n",
		"alias":  "integrations:\n  azure-devops:\n    enabled: false\n  otro: *a\n",
		"mezcla": "integrations:\n  azure-devops:\n    <<: {enabled: false}\n",
	} {
		doc := monoDoc + block
		if name == "alias" {
			doc = monoDoc + "integrations:\n  base: &a\n    enabled: false\n  azure-devops: *a\n"
		}
		snap := readSnapshot(t, doc)
		if snap.Writable || snap.WriteBlockedReason == "" {
			t.Errorf("%s: el documento no debe ser escribible", name)
		}
		p := newProject(t, doc)
		p.rejects(t, Patch{adoPrefix + "team": "t"}, CodeUnsupportedDocument)
	}
}

// RF-TOP-22: Write conserva los comentarios y el resultado lo acepta el decodificador estricto.
func TestIntegrationsWriteRoundTripsThroughTheStrictDecoder(t *testing.T) {
	p := newProject(t, monoDoc+"# integraciones del equipo\nintegrations:\n  azure-devops:\n    enabled: false # apagada\n")
	result, err := p.write(t, Patch{
		adoPrefix + "enabled": true, adoPrefix + "organization": "acme", adoPrefix + "project": "p", adoPrefix + "team": "t",
		adoPrefix + "workday_hours": 6, adoPrefix + "workdays": []any{"mon", "fri"},
	})
	if err != nil || !result.Changed {
		t.Fatalf("Write: %+v, %v", result, err)
	}
	text := string(p.bytes(t))
	if !strings.Contains(text, "# integraciones del equipo") || !strings.Contains(text, "# apagada") {
		t.Errorf("se perdieron comentarios:\n%s", text)
	}
	cfg, perr := workspace.ParseConfig(p.bytes(t))
	if perr != nil {
		t.Fatalf("ParseConfig: %v", perr)
	}
	if cfg.Integrations == nil || len(cfg.Integrations.Plugins) != 1 {
		t.Fatalf("integrations decodificadas: %+v", cfg.Integrations)
	}
	snap, _ := Read(p.path)
	if e := mustEntry(t, snap, adoPrefix+"organization"); e.Value != "acme" {
		t.Errorf("relectura: %+v", e)
	}
}

// RF-TOP-22: la obligatoriedad es condicional a `enabled` y la decide el decodificador.
func TestIntegrationsWriteRequiredFieldsAreCheckedByTheDecoder(t *testing.T) {
	p := newProject(t, monoDoc)
	typed := p.rejects(t, Patch{adoPrefix + "enabled": true}, CodeValidationFailed)
	if len(typed.Violations) == 0 {
		t.Error("la violación debe nombrar los campos obligatorios")
	}
	p.rejects(t, Patch{adoPrefix + "enabled": true, adoPrefix + "organization": "a", adoPrefix + "project": "p", adoPrefix + "team": "t", adoPrefix + "req_tag_max": 1000}, CodeValidationFailed)
	// Con `enabled: false` los obligatorios no se exigen: se puede guardar un borrador.
	if _, err := p.write(t, Patch{adoPrefix + "enabled": false, adoPrefix + "organization": "acme"}); err != nil {
		t.Fatalf("borrador desactivado: %v", err)
	}
}

// RF-TOP-21: retirar un plugin entero deja el documento sin el bloque.
func TestIntegrationsWriteRemovesTheWholePlugin(t *testing.T) {
	p := newProject(t, monoDoc+"integrations:\n  azure-devops:\n    enabled: false\n    team: t\n")
	if _, err := p.write(t, Patch{adoPrefix + "enabled": nil, adoPrefix + "team": nil}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if strings.Contains(string(p.bytes(t)), "integrations") {
		t.Errorf("queda el bloque:\n%s", p.bytes(t))
	}
}
