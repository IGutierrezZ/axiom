package gitref

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// refNameCase es un caso de tabla: want vacío significa «válido»; en otro caso
// es un fragmento que debe aparecer en el motivo del rechazo.
type refNameCase struct {
	name  string
	value string
	want  string
}

func runRefNameCases(t *testing.T, validate func(string) error, cases []refNameCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validate(tc.value)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("%q debía ser válido y dio: %v", tc.value, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("%q debía rechazarse por %q y se aceptó", tc.value, tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("%q: el motivo %q no contiene %q", tc.value, err.Error(), tc.want)
			}
		})
	}
}

// REQ-25.3 y REQ-25.26: ValidRefName aplica las reglas de `git check-ref-format`
// a un nombre de rama.
func TestValidRefName(t *testing.T) {
	runRefNameCases(t, ValidRefName, []refNameCase{
		{"rama simple", "main", ""},
		{"rama con jerarquía", "feat/mi-cambio", ""},
		{"rama con tres niveles", "equipo/inc/inc-25", ""},
		{"puntos y guiones bajos dentro de un componente", "release/v1.2_rc", ""},
		{"arroba dentro del nombre", "usuario@dominio", ""},
		{"llave sin arroba delante", "a{b}", ""},
		{"componente que contiene .lock sin terminar en él", "a.lockfile/b", ""},
		{"caracteres no ASCII", "feat/ñandú", ""},

		{"vacío", "", "no puede estar vacío"},
		{"arroba sola", "@", "«@»"},
		{"dos puntos seguidos", "a..b", "«..»"},
		{"dos puntos seguidos en un componente intermedio", "a/b..c/d", "«..»"},
		{"secuencia @{", "a@{b", "«@{»"},
		{"espacio", "a b", "espacio"},
		{"espacio y dos puntos", "a b:c", "espacio"},
		{"tilde", "a~b", "«~»"},
		{"circunflejo", "a^b", "«^»"},
		{"dos puntos", "a:b", "«:»"},
		{"interrogación", "a?b", "«?»"},
		{"asterisco", "a*b", "«*»"},
		{"corchete", "a[b", "«[»"},
		{"barra invertida", `a\b`, "barra invertida"},
		{"carácter de control SOH", "a\x01b", "control"},
		{"carácter de control DEL", "a\x7fb", "control"},
		{"tabulador", "a\tb", "control"},
		{"salto de línea", "a\nb", "control"},
		{"barras dobles", "a//b", "«//»"},
		{"empieza por barra", "/a", "«/»"},
		{"termina en barra", "a/", "«/»"},
		{"empieza por punto", ".a", "empezar por «.»"},
		{"componente intermedio que empieza por punto", "a/.b/c", "empezar por «.»"},
		{"termina en .lock", "a.lock", "«.lock»"},
		{"componente intermedio que termina en .lock", "a.lock/b", "«.lock»"},
		{"termina en punto", "a.", "terminar en «.»"},
		{"empieza por guion", "-a", "«-»"},
	})
}

// REQ-25.26 (escenario «Valor inválido») y design §7.1: los casos del diseño
// `"a b:c"` y `"mi prefijo//"`, más la regla propia del prefijo (sin `/` inicial
// ni final) además de todas las de ValidRefName.
func TestValidPrefix(t *testing.T) {
	runRefNameCases(t, ValidPrefix, []refNameCase{
		{"prefijo simple", "feat", ""},
		{"otro prefijo simple", "inc", ""},
		{"prefijo con jerarquía", "equipo/inc", ""},
		{"prefijo con guion", "release-1", ""},

		{"vacío", "", "no puede estar vacío"},
		{"barra inicial", "/inc", "«/»"},
		{"barra final", "inc/", "«/»"},
		{"barras finales dobles", "inc//", "«/»"},
		{"caso del diseño con espacio y barras", "mi prefijo//", "«/»"},
		{"caso del diseño con espacio y dos puntos", "a b:c", "espacio"},
		{"espacio", "mi prefijo", "espacio"},
		{"dos puntos seguidos", "a..b", "«..»"},
		{"barras dobles internas", "a//b", "«//»"},
		{"empieza por punto", ".oculto", "empezar por «.»"},
		{"termina en .lock", "x.lock", "«.lock»"},
		{"termina en punto", "a.", "terminar en «.»"},
		{"empieza por guion", "-x", "«-»"},
		{"carácter prohibido por git", "a:b", "«:»"},
	})
}

// gitref es una hoja pura (design §4.6, B1): solo importa la biblioteca estándar,
// de modo que `workspace`, `kickoff` y `branch` pueden usarla sin formar ciclos.
func TestGitrefImportsOnlyTheStandardLibrary(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("no se pudieron listar los ficheros del paquete: %v (%d)", err, len(files))
	}
	fset := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		for _, spec := range parsed.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatalf("%s: importación ilegible %s", file, spec.Path.Value)
			}
			first, _, _ := strings.Cut(path, "/")
			if strings.Contains(first, ".") {
				t.Errorf("%s importa %q: gitref solo puede importar la biblioteca estándar", file, path)
			}
		}
	}
}
