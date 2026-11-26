package manifest

import (
	"strings"
	"testing"
)

func TestCheckTagPrefixTable(t *testing.T) {
	for name, tc := range map[string]struct {
		in   string
		fail bool
	}{
		"por defecto":             {"req:", false},
		"un carácter":             {"r", false},
		"cien caracteres":         {strings.Repeat("a", 100), false},
		"ciento un caracteres":    {strings.Repeat("a", 101), true},
		"vacío":                   {"", true},
		"coma":                    {"a,b", true},
		"punto y coma":            {"a;b", true},
		"axiom en minúsculas":     {"axiom:x", true},
		"axiom en mayúsculas":     {"AXIOM:x", true},
		"axiom-task":              {"Axiom-Task:", true},
		"axiom sin dos puntos":    {"axiomatic", false},
		"axiom-tasks sin guion :": {"axiom-tasks", false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := CheckTagPrefix(tc.in) != ""; got != tc.fail {
				t.Fatalf("CheckTagPrefix(%q) falla = %v, esperaba %v", tc.in, got, tc.fail)
			}
		})
	}
}

func TestCheckNonEmpty(t *testing.T) {
	if CheckNonEmpty("  ") == "" || CheckNonEmpty("") == "" {
		t.Fatal("una cadena en blanco debe fallar")
	}
	if CheckNonEmpty(" Active ") != "" {
		t.Fatal("una cadena con texto debe pasar")
	}
}

func TestParseWeekdaySetTable(t *testing.T) {
	for name, tc := range map[string]struct {
		in      []string
		problem int
	}{
		"laborables":  {[]string{"mon", "tue", "wed", "thu", "fri"}, 0},
		"semana":      {[]string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}, 0},
		"vacía":       {nil, 1},
		"repetido":    {[]string{"mon", "mon"}, 1},
		"desconocido": {[]string{"mon", "lunes"}, 1},
		"mayúsculas":  {[]string{"Mon"}, 1},
		"dos fallos":  {[]string{"x", "mon", "mon"}, 2},
	} {
		t.Run(name, func(t *testing.T) {
			if got := len(ParseWeekdaySet(tc.in)); got != tc.problem {
				t.Fatalf("problemas = %d, esperaba %d", got, tc.problem)
			}
		})
	}
}

func TestManifestLifecycleFieldValidation(t *testing.T) {
	one, hundred := 1, 100
	field := func(f ConfigField) Manifest {
		m := validManifest()
		f.Key, f.Description = "clave_nueva", "x"
		m.Config = append(m.Config, f)
		return m
	}
	for name, f := range map[string]ConfigField{
		"int con rango":       {Type: FieldInt, Default: 5, Min: &one, Max: &hundred},
		"string non_empty":    {Type: FieldString, Format: FormatNonEmpty, Default: "Active"},
		"string tag_prefix":   {Type: FieldString, Format: FormatTagPrefix, Default: "req:"},
		"lista weekday_set":   {Type: FieldStringList, Format: FormatWeekdaySet, Default: []string{"mon"}},
		"formato sin default": {Type: FieldString, Format: FormatNonEmpty},
	} {
		if err := field(f).Validate(); err != nil {
			t.Errorf("%s: no debe fallar: %v", name, err)
		}
	}
	zero := 0
	for name, f := range map[string]ConfigField{
		"rango en string":             {Type: FieldString, Min: &one},
		"min mayor que max":           {Type: FieldInt, Min: &hundred, Max: &one},
		"default fuera de rango":      {Type: FieldInt, Default: 101, Max: &hundred},
		"default bajo el mínimo":      {Type: FieldInt, Default: 0, Min: &one},
		"non_empty en lista":          {Type: FieldStringList, Format: FormatNonEmpty},
		"weekday_set en string":       {Type: FieldString, Format: FormatWeekdaySet},
		"non_empty con default vacío": {Type: FieldString, Format: FormatNonEmpty, Default: " "},
		"tag_prefix con axiom:":       {Type: FieldString, Format: FormatTagPrefix, Default: "axiom:x"},
		"weekday_set repetido":        {Type: FieldStringList, Format: FormatWeekdaySet, Default: []string{"mon", "mon"}},
		"weekday_set vacío":           {Type: FieldStringList, Format: FormatWeekdaySet, Default: []string{}},
		"min cero sin default":        {Type: FieldBool, Min: &zero},
	} {
		if err := field(f).Validate(); err == nil {
			t.Errorf("%s: esperaba un error de validación", name)
		}
	}
}
