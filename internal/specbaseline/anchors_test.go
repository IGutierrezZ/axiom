package specbaseline

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseAnchor(t *testing.T) {
	tests := []struct {
		name string
		text string
		want Anchor
		fail string
	}{
		{name: "solo título", text: "Servicio de pagos", want: Anchor{Path: []string{"Servicio de pagos"}}},
		{name: "padre y título", text: "Componentes > Servicio de pagos", want: Anchor{Path: []string{"Componentes", "Servicio de pagos"}}},
		{name: "abuelo, padre y título", text: "Arquitectura > Componentes > Servicio", want: Anchor{Path: []string{"Arquitectura", "Componentes", "Servicio"}}},
		{name: "ordinal en el título", text: "Notas #2", want: Anchor{Path: []string{"Notas"}, Ordinal: 2}},
		{name: "ordinal en el último elemento de una ruta", text: "Pagos > Notas #3", want: Anchor{Path: []string{"Pagos", "Notas"}, Ordinal: 3}},
		{name: "espacios alrededor", text: "  Pagos   >   Notas  ", want: Anchor{Path: []string{"Pagos", "Notas"}}},
		{name: "un número sin almohadilla es parte del título", text: "Fase 2", want: Anchor{Path: []string{"Fase 2"}}},
		{name: "una almohadilla sin número es parte del título", text: "Notas #", want: Anchor{Path: []string{"Notas #"}}},
		{name: "un ordinal con letras es parte del título", text: "Notas #x", want: Anchor{Path: []string{"Notas #x"}}},
		{name: "ordinal en un elemento intermedio es parte de su título", text: "Fase #2 > Notas", want: Anchor{Path: []string{"Fase #2", "Notas"}}},
		{name: "ancla vacía", text: "   ", fail: "el ancla está vacía"},
		{name: "elemento vacío", text: "Pagos >  > Notas", fail: "elemento vacío"},
		{name: "ordinal cero", text: "Notas #0", fail: "mayor que cero"},
		{name: "ordinal que no cabe en un entero", text: "Notas #99999999999999999999", fail: "mayor que cero"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAnchor(tt.text)
			if tt.fail != "" {
				if err == nil || !strings.Contains(err.Error(), tt.fail) {
					t.Fatalf("ParseAnchor(%q) error = %v, quería que contuviera %q", tt.text, err, tt.fail)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAnchor(%q): %v", tt.text, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseAnchor(%q) = %#v, quería %#v", tt.text, got, tt.want)
			}
		})
	}
}

func TestAnchorStringRoundTrips(t *testing.T) {
	for _, text := range []string{"Título", "Padre > Título", "Abuelo > Padre > Título #2"} {
		anchor, err := ParseAnchor(text)
		if err != nil {
			t.Fatalf("ParseAnchor(%q): %v", text, err)
		}
		if got := anchor.String(); got != text {
			t.Errorf("Anchor.String() = %q, quería %q", got, text)
		}
	}
}

const anchorsDocument = `# Arquitectura

## Pagos

### Notas

Texto de pagos.

### Servicio de pagos

#### Detalle

## Envíos

### Notas

Texto de envíos.

### Notas

Otra sección con el mismo título bajo el mismo padre.

# Anexo

## Notas
`

func headingPaths(refs []HeadingRef) []string {
	paths := make([]string, len(refs))
	for i, ref := range refs {
		paths[i] = ref.FullPath()
	}
	return paths
}

func TestResolveAnchor(t *testing.T) {
	tests := []struct {
		name   string
		anchor string
		want   []string // rutas completas, en orden del documento
		lines  []int
	}{
		{name: "título único", anchor: "Servicio de pagos", want: []string{"Arquitectura > Pagos > Servicio de pagos"}, lines: []int{9}},
		{name: "título repetido sin ruta", anchor: "Notas", want: []string{
			"Arquitectura > Pagos > Notas", "Arquitectura > Envíos > Notas", "Arquitectura > Envíos > Notas", "Anexo > Notas",
		}, lines: []int{5, 15, 19, 25}},
		{name: "el padre desambigua", anchor: "Pagos > Notas", want: []string{"Arquitectura > Pagos > Notas"}, lines: []int{5}},
		{name: "el abuelo desambigua", anchor: "Arquitectura > Envíos > Notas", want: []string{"Arquitectura > Envíos > Notas", "Arquitectura > Envíos > Notas"}, lines: []int{15, 19}},
		{name: "el ordinal elige la segunda coincidencia", anchor: "Envíos > Notas #2", want: []string{"Arquitectura > Envíos > Notas"}, lines: []int{19}},
		{name: "el ordinal cuenta en orden de documento", anchor: "Notas #4", want: []string{"Anexo > Notas"}, lines: []int{25}},
		{name: "un ordinal fuera de rango no resuelve", anchor: "Envíos > Notas #3", want: nil},
		{name: "la cadena de ancestros debe ser la inmediata", anchor: "Arquitectura > Notas", want: nil},
		{name: "un ancestro que no es el padre no resuelve", anchor: "Anexo > Pagos", want: nil},
		{name: "un título que no existe", anchor: "Inexistente", want: nil},
		{name: "distingue mayúsculas", anchor: "notas", want: nil},
		{name: "encabezado de nivel mayor bajo su padre inmediato", anchor: "Servicio de pagos > Detalle", want: []string{"Arquitectura > Pagos > Servicio de pagos > Detalle"}, lines: []int{11}},
	}
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		doc := v.apply(anchorsDocument)
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				anchor, err := ParseAnchor(tt.anchor)
				if err != nil {
					t.Fatal(err)
				}
				got := ResolveAnchor(doc, anchor)
				if !reflect.DeepEqual(headingPaths(got), tt.want) && !(len(got) == 0 && len(tt.want) == 0) {
					t.Fatalf("ResolveAnchor(%q) = %v, quería %v", tt.anchor, headingPaths(got), tt.want)
				}
				for i, ref := range got {
					if ref.Line != tt.lines[i] {
						t.Errorf("coincidencia %d en la línea %d, quería %d", i, ref.Line, tt.lines[i])
					}
				}
			})
		}
	})
}

func TestResolveAnchorIgnoresFencesAndSetext(t *testing.T) {
	doc := []byte("# Guía\n\n```\n## Pagos\n```\n\nPagos\n=====\n\n## Pagos\n")
	anchor, _ := ParseAnchor("Pagos")
	got := ResolveAnchor(doc, anchor)
	if len(got) != 1 || got[0].Line != 10 {
		t.Fatalf("ResolveAnchor = %v, quería solo el encabezado ATX de la línea 10", got)
	}
}

func TestResolveAnchorSkipsHeadingLevels(t *testing.T) {
	doc := []byte("# Raíz\n\n### Profundo\n\n## Medio\n\n### Hijo\n")
	deep, _ := ParseAnchor("Raíz > Profundo")
	if got := ResolveAnchor(doc, deep); len(got) != 1 {
		t.Fatalf("un nivel saltado sigue siendo hijo del encabezado de nivel menor anterior: %v", headingPaths(got))
	}
	child, _ := ParseAnchor("Medio > Hijo")
	if got := ResolveAnchor(doc, child); len(got) != 1 {
		t.Fatalf("Medio > Hijo debe resolver: %v", headingPaths(got))
	}
	cross, _ := ParseAnchor("Profundo > Hijo")
	if got := ResolveAnchor(doc, cross); len(got) != 0 {
		t.Fatalf("Hijo cuelga de Medio y no de Profundo: %v", headingPaths(got))
	}
}

func contextEntry(action ContextAction, anchor string) ContextEntry {
	parsed, err := ParseAnchor(anchor)
	if err != nil {
		panic(err)
	}
	return ContextEntry{ID: "C1", Document: "context/ARQUITECTURA.md", Anchor: parsed, Action: action, Line: 7}
}

func TestCheckAnchorMissingDocument(t *testing.T) {
	create, err := CheckAnchor(contextEntry(ContextAdd, "Pagos"), nil, false)
	if err != nil || !create {
		t.Fatalf("add sobre un documento inexistente debe crearlo: create=%v err=%v", create, err)
	}
	for _, action := range []ContextAction{ContextModify, ContextRemove} {
		_, err := CheckAnchor(contextEntry(action, "Pagos"), nil, false)
		if err == nil || !strings.Contains(err.Error(), "no existe") || !strings.Contains(err.Error(), "solo la acción 'add'") {
			t.Errorf("%s sobre un documento inexistente debe rechazarse nombrando 'add': %v", action, err)
		}
	}
}

func TestCheckAnchorAgainstTheDocument(t *testing.T) {
	doc := []byte(anchorsDocument)
	tests := []struct {
		name   string
		action ContextAction
		anchor string
		fail   []string // fragmentos del rechazo; vacío si es válida
	}{
		{name: "modify con una coincidencia", action: ContextModify, anchor: "Pagos > Notas"},
		{name: "remove con una coincidencia", action: ContextRemove, anchor: "Servicio de pagos"},
		{name: "add sobre una sección existente", action: ContextAdd, anchor: "Servicio de pagos"},
		{name: "add sin coincidencia", action: ContextAdd, anchor: "Pagos > Reintentos"},
		{name: "modify sin coincidencia", action: ContextModify, anchor: "Pagos > Reintentos", fail: []string{"no resuelve a ninguna sección", "solo 'add' admite un ancla inexistente"}},
		{name: "remove sin coincidencia", action: ContextRemove, anchor: "Inexistente", fail: []string{"no resuelve a ninguna sección"}},
		{name: "ancla ambigua con modify", action: ContextModify, anchor: "Envíos > Notas", fail: []string{"es ambigua", "línea 15", "línea 19", "Arquitectura > Envíos > Notas"}},
		{name: "ancla ambigua con add", action: ContextAdd, anchor: "Notas", fail: []string{"resuelve a 4 secciones", "línea 5", "línea 25", "Anexo > Notas"}},
		{name: "el ordinal desambigua", action: ContextModify, anchor: "Envíos > Notas #2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			create, err := CheckAnchor(contextEntry(tt.action, tt.anchor), doc, true)
			if create {
				t.Error("un documento existente no se crea")
			}
			if len(tt.fail) == 0 {
				if err != nil {
					t.Fatalf("debía ser válida: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("debía rechazarse")
			}
			for _, fragment := range tt.fail {
				if !strings.Contains(err.Error(), fragment) {
					t.Errorf("el rechazo %q debe contener %q", err, fragment)
				}
			}
			if !strings.Contains(err.Error(), "C1") || !strings.Contains(err.Error(), "context/ARQUITECTURA.md") {
				t.Errorf("el rechazo debe nombrar la entrada y el documento: %v", err)
			}
		})
	}
}

// R33: lo que el formato no puede anclar se rechaza nombrando las apariciones,
// también con la acción add, que de otro modo aceptaría un ancla sin coincidencia.
func TestCheckAnchorNamesTheOccurrencesItCannotAddress(t *testing.T) {
	doc := []byte("# Guía\n\n## Entrada > Salida\n\nTexto.\n\nReintentos\n----------\n\nOtro texto.\n\nNotas del servicio\n==================\n")
	tests := []struct {
		name   string
		action ContextAction
		anchor string
		want   string
	}{
		{name: "título con separador y add", action: ContextAdd, anchor: "Entrada > Salida", want: `línea 3 "Entrada > Salida": su título contiene ">"`},
		{name: "título con separador y modify", action: ContextModify, anchor: "Entrada > Salida", want: "R33"},
		{name: "setext con subrayado de guiones y add", action: ContextAdd, anchor: "Reintentos", want: `línea 7 "Reintentos": es un encabezado setext`},
		{name: "setext con subrayado de iguales y modify", action: ContextModify, anchor: "Guía > Notas del servicio", want: `línea 12 "Notas del servicio"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CheckAnchor(contextEntry(tt.action, tt.anchor), doc, true)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("el rechazo debe contener %q: %v", tt.want, err)
			}
		})
	}
}

func TestCheckAnchorDoesNotMistakeSeparatorsForSetext(t *testing.T) {
	// Una línea `---` tras un encabezado ATX o tras una línea en blanco es una
	// regla horizontal, no un subrayado setext.
	doc := []byte("# Guía\n\n---\n\n## Pagos\n---\n\nTexto\n")
	anchor, _ := ParseAnchor("Reintentos")
	if hidden := unaddressable(doc, anchor); len(hidden) != 0 {
		t.Fatalf("no hay encabezados inanclables: %v", hidden)
	}
	if lines := setextHeadings(doc); len(lines) != 0 {
		t.Fatalf("no hay encabezados setext: %v", lines)
	}
}
