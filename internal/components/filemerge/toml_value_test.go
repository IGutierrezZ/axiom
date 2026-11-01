package filemerge

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// decodeTOMLKey analiza el documento con el analizador de referencia y devuelve la
// clave raíz como cadena.
func decodeTOMLKey(t *testing.T, document, key string) string {
	t.Helper()
	var parsed map[string]any
	if _, err := toml.Decode(document, &parsed); err != nil {
		t.Fatalf("el TOML resultante no se puede analizar: %v\n%s", err, document)
	}
	value, ok := parsed[key].(string)
	if !ok {
		t.Fatalf("la clave %q no es una cadena en:\n%s", key, document)
	}
	return value
}

// FormatTOMLString debe producir un TOML que, al leerse, devuelva exactamente el
// texto original, también con comillas, barras invertidas, delimitadores TOML,
// caracteres de control y Unicode.
func TestFormatTOMLStringRoundTrip(t *testing.T) {
	cases := map[string]string{
		"una línea":                    "solo una línea con \"comillas\" y \\ barra",
		"vacío":                        "",
		"multilínea simple":            "primera\nsegunda\n",
		"con tabulador":                "a\tb\nc\n",
		"con comillas dobles":          "dice \"hola\" y \"\"adiós\"\"\nfin\n",
		"tres comillas dobles":         "esto \"\"\" cierra\nfin\n",
		"cuatro comillas dobles":       "x \"\"\"\" y\nfin\n",
		"comillas simples triples":     "esto ''' cierra\nfin\n",
		"ambas triples":                "''' y \"\"\"\nfin\n",
		"barras invertidas":            "ruta C:\\Users\\yo\\nuevo \\n \\t \\u0041\nfin\n",
		"termina en comilla doble":     "linea\n\"",
		"termina en dos dobles":        "linea\n\"\"",
		"termina en comilla simple":    "linea\n'",
		"termina en newline y comilla": "linea\n\"\n",
		"con control":                  "a\x01b\x7fc\nfin\n",
		"unicode":                      "ñandú — «ok» ✓ 日本語 😀\nfin\n",
		"cabecera de tabla en texto":   "[features]\nmulti_agent = false\n[mcp_servers.engram]\n",
		"empieza por salto":            "\nsegunda línea\n",
		"marcadores html":              "<!-- axiom:persona -->\ntexto\n<!-- /axiom:persona -->\n",
		"crlf":                         "a\r\nb\r\n",
	}
	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			document := "developer_instructions = " + FormatTOMLString(value) + "\n"
			want := strings.ReplaceAll(value, "\r\n", "\n")
			if got := decodeTOMLKey(t, document, "developer_instructions"); got != want {
				t.Errorf("texto leído = %q, want %q\n%s", got, want, document)
			}
		})
	}
}

// El texto sin caracteres problemáticos va como cadena literal, sin escapes, que es
// la forma legible.
func TestFormatTOMLStringPrefersReadableLiteral(t *testing.T) {
	got := FormatTOMLString("primera\n\"segunda\" \\ tercera\n")
	if !strings.HasPrefix(got, "'''\n") || !strings.HasSuffix(got, "'''") || strings.Contains(got, `\\`) {
		t.Errorf("se esperaba una cadena literal sin escapes, got %q", got)
	}
	if basic := FormatTOMLString("x\n''' y\n"); !strings.HasPrefix(basic, `"""`) {
		t.Errorf("con tres comillas simples debe pasar a la forma básica, got %q", basic)
	}
}

func TestTopLevelTOMLString(t *testing.T) {
	value, declared, isString, err := TopLevelTOMLString("model = \"x\"\ndeveloper_instructions = '''\nhola\n'''\n[features]\ndeveloper_instructions = 3\n", "developer_instructions")
	if err != nil || !declared || !isString || value != "hola\n" {
		t.Errorf("got value=%q declared=%v isString=%v err=%v", value, declared, isString, err)
	}
	if _, declared, _, err = TopLevelTOMLString("model = \"x\"\n", "developer_instructions"); err != nil || declared {
		t.Errorf("clave ausente: declared=%v err=%v", declared, err)
	}
	if _, declared, isString, _ = TopLevelTOMLString("developer_instructions = [\"a\"]\n", "developer_instructions"); !declared || isString {
		t.Errorf("una lista debe contar como declarada y no cadena: declared=%v isString=%v", declared, isString)
	}
	if _, _, _, err = TopLevelTOMLString("developer_instructions = \n[", "developer_instructions"); err == nil {
		t.Error("un TOML inválido debe devolver error")
	}
	if _, declared, _, err = TopLevelTOMLString("", "x"); err != nil || declared {
		t.Errorf("documento vacío: declared=%v err=%v", declared, err)
	}
}

func TestValidateTOML(t *testing.T) {
	if err := ValidateTOML(""); err != nil {
		t.Errorf("un fichero vacío es válido: %v", err)
	}
	if err := ValidateTOML("model = \"x\"\n[features]\nmulti_agent = true\n"); err != nil {
		t.Errorf("TOML válido rechazado: %v", err)
	}
	if err := ValidateTOML("model = \n"); err == nil {
		t.Error("TOML inválido aceptado")
	}
}

func TestUpsertTopLevelTOMLValueInsertsBeforeFirstTable(t *testing.T) {
	existing := "# comentario del usuario\nmodel = \"gpt\"\n\n[features]\nmulti_agent = true\n"
	got := UpsertTopLevelTOMLValue(existing, "developer_instructions", FormatTOMLString("uno\ndos\n"))

	if decodeTOMLKey(t, got, "developer_instructions") != "uno\ndos\n" {
		t.Errorf("el valor no se lee de la raíz:\n%s", got)
	}
	var parsed map[string]any
	if _, err := toml.Decode(got, &parsed); err != nil {
		t.Fatal(err)
	}
	if features, _ := parsed["features"].(map[string]any); features["multi_agent"] != true {
		t.Errorf("la tabla [features] debe conservarse:\n%s", got)
	}
	if _, leaked := parsed["features"].(map[string]any)["developer_instructions"]; leaked {
		t.Errorf("la clave cayó dentro de [features]:\n%s", got)
	}
	if !strings.Contains(got, "# comentario del usuario") || !strings.Contains(got, `model = "gpt"`) {
		t.Errorf("se perdió contenido del usuario:\n%s", got)
	}
	if strings.Index(got, "developer_instructions") > strings.Index(got, "[features]") {
		t.Errorf("la clave debe ir antes de la primera tabla:\n%s", got)
	}
}

func TestUpsertTopLevelTOMLValueAppendsWithoutTables(t *testing.T) {
	got := UpsertTopLevelTOMLValue("model = \"x\"\n", "developer_instructions", FormatTOMLString("hola"))
	if want := "model = \"x\"\ndeveloper_instructions = \"hola\"\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := UpsertTopLevelTOMLValue("", "k", `"v"`); got != "k = \"v\"\n" {
		t.Errorf("documento vacío: %q", got)
	}
}

// Una clave existente con un valor multilínea se sustituye en su sitio y no se
// confunde con las líneas de su contenido, aunque parezcan claves o tablas.
func TestUpsertTopLevelTOMLValueReplacesMultilineInPlace(t *testing.T) {
	existing := "first = 1\n" +
		"developer_instructions = '''\n" +
		"[features]\n" +
		"developer_instructions = \"falso\"\n" +
		"'''\n" +
		"last = 2\n" +
		"\n[agents]\nmax_depth = 2\n"

	got := UpsertTopLevelTOMLValue(existing, "developer_instructions", FormatTOMLString("nuevo\ntexto\n"))

	if decodeTOMLKey(t, got, "developer_instructions") != "nuevo\ntexto\n" {
		t.Errorf("el valor no se sustituyó:\n%s", got)
	}
	if strings.Index(got, "first") > strings.Index(got, "developer_instructions") || strings.Index(got, "developer_instructions") > strings.Index(got, "last") {
		t.Errorf("la clave debe conservar su posición entre first y last:\n%s", got)
	}
	var parsed map[string]any
	if _, err := toml.Decode(got, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["first"] != int64(1) || parsed["last"] != int64(2) {
		t.Errorf("se perdieron las claves vecinas: %v", parsed)
	}
	if again := UpsertTopLevelTOMLValue(got, "developer_instructions", FormatTOMLString("nuevo\ntexto\n")); again != got {
		t.Errorf("repetir la operación debe ser idempotente:\n%s\n----\n%s", got, again)
	}
}

// Una asignación con el mismo nombre dentro de una tabla no es la de la raíz.
func TestUpsertTopLevelTOMLValueIgnoresKeysInsideTables(t *testing.T) {
	existing := "model = \"x\"\n[features]\ndeveloper_instructions = \"de la tabla\"\n"
	got := UpsertTopLevelTOMLValue(existing, "developer_instructions", `"raíz"`)

	if decodeTOMLKey(t, got, "developer_instructions") != "raíz" {
		t.Errorf("la clave de raíz no se escribió:\n%s", got)
	}
	var parsed map[string]any
	if _, err := toml.Decode(got, &parsed); err != nil {
		t.Fatal(err)
	}
	if features, _ := parsed["features"].(map[string]any); features["developer_instructions"] != "de la tabla" {
		t.Errorf("la clave de la tabla debe quedar intacta:\n%s", got)
	}
}

// Una clave cuyo prefijo coincide con otra (model frente a model_instructions_file)
// no se confunde con ella.
func TestUpsertTopLevelTOMLValueDoesNotMatchLongerKey(t *testing.T) {
	existing := "model_instructions_file = \"a.md\"\n"
	got := UpsertTopLevelTOMLValue(existing, "model", `"x"`)
	if !strings.Contains(got, `model_instructions_file = "a.md"`) || decodeTOMLKey(t, got, "model") != "x" {
		t.Errorf("got:\n%s", got)
	}
}

// Los ayudantes de tablas no deben confundir una línea de una cadena multilínea (el
// texto de developer_instructions) con una cabecera o con una clave: el texto del
// usuario con una línea "[features]" o "multi_agent = false" se conserva exacto.
func TestTableHelpersIgnoreLinesInsideMultilineStrings(t *testing.T) {
	userText := "texto\n[features]\nmulti_agent = false\n[mcp_servers.engram]\ncommand = \"falso\"\n[agents]\nmax_depth = 9\n"
	base := "developer_instructions = " + FormatTOMLString(userText) + "\n\n[features]\nmulti_agent = false\n"

	for name, apply := range map[string]func(string) string{
		"UpsertTOMLTableKey features": func(s string) string { return UpsertTOMLTableKey(s, "features", "multi_agent", "true") },
		"UpsertTOMLTableKey agents":   func(s string) string { return UpsertTOMLTableKey(s, "agents", "max_depth", "2") },
		"UpsertCodexEngramBlock":      func(s string) string { return UpsertCodexEngramBlock(s, "engram") },
		"UpsertCodexMCPServerBlock": func(s string) string {
			return UpsertCodexMCPServerBlock(s, "engram", "engram", []string{"mcp"})
		},
		"UpsertCodexRemoteMCPServerBlock": func(s string) string {
			return UpsertCodexRemoteMCPServerBlock(s, "engram", "https://example.test/mcp")
		},
		"RemoveTOMLTableKeys": func(s string) string { return RemoveTOMLTableKeys(s, "features", []string{"multi_agent"}) },
		"RemoveTOMLTable":     func(s string) string { return RemoveTOMLTable(s, "features") },
	} {
		t.Run(name, func(t *testing.T) {
			got := apply(base)
			if decodeTOMLKey(t, got, "developer_instructions") != userText {
				t.Errorf("el texto multilínea cambió:\n%s", got)
			}
			if again := apply(got); again != got {
				t.Errorf("repetir la operación debe ser idempotente:\n%s\n----\n%s", got, again)
			}
		})
	}

	// Efecto esperado sobre la tabla real, no sobre el texto.
	var parsed map[string]any
	if _, err := toml.Decode(UpsertTOMLTableKey(base, "features", "multi_agent", "true"), &parsed); err != nil {
		t.Fatal(err)
	}
	if features, _ := parsed["features"].(map[string]any); features["multi_agent"] != true {
		t.Errorf("features.multi_agent = %v, quiero true", features["multi_agent"])
	}
}

// Una cabecera dentro de una lista abierta no es una tabla.
func TestTableHelpersIgnoreHeaderLikeLinesInsideArrays(t *testing.T) {
	base := "[mcp_servers.engram]\ncommand = \"viejo\"\nargs = [\n[\"anidada\"]\n]\n\n[features]\nmulti_agent = true\n"
	got := UpsertCodexEngramBlock(base, "engram")

	var parsed map[string]any
	if _, err := toml.Decode(got, &parsed); err != nil {
		t.Fatalf("TOML inválido: %v\n%s", err, got)
	}
	servers, _ := parsed["mcp_servers"].(map[string]any)
	engram, _ := servers["engram"].(map[string]any)
	if engram["command"] != "engram" {
		t.Errorf("el bloque de Engram debe haberse reemplazado entero: %v", engram)
	}
	if features, _ := parsed["features"].(map[string]any); features["multi_agent"] != true {
		t.Errorf("la tabla [features] debe conservarse: %v", parsed["features"])
	}
}

func TestUpsertTopLevelTOMLValueNormalizesCRLF(t *testing.T) {
	got := UpsertTopLevelTOMLValue("model = \"x\"\r\n[features]\r\nmulti_agent = true\r\n", "k", `"v"`)
	if strings.Contains(got, "\r") {
		t.Errorf("el resultado debe usar LF: %q", got)
	}
}
