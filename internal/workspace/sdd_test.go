package workspace

import (
	"errors"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

func sddNode(t *testing.T, source string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(source), &doc); err != nil {
		t.Fatalf("YAML de prueba inválido: %v", err)
	}
	node, ok := mappingValue(doc.Content[0], "sdd")
	if !ok {
		t.Fatalf("el YAML de prueba no tiene bloque sdd")
	}
	return node
}

// RF-ISO-01: el booleano es estricto y la lista de claves, cerrada.
func TestDecodeSDD(t *testing.T) {
	valid := []struct {
		name   string
		source string
		want   bool
	}{
		{"iso true", "sdd:\n  iso: true\n", true},
		{"iso false", "sdd:\n  iso: false\n", false},
		{"mapeo vacío", "sdd: {}\n", false},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeSDD(sddNode(t, tc.source))
			if err != nil || got == nil || got.ISO != tc.want {
				t.Fatalf("decodeSDD = %+v, %v; se esperaba iso=%v", got, err, tc.want)
			}
		})
	}

	invalid := []struct {
		name   string
		source string
		wants  []string
	}{
		{"valor no booleano", "sdd:\n  iso: quizás\n", []string{"sdd.iso", "quizás", "true o false"}},
		{"booleano entre comillas", "sdd:\n  iso: \"true\"\n", []string{"sdd.iso", "sin comillas"}},
		{"nulo", "sdd:\n  iso:\n", []string{"sdd.iso", "nulo"}},
		{"lista", "sdd:\n  iso: [true]\n", []string{"sdd.iso", "lista"}},
		{"mapeo como valor", "sdd:\n  iso: {a: b}\n", []string{"sdd.iso", "mapeo"}},
		{"clave desconocida archive", "sdd:\n  archive: true\n", []string{"archive", "claves admitidas: iso"}},
		{"clave desconocida design_triggers", "sdd:\n  design_triggers: true\n", []string{"design_triggers"}},
		{"clave duplicada", "sdd:\n  iso: true\n  iso: false\n", []string{"clave duplicada 'iso'"}},
		{"bloque nulo", "sdd:\n", []string{"bloque 'sdd'", "mapeo"}},
		{"bloque lista", "sdd:\n  - iso\n", []string{"bloque 'sdd'", "mapeo"}},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeSDD(sddNode(t, tc.source))
			if got != nil {
				t.Errorf("un bloque inválido no devuelve configuración: %+v", got)
			}
			wantErrorContains(t, err, tc.wants...)
		})
	}
}

// RF-ISO-01: solo iso true con lane odd es contradictorio.
func TestCheckISOWorkflow(t *testing.T) {
	lanes := []*WorkflowConfig{nil, {}, {Lane: WorkflowLaneODD}, {Lane: WorkflowLaneSDD}, {Lane: WorkflowLaneAsk}}
	isos := []*SDDConfig{nil, {ISO: false}, {ISO: true}}
	for _, sdd := range isos {
		for _, workflow := range lanes {
			err := checkISOWorkflow(sdd, workflow)
			contradiction := sdd != nil && sdd.ISO && workflow != nil && workflow.Lane == WorkflowLaneODD
			if contradiction != (err != nil) {
				t.Errorf("sdd=%+v workflow=%+v: error=%v, contradicción esperada=%v", sdd, workflow, err, contradiction)
				continue
			}
			if err == nil {
				continue
			}
			if !errors.Is(err, ErrISOWorkflowContradiction) {
				t.Errorf("el error debe envolver ErrISOWorkflowContradiction: %v", err)
			}
			wantErrorContains(t, err, "sdd.iso", "workflow.lane", "retira workflow.lane: odd", "retira sdd.iso")
		}
	}
}

// RF-ISO-01 y RF-ISO-02: carga por capas (parseLayered, ParseConfig, ReadDeclarations
// y Locate) de un bloque `sdd:` válido, inválido y contradictorio.
func TestSDDLoad(t *testing.T) {
	t.Run("sin bloque nada cambia", func(t *testing.T) {
		cfg, baseErr, declarations := parseLayered([]byte(declarationsBase))
		if baseErr != nil || declarations.SDD != nil || declarations.SDDErr != nil {
			t.Fatalf("sin bloque no hay declaración ni error: base=%v %+v", baseErr, declarations)
		}
		wantWorkflow(t, cfg, WorkflowLaneSDD)
		if _, err := ParseConfig([]byte(declarationsBase)); err != nil {
			t.Fatalf("ParseConfig sin bloque: %v", err)
		}
	})

	for name, blocks := range map[string]string{
		"iso true solo":            "sdd:\n  iso: true\n",
		"iso true con lane sdd":    "sdd:\n  iso: true\nworkflow:\n  lane: sdd\n",
		"iso true con lane ask":    "sdd:\n  iso: true\nworkflow:\n  lane: ask\n",
		"iso false con lane odd":   "sdd:\n  iso: false\nworkflow:\n  lane: odd\n",
		"iso true con workflow {}": "sdd:\n  iso: true\nworkflow: {}\n",
	} {
		t.Run("valido/"+name, func(t *testing.T) {
			_, baseErr, declarations := parseDeclarations(t, blocks)
			if baseErr != nil || declarations.SDDErr != nil || declarations.WorkflowErr != nil || declarations.SDD == nil {
				t.Fatalf("base=%v sdd=%v workflow=%v declarado=%+v", baseErr, declarations.SDDErr, declarations.WorkflowErr, declarations.SDD)
			}
			if _, err := ParseConfig([]byte(declarationsBase + blocks)); err != nil {
				t.Fatalf("ParseConfig debe aceptarlo: %v", err)
			}
		})
	}

	t.Run("contradiccion", func(t *testing.T) {
		blocks := "sdd:\n  iso: true\nworkflow:\n  lane: odd\n"
		cfg, baseErr, declarations := parseDeclarations(t, blocks)
		if baseErr != nil {
			t.Fatalf("la base es válida: %v", baseErr)
		}
		if declarations.SDD != nil || !errors.Is(declarations.SDDErr, ErrISOWorkflowContradiction) {
			t.Fatalf("la contradicción va a SDDErr y deja SDD en nil: %+v", declarations)
		}
		if declarations.WorkflowErr != nil || declarations.Workflow == nil || declarations.Workflow.Lane != WorkflowLaneODD {
			t.Errorf("workflow se decodifica igual que siempre: %+v err=%v", declarations.Workflow, declarations.WorkflowErr)
		}
		wantWorkflow(t, cfg, WorkflowLaneODD)
		_, err := ParseConfig([]byte(declarationsBase + blocks))
		wantErrorContains(t, err, "sdd.iso", "workflow.lane")
		if !errors.Is(err, ErrISOWorkflowContradiction) {
			t.Errorf("ParseConfig debe conservar el centinela: %v", err)
		}
	})

	t.Run("sdd invalido no cambia workflow", func(t *testing.T) {
		blocks := "sdd:\n  iso: quizás\nworkflow:\n  lane: odd\n"
		cfg, baseErr, declarations := parseDeclarations(t, blocks)
		if baseErr != nil || declarations.SDD != nil || declarations.SDDErr == nil || errors.Is(declarations.SDDErr, ErrISOWorkflowContradiction) {
			t.Fatalf("base=%v %+v", baseErr, declarations)
		}
		wantWorkflow(t, cfg, WorkflowLaneODD)
		_, err := ParseConfig([]byte(declarationsBase + blocks))
		wantErrorContains(t, err, "sdd.iso", "quizás")
	})

	t.Run("errores de sdd y de otro bloque se acumulan", func(t *testing.T) {
		_, err := ParseConfig([]byte(declarationsBase + "sdd:\n  iso: nope\nworkflow:\n  lane: nope\n"))
		wantErrorContains(t, err, "sdd.iso", "lane")
	})

	t.Run("ReadDeclarations y Locate", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		configPath := filepath.Join(ws.Master, "axiom.yaml")
		workspacetest.WriteFile(t, configPath, locateEmbeddedConfig+"sdd:\n  iso: true\nworkflow:\n  lane: odd\n")
		got := ReadDeclarations(configPath)
		if got.SDD != nil || !errors.Is(got.SDDErr, ErrISOWorkflowContradiction) {
			t.Fatalf("ReadDeclarations: %+v", got)
		}
		loc := mustLocate(t, ws.Master)
		if loc.Config == nil || loc.ConfigErr != nil || !errors.Is(loc.Declarations.SDDErr, ErrISOWorkflowContradiction) {
			t.Fatalf("Locate no se rechaza por la contradicción; queda en SDDErr: %+v", loc)
		}
	})
}

// RF-ISO-01: workspace validate carga con LoadConfig y falla (salida 1) con un
// `sdd:` inválido o contradictorio; con un bloque válido carga sin error.
func TestWorkspaceValidateSDD(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	configPath := filepath.Join(ws.Master, "axiom.yaml")

	workspacetest.WriteFile(t, configPath, locateEmbeddedConfig+"sdd:\n  iso: true\n")
	if _, err := LoadConfig(configPath); err != nil {
		t.Fatalf("un sdd válido debe cargar: %v", err)
	}
	for name, blocks := range map[string]string{
		"contradictorio": "sdd:\n  iso: true\nworkflow:\n  lane: odd\n",
		"inválido":       "sdd:\n  archive: true\n",
	} {
		workspacetest.WriteFile(t, configPath, locateEmbeddedConfig+blocks)
		if _, err := LoadConfig(configPath); err == nil {
			t.Errorf("%s: LoadConfig debe fallar", name)
		}
	}
}

// RF-RID-01: `sdd.id_pattern` es un texto no vacío que compila como plantilla.
func TestDecodeSDDIDPattern(t *testing.T) {
	t.Run("ausente deja el patrón sin declarar", func(t *testing.T) {
		got, err := decodeSDD(sddNode(t, "sdd:\n  iso: true\n"))
		if err != nil || got == nil || got.IDPattern != nil {
			t.Fatalf("decodeSDD = %+v, %v", got, err)
		}
	})
	valid := map[string]string{
		"solo el patrón":       "sdd:\n  id_pattern: \"RF-<DOM>-<NN>\"\n",
		"junto a iso":          "sdd:\n  iso: true\n  id_pattern: \"RF-<DOM>-<NN>\"\n",
		"antes de iso":         "sdd:\n  id_pattern: \"RF-<DOM>-<NN>\"\n  iso: false\n",
		"sin comillas":         "sdd:\n  id_pattern: HU-<NNN>\n",
		"el defecto explícito": "sdd:\n  id_pattern: \"REQ-<n>.<m>\"\n",
	}
	for name, source := range valid {
		t.Run("valido/"+name, func(t *testing.T) {
			got, err := decodeSDD(sddNode(t, source))
			if err != nil || got == nil || got.IDPattern == nil {
				t.Fatalf("decodeSDD = %+v, %v", got, err)
			}
		})
	}
	invalid := []struct {
		name   string
		source string
		wants  []string
	}{
		{"plantilla inválida", "sdd:\n  id_pattern: \"RF-<DOM><NN>\"\n", []string{
			"sdd.id_pattern (línea 2)", `la plantilla "RF-<DOM><NN>" no es válida:`, "seguidos",
			`un ejemplo válido es "RF-<DOM>-<NN>"`, "para el patrón por defecto (REQ-<n>.<m>) retira la clave"}},
		{"lista", "sdd:\n  id_pattern: [RF-<m>]\n", []string{"sdd.id_pattern", "debe ser un texto no vacío"}},
		{"mapa", "sdd:\n  id_pattern: {a: b}\n", []string{"sdd.id_pattern", "debe ser un texto no vacío"}},
		{"nulo", "sdd:\n  id_pattern:\n", []string{"sdd.id_pattern", "debe ser un texto no vacío"}},
		{"cadena vacía", "sdd:\n  id_pattern: \"\"\n", []string{"sdd.id_pattern", "debe ser un texto no vacío"}},
		{"número", "sdd:\n  id_pattern: 12\n", []string{"sdd.id_pattern", "debe ser un texto no vacío"}},
		{"clave repetida", "sdd:\n  id_pattern: RF-<m>\n  id_pattern: HU-<m>\n", []string{"clave duplicada 'id_pattern'"}},
		{"clave desconocida", "sdd:\n  id_patern: RF-<m>\n", []string{"id_patern", "claves admitidas: iso, id_pattern"}},
		{"patrón inválido con iso", "sdd:\n  iso: true\n  id_pattern: nope\n", []string{"sdd.id_pattern (línea 3)", `"nope"`}},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeSDD(sddNode(t, tc.source))
			if got != nil {
				t.Errorf("un bloque inválido no devuelve configuración: %+v", got)
			}
			wantErrorContains(t, err, tc.wants...)
		})
	}
}

// RF-RID-01 y RF-RID-08: IDPattern() lee en vivo la declaración y nunca cae en
// silencio al defecto cuando el bloque o el fichero no se pueden leer.
func TestDeclarationsIDPattern(t *testing.T) {
	template := func(t *testing.T, blocks string) string {
		t.Helper()
		_, _, declarations := parseDeclarations(t, blocks)
		p, err := declarations.IDPattern()
		if err != nil {
			t.Fatalf("IDPattern(%q): %v", blocks, err)
		}
		return p.Template()
	}
	if got := template(t, ""); got != "REQ-<n>.<m>" {
		t.Errorf("sin bloque: %q", got)
	}
	if got := template(t, "sdd:\n  iso: true\n"); got != "REQ-<n>.<m>" {
		t.Errorf("sin clave: %q", got)
	}
	if got := template(t, "sdd:\n  id_pattern: \"RF-<DOM>-<NN>\"\n"); got != "RF-<DOM>-<NN>" {
		t.Errorf("declarado: %q", got)
	}

	_, _, bad := parseDeclarations(t, "sdd:\n  id_pattern: nope\n")
	if _, err := bad.IDPattern(); err == nil || !errors.Is(err, bad.SDDErr) {
		t.Errorf("con SDDErr debe envolverlo con %%w: %v", err)
	}
	contradiction := Declarations{SDDErr: ErrISOWorkflowContradiction}
	if _, err := contradiction.IDPattern(); !errors.Is(err, ErrISOWorkflowContradiction) {
		t.Errorf("con la contradicción debe envolverla: %v", err)
	}
	parseErr := errors.New("yaml roto")
	broken := Declarations{ParseErr: parseErr}
	if _, err := broken.IDPattern(); !errors.Is(err, parseErr) {
		t.Errorf("con ParseErr debe envolverlo: %v", err)
	}
}

// RF-RID-01: workspace validate (LoadConfig) falla con un patrón inválido.
func TestWorkspaceValidateIDPattern(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	configPath := filepath.Join(ws.Master, "axiom.yaml")

	workspacetest.WriteFile(t, configPath, locateEmbeddedConfig+"sdd:\n  id_pattern: \"RF-<DOM>-<NN>\"\n")
	if _, err := LoadConfig(configPath); err != nil {
		t.Fatalf("un patrón válido debe cargar: %v", err)
	}
	workspacetest.WriteFile(t, configPath, locateEmbeddedConfig+"sdd:\n  id_pattern: \"<m><m>\"\n")
	_, err := LoadConfig(configPath)
	wantErrorContains(t, err, "sdd.id_pattern", `"<m><m>"`)
}

// `sdd.explainer_html` es un texto de la lista cerrada ask, always o never.
func TestDecodeSDDExplainerHTML(t *testing.T) {
	valid := map[string]ExplainerHTMLMode{
		"ask":           ExplainerHTMLAsk,
		"always":        ExplainerHTMLAlways,
		"never":         ExplainerHTMLNever,
		"\"always\"":    ExplainerHTMLAlways,
		"always # nota": ExplainerHTMLAlways,
	}
	for value, want := range valid {
		got, err := decodeSDD(sddNode(t, "sdd:\n  explainer_html: "+value+"\n"))
		if err != nil || got == nil || got.ExplainerHTML != want {
			t.Errorf("explainer_html: %s -> %+v, %v", value, got, err)
		}
	}

	t.Run("independiente de iso y de id_pattern", func(t *testing.T) {
		got, err := decodeSDD(sddNode(t, "sdd:\n  iso: true\n  explainer_html: never\n  id_pattern: \"RF-<DOM>-<NN>\"\n"))
		if err != nil || got == nil || !got.ISO || got.IDPattern == nil || got.ExplainerHTML != ExplainerHTMLNever {
			t.Fatalf("decodeSDD = %+v, %v", got, err)
		}
	})
	t.Run("ausente deja el modo sin declarar", func(t *testing.T) {
		got, err := decodeSDD(sddNode(t, "sdd:\n  iso: true\n"))
		if err != nil || got == nil || got.ExplainerHTML != "" {
			t.Fatalf("decodeSDD = %+v, %v", got, err)
		}
	})

	invalid := []struct {
		name   string
		source string
		wants  []string
	}{
		{"texto fuera de la lista", "sdd:\n  explainer_html: quizás\n", []string{"sdd.explainer_html (línea 2)", `"quizás"`, "usa ask, always o never"}},
		{"mayúsculas", "sdd:\n  explainer_html: Always\n", []string{`"Always"`, "usa ask, always o never"}},
		{"booleano", "sdd:\n  explainer_html: true\n", []string{`"true"`, "usa ask, always o never"}},
		{"nulo", "sdd:\n  explainer_html:\n", []string{"sdd.explainer_html", "usa ask, always o never"}},
		{"lista", "sdd:\n  explainer_html: [ask]\n", []string{"sdd.explainer_html", "usa ask, always o never"}},
		{"mapa", "sdd:\n  explainer_html: {a: b}\n", []string{"sdd.explainer_html", "usa ask, always o never"}},
		{"clave repetida", "sdd:\n  explainer_html: ask\n  explainer_html: never\n", []string{"clave duplicada 'explainer_html'"}},
		{"clave desconocida", "sdd:\n  explainer_htm: ask\n", []string{"explainer_htm", "claves admitidas: iso, id_pattern, explainer_html"}},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeSDD(sddNode(t, tc.source))
			if got != nil {
				t.Errorf("un bloque inválido no devuelve configuración: %+v", got)
			}
			wantErrorContains(t, err, tc.wants...)
		})
	}
}

// Declarations.ExplainerHTML() lee en vivo la declaración y nunca cae en silencio
// a ask con un bloque o un fichero que no se pudo leer.
func TestDeclarationsExplainerHTML(t *testing.T) {
	mode := func(t *testing.T, blocks string) ExplainerHTMLMode {
		t.Helper()
		_, _, declarations := parseDeclarations(t, blocks)
		got, err := declarations.ExplainerHTML()
		if err != nil {
			t.Fatalf("ExplainerHTML(%q): %v", blocks, err)
		}
		return got
	}
	if got := mode(t, ""); got != ExplainerHTMLAsk {
		t.Errorf("sin bloque: %q", got)
	}
	if got := mode(t, "sdd:\n  iso: true\n"); got != ExplainerHTMLAsk {
		t.Errorf("sin clave: %q", got)
	}
	if got := mode(t, "sdd:\n  explainer_html: always\n"); got != ExplainerHTMLAlways {
		t.Errorf("always: %q", got)
	}
	// Independiente del carril: workflow.lane no cambia el valor.
	if got := mode(t, "workflow:\n  lane: odd\nsdd:\n  explainer_html: never\n"); got != ExplainerHTMLNever {
		t.Errorf("never con workflow.lane odd: %q", got)
	}

	_, _, bad := parseDeclarations(t, "sdd:\n  explainer_html: quizás\n")
	if _, err := bad.ExplainerHTML(); err == nil || !errors.Is(err, bad.SDDErr) {
		t.Errorf("con SDDErr debe envolverlo con %%w: %v", err)
	}
	contradiction := Declarations{SDDErr: ErrISOWorkflowContradiction}
	if _, err := contradiction.ExplainerHTML(); !errors.Is(err, ErrISOWorkflowContradiction) {
		t.Errorf("con la contradicción debe envolverla: %v", err)
	}
	parseErr := errors.New("yaml roto")
	if _, err := (Declarations{ParseErr: parseErr}).ExplainerHTML(); !errors.Is(err, parseErr) {
		t.Errorf("con ParseErr debe envolverlo: %v", err)
	}
}

// workspace validate (LoadConfig) rechaza un valor no admitido y acepta los tres.
func TestWorkspaceValidateExplainerHTML(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	configPath := filepath.Join(ws.Master, "axiom.yaml")

	for _, v := range []string{"ask", "always", "never"} {
		workspacetest.WriteFile(t, configPath, locateEmbeddedConfig+"sdd:\n  explainer_html: "+v+"\n")
		if _, err := LoadConfig(configPath); err != nil {
			t.Fatalf("%s debe cargar: %v", v, err)
		}
	}
	workspacetest.WriteFile(t, configPath, locateEmbeddedConfig+"sdd:\n  explainer_html: quizás\n")
	_, err := LoadConfig(configPath)
	wantErrorContains(t, err, "sdd.explainer_html", `"quizás"`, "usa ask, always o never")
}
