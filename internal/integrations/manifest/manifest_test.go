package manifest

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// validManifest devuelve un manifiesto correcto que cada caso de la tabla modifica.
func validManifest() Manifest {
	return Manifest{
		Schema:       SchemaV1,
		ID:           "fake",
		Title:        "Plugin de prueba en memoria",
		Capabilities: []Capability{CapItemsRead, CapItemsTransition},
		Events:       []EventType{EventGateSpecApproved, EventArchive},
		Config: []ConfigField{
			{Key: "project", Type: FieldString, Required: true, Description: "Proyecto del tracker"},
		},
		Secrets:  []SecretSpec{{Name: "token", Description: "Token de acceso", Required: true}},
		Identity: IdentitySpec{Source: IdentityGitActor},
	}
}

func TestValidateAcceptsValidManifest(t *testing.T) {
	if err := validManifest().Validate(); err != nil {
		t.Fatalf("un manifiesto válido no debe fallar: %v", err)
	}
}

func TestValidateIdentifier(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
	}{
		{"fake", false},
		{"azure-devops", false},
		{"a1-b2", false},
		{"Azure DevOps", true},
		{"", true},
		{"-x", true},
		{"a--b", true},
		{"x-", true},
		{"1abc", true},
		{strings.Repeat("a", 32), false},
		{strings.Repeat("a", 33), true},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			m := validManifest()
			m.ID = tt.id
			err := m.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate(id=%q) = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSchema(t *testing.T) {
	m := validManifest()
	m.Schema = "otro/v1"
	if err := m.Validate(); err == nil || !strings.Contains(err.Error(), SchemaV1) {
		t.Fatalf("un esquema distinto debe rechazarse nombrando %q, obtuvo %v", SchemaV1, err)
	}
}

func TestValidateCapabilitiesAndEvents(t *testing.T) {
	t.Run("sin capacidades", func(t *testing.T) {
		m := validManifest()
		m.Capabilities = nil
		if err := m.Validate(); err == nil {
			t.Fatal("al menos una capacidad")
		}
	})
	t.Run("sin eventos", func(t *testing.T) {
		m := validManifest()
		m.Events = nil
		if err := m.Validate(); err == nil {
			t.Fatal("al menos un evento")
		}
	})
	t.Run("capacidad repetida", func(t *testing.T) {
		m := validManifest()
		m.Capabilities = []Capability{CapTags, CapTags}
		if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "repetida") {
			t.Fatalf("capacidad repetida: %v", err)
		}
	})
	t.Run("evento repetido", func(t *testing.T) {
		m := validManifest()
		m.Events = []EventType{EventArchive, EventArchive}
		if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "repetido") {
			t.Fatalf("evento repetido: %v", err)
		}
	})
	t.Run("capacidad fuera de la lista nombra la inválida y las admitidas", func(t *testing.T) {
		m := validManifest()
		m.Capabilities = []Capability{"sprints"}
		err := m.Validate()
		if err == nil || !strings.Contains(err.Error(), "sprints") || !strings.Contains(err.Error(), "items.read") {
			t.Fatalf("debe nombrar sprints y listar las admitidas: %v", err)
		}
	})
	t.Run("evento fuera de la lista nombra el inválido y los admitidos", func(t *testing.T) {
		m := validManifest()
		m.Events = []EventType{"change.close"}
		err := m.Validate()
		if err == nil || !strings.Contains(err.Error(), "change.close") || !strings.Contains(err.Error(), "gate.spec.approved") {
			t.Fatalf("debe nombrar change.close y listar los admitidos: %v", err)
		}
	})
}

func TestValidateConfigKeys(t *testing.T) {
	tests := []struct {
		name    string
		field   ConfigField
		secrets []SecretSpec
		wantErr string
	}{
		{name: "válida", field: ConfigField{Key: "project", Type: FieldString}},
		{name: "con guion bajo", field: ConfigField{Key: "area_path", Type: FieldString}},
		{name: "mayúsculas", field: ConfigField{Key: "Project", Type: FieldString}, wantErr: "debe cumplir"},
		{name: "con guion", field: ConfigField{Key: "area-path", Type: FieldString}, wantErr: "debe cumplir"},
		{name: "demasiado larga", field: ConfigField{Key: strings.Repeat("a", 65), Type: FieldString}, wantErr: "supera"},
		{name: "reservada", field: ConfigField{Key: "enabled", Type: FieldBool}, wantErr: "reservada"},
		{name: "igual que un secreto", field: ConfigField{Key: "cred", Type: FieldString}, secrets: []SecretSpec{{Name: "cred"}}, wantErr: "coincide con un secreto"},
		{name: "segmento token", field: ConfigField{Key: "token", Type: FieldString}, wantErr: "parece un secreto"},
		{name: "segmento secret", field: ConfigField{Key: "client_secret", Type: FieldString}, wantErr: "parece un secreto"},
		{name: "segmento secrets", field: ConfigField{Key: "secrets", Type: FieldString}, wantErr: "parece un secreto"},
		{name: "segmento password", field: ConfigField{Key: "db_password", Type: FieldString}, wantErr: "parece un secreto"},
		{name: "segmento passwd", field: ConfigField{Key: "passwd", Type: FieldString}, wantErr: "parece un secreto"},
		{name: "segmento pwd", field: ConfigField{Key: "pwd", Type: FieldString}, wantErr: "parece un secreto"},
		{name: "segmento pat", field: ConfigField{Key: "azure_pat", Type: FieldString}, wantErr: "parece un secreto"},
		{name: "segmento credential", field: ConfigField{Key: "credential", Type: FieldString}, wantErr: "parece un secreto"},
		{name: "segmento credentials", field: ConfigField{Key: "my_credentials", Type: FieldString}, wantErr: "parece un secreto"},
		{name: "segmento apikey", field: ConfigField{Key: "apikey", Type: FieldString}, wantErr: "parece un secreto"},
		{name: "par api_key", field: ConfigField{Key: "service_api_key", Type: FieldString}, wantErr: "parece un secreto"},
		{name: "api sin key es válida", field: ConfigField{Key: "api_version", Type: FieldString}},
		{name: "subcadena no cuenta como segmento", field: ConfigField{Key: "patrol", Type: FieldString}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := validManifest()
			m.Config = []ConfigField{tt.field}
			m.Secrets = []SecretSpec{{Name: "access"}}
			if tt.secrets != nil {
				m.Secrets = tt.secrets
			}
			err := m.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("no debe fallar: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("esperaba un error con %q, obtuvo %v", tt.wantErr, err)
			}
		})
	}

	t.Run("clave repetida", func(t *testing.T) {
		m := validManifest()
		m.Config = []ConfigField{{Key: "a", Type: FieldString}, {Key: "a", Type: FieldInt}}
		if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "repetida") {
			t.Fatalf("clave repetida: %v", err)
		}
	})
}

func TestValidateDefaultsAndEnums(t *testing.T) {
	tests := []struct {
		name    string
		field   ConfigField
		wantErr bool
	}{
		{"default string", ConfigField{Key: "k", Type: FieldString, Default: "x"}, false},
		{"default string con tipo erróneo", ConfigField{Key: "k", Type: FieldString, Default: 3}, true},
		{"default bool", ConfigField{Key: "k", Type: FieldBool, Default: true}, false},
		{"default bool con tipo erróneo", ConfigField{Key: "k", Type: FieldBool, Default: "true"}, true},
		{"default int", ConfigField{Key: "k", Type: FieldInt, Default: 3}, false},
		{"default int tras JSON", ConfigField{Key: "k", Type: FieldInt, Default: float64(3)}, false},
		{"default int con decimal", ConfigField{Key: "k", Type: FieldInt, Default: 3.5}, true},
		{"default lista", ConfigField{Key: "k", Type: FieldStringList, Default: []string{"a"}}, false},
		{"default lista tras JSON", ConfigField{Key: "k", Type: FieldStringList, Default: []any{"a"}}, false},
		{"default lista con elemento erróneo", ConfigField{Key: "k", Type: FieldStringList, Default: []any{1}}, true},
		{"enum en string", ConfigField{Key: "k", Type: FieldString, Enum: []string{"a", "b"}}, false},
		{"enum en lista", ConfigField{Key: "k", Type: FieldStringList, Enum: []string{"a"}}, false},
		{"enum en bool", ConfigField{Key: "k", Type: FieldBool, Enum: []string{"a"}}, true},
		{"enum en int", ConfigField{Key: "k", Type: FieldInt, Enum: []string{"a"}}, true},
		{"tipo desconocido", ConfigField{Key: "k", Type: "float"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := validManifest()
			m.Config = []ConfigField{tt.field}
			err := m.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSecretsAndIdentity(t *testing.T) {
	t.Run("secretos repetidos", func(t *testing.T) {
		m := validManifest()
		m.Secrets = []SecretSpec{{Name: "token"}, {Name: "token"}}
		if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "repetido") {
			t.Fatalf("secreto repetido: %v", err)
		}
	})
	t.Run("nombre de secreto inválido", func(t *testing.T) {
		m := validManifest()
		m.Secrets = []SecretSpec{{Name: "Token"}}
		if err := m.Validate(); err == nil {
			t.Fatal("nombre de secreto inválido")
		}
	})
	t.Run("identidad con origen desconocido", func(t *testing.T) {
		m := validManifest()
		m.Identity.Source = "ldap"
		if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "ldap") {
			t.Fatalf("origen de identidad desconocido: %v", err)
		}
	})
	t.Run("identidad con secreto no declarado", func(t *testing.T) {
		m := validManifest()
		m.Identity = IdentitySpec{Source: IdentityPlugin, Secret: "otro"}
		if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "otro") {
			t.Fatalf("identity.secret no declarado: %v", err)
		}
	})
	t.Run("identidad con secreto declarado", func(t *testing.T) {
		m := validManifest()
		m.Identity = IdentitySpec{Source: IdentityPlugin, Secret: "token"}
		if err := m.Validate(); err != nil {
			t.Fatalf("identity.secret declarado debe valer: %v", err)
		}
	})
}

func TestValidateAccumulatesProblems(t *testing.T) {
	m := validManifest()
	m.ID = "Mal Id"
	m.Capabilities = nil
	m.Events = []EventType{"x"}
	err := m.Validate()
	var ve *ValidationError
	if e, ok := err.(*ValidationError); ok {
		ve = e
	}
	if ve == nil || len(ve.Problems) < 3 {
		t.Fatalf("debe acumular todos los problemas, obtuvo %#v", err)
	}
}

func TestManifestJSONRoundTrip(t *testing.T) {
	m := validManifest()
	m.Config = append(m.Config, ConfigField{Key: "mode", Type: FieldString, Enum: []string{"a", "b"}, Default: "a"})
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"schema":"axiom.integration-manifest/v1"`) {
		t.Fatalf("el JSON debe llevar el campo schema: %s", raw)
	}
	var back Manifest
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(m, back) {
		t.Fatalf("la ida y vuelta no es estable:\n%#v\n%#v", m, back)
	}
	raw2, _ := json.Marshal(back)
	if string(raw) != string(raw2) {
		t.Fatalf("la serialización no es estable:\n%s\n%s", raw, raw2)
	}
}

func TestEnvVarName(t *testing.T) {
	if got := EnvVarName("azure-devops", "pat"); got != "AXIOM_INTEGRATION_AZURE_DEVOPS_PAT" {
		t.Fatalf("EnvVarName = %q", got)
	}
	if got := EnvVarName("fake", "token"); got != "AXIOM_INTEGRATION_FAKE_TOKEN" {
		t.Fatalf("EnvVarName = %q", got)
	}
}

func TestClosedListsAreStable(t *testing.T) {
	if got := Events(); len(got) != 6 || got[0] != EventChangeStart || got[5] != EventArchive {
		t.Fatalf("Events() = %v", got)
	}
	if got := Capabilities(); len(got) != 11 || got[0] != CapItemsRead || got[10] != CapDiscovery {
		t.Fatalf("Capabilities() = %v", got)
	}
}

func TestCloneIsDeep(t *testing.T) {
	m := validManifest()
	m.Config = []ConfigField{{Key: "k", Type: FieldStringList, Enum: []string{"a"}, Default: []string{"a"}}}
	c := m.Clone()
	c.Config[0].Enum[0] = "z"
	c.Config[0].Default.([]string)[0] = "z"
	c.Capabilities[0] = CapTags
	if m.Config[0].Enum[0] != "a" || m.Config[0].Default.([]string)[0] != "a" || m.Capabilities[0] != CapItemsRead {
		t.Fatal("Clone debe devolver una copia profunda")
	}
}

func TestManifestRefAliasPattern(t *testing.T) {
	for alias, wantErr := range map[string]bool{
		"":                  false, // opcional
		"ado":               false,
		"ab":                false,
		"a":                 true,
		"1ab":               true,
		"Ado":               true,
		"a-b":               true,
		"abcdefghijklmnop":  false, // 16 caracteres
		"abcdefghijklmnopq": true,  // 17 caracteres
	} {
		m := validManifest()
		m.RefAlias = alias
		err := m.Validate()
		if (err != nil) != wantErr {
			t.Errorf("ref_alias %q: error = %v, esperaba error=%v", alias, err, wantErr)
		}
	}
}

func TestManifestDefaultMustMatchFormat(t *testing.T) {
	field := func(def any, f ValueFormat, typ FieldType) Manifest {
		m := validManifest()
		m.Config = append(m.Config, ConfigField{Key: "item_keys", Type: typ, Format: f, Default: def, Description: "x"})
		return m
	}
	if err := field([]string{"Bug=BUG"}, FormatTypePrefixPairs, FieldStringList).Validate(); err != nil {
		t.Fatalf("un default válido no debe fallar: %v", err)
	}
	if err := field([]any{"Bug=BUG"}, FormatTypePrefixPairs, FieldStringList).Validate(); err != nil {
		t.Fatalf("un default []any válido no debe fallar: %v", err)
	}
	for name, m := range map[string]Manifest{
		"default sin igual":   field([]string{"Bug"}, FormatTypePrefixPairs, FieldStringList),
		"formato desconocido": field(nil, ValueFormat("otro"), FieldStringList),
		"formato en string":   field(nil, FormatTypePrefixPairs, FieldString),
		"prefijo inválido":    field([]string{"Bug=B-G"}, FormatTypePrefixPairs, FieldStringList),
		"tipo repetido":       field([]string{"Bug=A", "bug=B"}, FormatTypePrefixPairs, FieldStringList),
	} {
		if err := m.Validate(); err == nil {
			t.Errorf("%s: esperaba un error de validación", name)
		}
	}
}
