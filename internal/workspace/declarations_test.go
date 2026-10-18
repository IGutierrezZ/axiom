package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// declarationsBase es una configuración base válida a la que cada prueba añade
// los bloques `baseline:` y `technical_context:`.
const declarationsBase = `workspace:
  name: ejemplo
  topology: monorepo-embedded
  specs_repository: .
roles:
  core:
    name: Core
    repositories:
      - path: .
`

// parseDeclarations carga declarationsBase más el bloque dado con parseLayered.
func parseDeclarations(t *testing.T, blocks string) (*WorkspaceConfig, error, Declarations) {
	t.Helper()
	return parseLayered([]byte(declarationsBase + blocks))
}

// wantErrorContains comprueba que err existe y que su texto contiene cada
// fragmento; los errores de las declaraciones se acumulan, así que una misma
// violación puede convivir con otras.
func wantErrorContains(t *testing.T, err error, wants ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("se esperaba un error que contuviera %q, no hubo ninguno", wants)
	}
	for _, want := range wants {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no contiene %q:\n%v", want, err)
		}
	}
}

func TestDeclarationsAbsentAreTheZeroValue(t *testing.T) {
	cfg, baseErr, declarations := parseLayered([]byte(declarationsBase))
	if baseErr != nil {
		t.Fatalf("baseErr = %v, se esperaba nil", baseErr)
	}
	if declarations.ParseErr != nil || declarations.Baseline != nil || declarations.BaselineErr != nil ||
		declarations.TechnicalContext != nil || declarations.TechnicalContextErr != nil {
		t.Fatalf("sin bloques nuevos las declaraciones deben ser el valor cero, no %+v", declarations)
	}
	if cfg == nil || cfg.Baseline != nil || cfg.TechnicalContext != nil {
		t.Fatalf("la configuración base no debe llevar bloques nuevos: %+v", cfg)
	}
}

// REQ-24.16: un documento de prosa integra con directed por defecto o con
// register si lo declara; un documento con perfil integra por composición.
func TestDeclarationsValidBlocksAreLoaded(t *testing.T) {
	cfg, baseErr, declarations := parseDeclarations(t, `baseline:
  documents:
    - domain: requisitos-funcionales
      path: 01_Requisitos_Funcionales.md
      requirement:
        heading: '^### (?P<id>RF-\d+): (?P<title>.+)$'
        level: 3
        identity: id
        render: '### {id}: {title}'
    - domain: modelo-de-datos
      path: docs\03_Modelo_Datos.md
    - domain: flujos-de-negocio
      path: 04_Flujos_Negocio.md
      integration: register
technical_context:
  maintained:
    entry: context/TECHNICAL_CONTEXT.md
    paths: [context/]
  reference:
    paths: [00_Glosario.md]
governance:
  context: docs/contexto.md
`)
	if baseErr != nil || declarations.BaselineErr != nil || declarations.TechnicalContextErr != nil {
		t.Fatalf("la declaración válida no debe fallar: base=%v baseline=%v contexto=%v", baseErr, declarations.BaselineErr, declarations.TechnicalContextErr)
	}
	if cfg.Baseline == nil || cfg.TechnicalContext == nil {
		t.Fatalf("la configuración debe incluir ambos bloques poblados: %+v", cfg)
	}
	if cfg.Governance.Context != "docs/contexto.md" {
		t.Errorf("governance.context debe convivir con la declaración sin interpretarse: %q", cfg.Governance.Context)
	}
	documents := cfg.Baseline.Documents
	if len(documents) != 3 {
		t.Fatalf("documentos = %d, se esperaban 3", len(documents))
	}

	profile := documents[0]
	if profile.EffectiveIntegration() != IntegrationCompose {
		t.Errorf("un documento con perfil integra por composición, no %q", profile.EffectiveIntegration())
	}
	if profile.EffectiveScenarios() != ScenariosInclude {
		t.Errorf("scenarios por defecto = %q, se esperaba include", profile.EffectiveScenarios())
	}
	if profile.Requirement.EffectiveIdentity() != IdentityID {
		t.Errorf("identity = %q, se esperaba id", profile.Requirement.EffectiveIdentity())
	}
	re, err := profile.Requirement.Compile()
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !re.MatchString("### RF-001: Alta de clientes") {
		t.Error("el patrón compilado debe reconocer un encabezado de requisito")
	}
	if documents[1].EffectiveIntegration() != IntegrationDirected {
		t.Errorf("un documento de prosa integra con directed por defecto, no %q", documents[1].EffectiveIntegration())
	}
	if documents[2].EffectiveIntegration() != IntegrationRegister {
		t.Errorf("integration declarado register, obtenido %q", documents[2].EffectiveIntegration())
	}
	if got := cfg.TechnicalContext.Maintained; got == nil || got.Entry != "context/TECHNICAL_CONTEXT.md" || len(got.Paths) != 1 {
		t.Errorf("maintained mal cargado: %+v", got)
	}
	if got := cfg.TechnicalContext.Reference; got == nil || got.Entry != "" || len(got.Paths) != 1 {
		t.Errorf("reference mal cargado: %+v", got)
	}
}

func TestDeclarationsEffectiveValuesOfAnEmptyProfile(t *testing.T) {
	if got := (BaselineDocument{}).EffectiveScenarios(); got != ScenariosInclude {
		t.Errorf("EffectiveScenarios() = %q, se esperaba include", got)
	}
	if got := (BaselineDocument{Scenarios: ScenariosOmit}).EffectiveScenarios(); got != ScenariosOmit {
		t.Errorf("EffectiveScenarios() = %q, se esperaba omit", got)
	}
	if got := (BaselineDocument{Integration: IntegrationRegister}).EffectiveIntegration(); got != IntegrationRegister {
		t.Errorf("EffectiveIntegration() = %q, se esperaba register", got)
	}
	if got := (RequirementProfile{Identity: IdentityText}).EffectiveIdentity(); got != IdentityText {
		t.Errorf("EffectiveIdentity() = %q, se esperaba text", got)
	}
}

func TestDeclarationsRequirementProfileCompile(t *testing.T) {
	tests := []struct {
		name    string
		heading string
		want    []string
	}{
		{"valido", `^### (?P<id>RF-\d+): (?P<title>.+)$`, nil},
		{"grupos sin nombre", `^### (RF-\d+): (.+)$`, []string{"los grupos con nombre 'id' y 'title'"}},
		{"solo el grupo id", `^### (?P<id>RF-\d+)$`, []string{"los grupos con nombre 'id' y 'title'"}},
		{"parentesis sin cerrar", `^### (?P<id>RF-\d+: (?P<title>.+)$`, []string{"no compila"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := RequirementProfile{Heading: tc.heading}.Compile()
			if tc.want == nil {
				if err != nil {
					t.Fatalf("Compile: %v", err)
				}
				return
			}
			wantErrorContains(t, err, tc.want...)
		})
	}
}

// baselineDoc compone un bloque baseline con un único documento de prosa.
func baselineDoc(entries ...string) string {
	return "baseline:\n  documents:\n" + strings.Join(entries, "")
}

func proseDoc(domain, path string) string {
	return "    - domain: " + domain + "\n      path: " + path + "\n"
}

// REQ-24.13 a REQ-24.16 y REQ-24.60: tablas de validación sintáctica de baseline.
// Los casos de integration cubren REQ-24.16: solo 'directed' o 'register', y
// nunca en un documento con perfil de requisito.
func TestDeclarationsBaselineValidation(t *testing.T) {
	profile := func(domain, extra string) string {
		return "    - domain: " + domain + "\n      path: " + domain + ".md\n      requirement:\n" +
			"        heading: '^### (?P<id>RF-\\d+): (?P<title>.+)$'\n        level: 3\n        render: '### {id}: {title}'\n" + extra
	}
	tests := []struct {
		name    string
		blocks  string
		valid   bool
		wantErr []string
	}{
		{"documentos validos", baselineDoc(proseDoc("a", "a.md"), proseDoc("b-2", "dir/b.md")), true, nil},
		{"baseline vacia", "baseline: {}\n", false, []string{"requiere al menos un documento"}},
		{"baseline nula", "baseline:\n", false, []string{"requiere al menos un documento"}},
		{"documents vacio", "baseline:\n  documents: []\n", false, []string{"requiere al menos un documento"}},
		{"documents no es una lista", "baseline:\n  documents: x\n", false, []string{"'baseline.documents' debe ser una lista"}},
		{"baseline no es un mapeo", "baseline: [a]\n", false, []string{"debe ser un mapeo"}},
		{"clave desconocida en baseline", "baseline:\n  documentss: []\n", false, []string{"clave desconocida 'documentss'", "línea 11", "claves admitidas: documents"}},
		{"clave desconocida en una entrada", baselineDoc("    - domain: a\n      path: a.md\n      requierment: {}\n"), false, []string{"clave desconocida 'requierment'", "baseline.documents[0]", "domain, path, requirement, integration, scenarios"}},
		{"clave desconocida en el perfil", baselineDoc(profile("a", "        extra: 1\n")), false, []string{"clave desconocida 'extra'", "baseline.documents[0].requirement", "heading, level, identity, render"}},
		{"dominio duplicado", baselineDoc(proseDoc("a", "a.md"), proseDoc("a", "b.md")), false, []string{"el dominio 'a' está duplicado", "entradas 0 y 1"}},
		{"dominio con mayusculas y espacios", baselineDoc(proseDoc("'Requisitos Funcionales'", "a.md")), false, []string{"el dominio 'Requisitos Funcionales'", "minúsculas, dígitos y guiones"}},
		{"dominio con guion al final", baselineDoc(proseDoc("a-", "a.md")), false, []string{"el dominio 'a-'"}},
		{"dominio ausente", baselineDoc("    - path: a.md\n"), false, []string{"no declara 'domain'"}},
		{"documento en dos dominios", baselineDoc(proseDoc("a", "doc.md"), proseDoc("b", "./doc.md")), false, []string{"la ruta './doc.md' se declara en los dominios 'a' y 'b'"}},
		{"documento en dos dominios sin distinguir mayusculas", baselineDoc(proseDoc("a", "Doc.md"), proseDoc("b", "doc.md")), false, []string{"los dominios 'a' y 'b'"}},
		{"ruta que sale del repositorio", baselineDoc(proseDoc("a", "../fuera.md")), false, []string{"el dominio 'a'", "'../fuera.md' sale de workspaceRoot"}},
		{"ruta que sale tras limpiar", baselineDoc(proseDoc("a", "x/../../fuera.md")), false, []string{"sale de workspaceRoot"}},
		{"ruta absoluta posix", baselineDoc(proseDoc("a", "/etc/doc.md")), false, []string{"debe ser relativa a workspaceRoot", "sin unidad de disco ni barra inicial"}},
		{"ruta con unidad de disco", baselineDoc(proseDoc("a", `'C:\docs\a.md'`)), false, []string{"debe ser relativa a workspaceRoot"}},
		{"ruta con barra invertida inicial", baselineDoc(proseDoc("a", `'\docs\a.md'`)), false, []string{"debe ser relativa a workspaceRoot"}},
		{"ruta con separador de Windows valida", baselineDoc(proseDoc("a", `'docs\a.md'`)), true, nil},
		{"ruta vacia", baselineDoc("    - domain: a\n      path: ''\n"), false, []string{"la ruta está vacía"}},
		{"ruta es workspaceRoot", baselineDoc(proseDoc("a", ".")), false, []string{"debe apuntar a un fichero"}},
		{"perfil valido", baselineDoc(profile("a", "")), true, nil},
		{"identidad text valida", baselineDoc(profile("a", "        identity: text\n")), true, nil},
		{"perfil sin grupos con nombre", baselineDoc("    - domain: a\n      path: a.md\n      requirement:\n        heading: '^### (RF-\\d+): (.+)$'\n        level: 3\n        render: '### {id}: {title}'\n"), false, []string{"el dominio 'a'", "los grupos con nombre 'id' y 'title'"}},
		{"perfil con patron que no compila", baselineDoc("    - domain: a\n      path: a.md\n      requirement:\n        heading: '^### (?P<id>RF-\\d+: (?P<title>.+)$'\n        level: 3\n        render: '### {id}: {title}'\n"), false, []string{"el dominio 'a'", "no compila"}},
		{"perfil sin heading", baselineDoc("    - domain: a\n      path: a.md\n      requirement:\n        level: 3\n        render: '### {id}: {title}'\n"), false, []string{"'requirement.heading' es obligatorio"}},
		{"nivel cero", baselineDoc("    - domain: a\n      path: a.md\n      requirement:\n        heading: '^(?P<id>x) (?P<title>y)$'\n        render: '# {id} {title}'\n"), false, []string{"entero entre 1 y 6, no 0"}},
		{"nivel siete", baselineDoc("    - domain: a\n      path: a.md\n      scenarios: omit\n      requirement:\n        heading: '^(?P<id>x) (?P<title>y)$'\n        level: 7\n        render: '####### {id} {title}'\n"), false, []string{"entero entre 1 y 6, no 7"}},
		{"identidad desconocida", baselineDoc(profile("a", "        identity: nombre\n")), false, []string{"el dominio 'a'", "'id' o 'text', no 'nombre'"}},
		{"render incoherente con el nivel", baselineDoc("    - domain: a\n      path: a.md\n      requirement:\n        heading: '^(?P<id>x) (?P<title>y)$'\n        level: 3\n        render: '## {id}: {title}'\n"), false, []string{"debe comenzar con exactamente 3 signos '#'"}},
		{"render con demasiados signos", baselineDoc("    - domain: a\n      path: a.md\n      requirement:\n        heading: '^(?P<id>x) (?P<title>y)$'\n        level: 3\n        render: '#### {id}: {title}'\n"), false, []string{"debe comenzar con exactamente 3 signos '#'"}},
		{"render sin marcadores", baselineDoc("    - domain: a\n      path: a.md\n      requirement:\n        heading: '^(?P<id>x) (?P<title>y)$'\n        level: 3\n        render: '### {id}'\n"), false, []string{"los marcadores {id} y {title}"}},
		{"render ausente", baselineDoc("    - domain: a\n      path: a.md\n      requirement:\n        heading: '^(?P<id>x) (?P<title>y)$'\n        level: 3\n"), false, []string{"'requirement.render' es obligatorio"}},
		{"requirement nulo", baselineDoc("    - domain: a\n      path: a.md\n      requirement:\n"), false, []string{"'requirement' debe ser un mapeo"}},
		{"integracion register en prosa", baselineDoc("    - domain: a\n      path: a.md\n      integration: register\n"), true, nil},
		{"integracion desconocida", baselineDoc("    - domain: a\n      path: a.md\n      integration: auto\n"), false, []string{"el dominio 'a'", "'directed' o 'register', no 'auto'"}},
		{"integracion compose declarada", baselineDoc("    - domain: a\n      path: a.md\n      integration: compose\n"), false, []string{"'directed' o 'register', no 'compose'"}},
		{"integracion en un documento con perfil", baselineDoc(profile("a", "      integration: directed\n")), false, []string{"no se admite en un documento con perfil de requisito"}},
		{"scenarios omit en perfil", baselineDoc(profile("a", "      scenarios: omit\n")), true, nil},
		{"scenarios invalido", baselineDoc(profile("a", "      scenarios: ninguno\n")), false, []string{"el dominio 'a'", "'include' u 'omit', no 'ninguno'"}},
		{"scenarios en prosa", baselineDoc("    - domain: a\n      path: a.md\n      scenarios: omit\n"), false, []string{"el dominio 'a'", "solo se admite en documentos con perfil de requisito"}},
		{"nivel 3 con include por defecto es valido", baselineDoc(profile("a", "")), true, nil},
		{"include por defecto con nivel 4", baselineDoc("    - domain: a\n      path: a.md\n      requirement:\n        heading: '^#### (?P<id>x) (?P<title>y)$'\n        level: 4\n        render: '#### {id}: {title}'\n"), false, []string{"el dominio 'a'", "'requirement.level' es 4", "'#### Scenario:'", "'scenarios: omit' o un 'level' menor o igual que 3"}},
		{"include explicito con nivel 5", baselineDoc("    - domain: a\n      path: a.md\n      scenarios: include\n      requirement:\n        heading: '^##### (?P<id>x) (?P<title>y)$'\n        level: 5\n        render: '##### {id}: {title}'\n"), false, []string{"el dominio 'a'", "'requirement.level' es 5", "'scenarios: omit' o un 'level' menor o igual que 3"}},
		{"omit admite nivel 4", baselineDoc("    - domain: a\n      path: a.md\n      scenarios: omit\n      requirement:\n        heading: '^#### (?P<id>x) (?P<title>y)$'\n        level: 4\n        render: '#### {id}: {title}'\n"), true, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, baseErr, declarations := parseDeclarations(t, tc.blocks)
			if baseErr != nil {
				t.Fatalf("la configuración base es válida y no debe fallar: %v", baseErr)
			}
			if tc.valid {
				if declarations.BaselineErr != nil {
					t.Fatalf("la declaración debe ser válida: %v", declarations.BaselineErr)
				}
				if declarations.Baseline == nil || cfg.Baseline == nil {
					t.Fatal("una declaración válida debe poblar Declarations.Baseline y WorkspaceConfig.Baseline")
				}
				return
			}
			wantErrorContains(t, declarations.BaselineErr, tc.wantErr...)
			if declarations.Baseline != nil || cfg.Baseline != nil {
				t.Error("un bloque inválido no debe poblar la baseline ni parcial ni por defecto")
			}
		})
	}
}

// REQ-24.17: todas las violaciones se acumulan con errors.Join, una por ruta o dominio.
func TestDeclarationsErrorsAccumulate(t *testing.T) {
	_, baseErr, declarations := parseDeclarations(t, baselineDoc(
		proseDoc("a", "../uno.md"),
		proseDoc("a", "/dos.md"),
		proseDoc("c", "c.md"),
	)+"technical_context:\n  maintained:\n    paths: []\n")
	if baseErr != nil {
		t.Fatalf("baseErr = %v", baseErr)
	}
	wantErrorContains(t, declarations.BaselineErr, "'../uno.md' sale de workspaceRoot", "'/dos.md'", "el dominio 'a' está duplicado")
	if joined, ok := declarations.BaselineErr.(interface{ Unwrap() []error }); !ok || len(joined.Unwrap()) < 3 {
		t.Errorf("BaselineErr debe acumular un error por violación con errors.Join: %v", declarations.BaselineErr)
	}
	wantErrorContains(t, declarations.TechnicalContextErr, "requiere al menos una ruta en 'paths'")
}

func TestDeclarationsTechnicalContextValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  string // sustituye a la configuración base cuando no está vacío
		blocks  string
		valid   bool
		wantErr []string
	}{
		{"ambos modos", "", "technical_context:\n  maintained:\n    entry: context/TECHNICAL_CONTEXT.md\n    paths: [context/]\n  reference:\n    paths: [00_Glosario.md]\n", true, nil},
		{"solo referencia", "", "technical_context:\n  reference:\n    paths: [00_Glosario.md]\n", true, nil},
		{"bloque vacio", "", "technical_context: {}\n", false, []string{"requiere al menos el modo 'maintained' o 'reference'"}},
		{"bloque nulo", "", "technical_context:\n", false, []string{"requiere al menos el modo 'maintained' o 'reference'"}},
		{"bloque que no es un mapeo", "", "technical_context: [a]\n", false, []string{"debe ser un mapeo con los modos"}},
		{"modo desconocido", "", "technical_context:\n  readonly:\n    paths: [a.md]\n", false, []string{"clave desconocida 'readonly'", "technical_context", "claves admitidas: maintained, reference"}},
		{"clave desconocida en un modo", "", "technical_context:\n  reference:\n    paths: [a.md]\n    extra: 1\n", false, []string{"clave desconocida 'extra'", "technical_context.reference", "entry, paths"}},
		{"modo nulo", "", "technical_context:\n  maintained:\n", false, []string{"el modo 'technical_context.maintained' debe ser un mapeo"}},
		{"paths vacio", "", "technical_context:\n  maintained:\n    paths: []\n", false, []string{"el modo 'technical_context.maintained' requiere al menos una ruta"}},
		{"paths ausente", "", "technical_context:\n  reference:\n    entry: a.md\n", false, []string{"requiere al menos una ruta"}},
		{"entrada fuera de las rutas del modo", "", "technical_context:\n  maintained:\n    entry: README.md\n    paths: [context/]\n", false, []string{"'technical_context.maintained.entry' ('README.md') debe estar contenida"}},
		{"entrada igual a una ruta", "", "technical_context:\n  maintained:\n    entry: context/INDEX.md\n    paths: [context/INDEX.md]\n", true, nil},
		{"entrada contenida sin distinguir mayusculas", "", "technical_context:\n  maintained:\n    entry: Context/Index.md\n    paths: [context]\n", true, nil},
		{"entrada absoluta", "", "technical_context:\n  maintained:\n    entry: /abs/a.md\n    paths: [context/]\n", false, []string{"'technical_context.maintained.entry'", "relativa"}},
		{"ruta en ambos modos", "", "technical_context:\n  maintained:\n    paths: [context/a.md]\n  reference:\n    paths: [context/a.md]\n", false, []string{"'context/a.md'", "technical_context.maintained", "technical_context.reference", "no puede pertenecer a los dos modos"}},
		{"solapamiento por contencion", "", "technical_context:\n  maintained:\n    paths: [context/]\n  reference:\n    paths: [context/a.md]\n", false, []string{"'context/'", "'context/a.md'", "solapa"}},
		{"solapamiento sin distinguir mayusculas", "", "technical_context:\n  maintained:\n    paths: [Context/]\n  reference:\n    paths: [context/a.md]\n", false, []string{"solapa"}},
		{"prefijos que no son contencion", "", "technical_context:\n  maintained:\n    paths: [context]\n  reference:\n    paths: [context-extra]\n", true, nil},
		{"ruta compartida con la baseline", "", baselineDoc(proseDoc("modelo-de-datos", "03_Modelo_Datos.md")) + "technical_context:\n  reference:\n    paths: [03_Modelo_Datos.md]\n", false, []string{"'03_Modelo_Datos.md'", "technical_context.reference", "'baseline.documents'", "'modelo-de-datos'"}},
		{"directorio que contiene un documento de la baseline", "", baselineDoc(proseDoc("a", "context/a.md")) + "technical_context:\n  maintained:\n    paths: [context/]\n", false, []string{"solapa con el documento 'context/a.md'"}},
		{"ruta absoluta y ruta fuera de la maestra", "", "technical_context:\n  maintained:\n    paths: [/abs/dir, ../../fuera/]\n", false, []string{"'/abs/dir'", "relativa", "'../../fuera/'", "sale de la carpeta maestra"}},
		{"ruta que sale de la maestra en embebido", "", "technical_context:\n  reference:\n    paths: [../hermano.md]\n", false, []string{"'../hermano.md'", "sale de la carpeta maestra"}},
		{
			name:    "multirepo permite subir al repositorio hermano",
			config:  "workspace:\n  name: e\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: repo-backend\n",
			blocks:  "technical_context:\n  maintained:\n    paths: [../repo-backend/docs/]\n",
			valid:   true,
			wantErr: nil,
		},
		{
			name:    "multirepo no permite salir de la carpeta maestra",
			config:  "workspace:\n  name: e\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: repo-backend\n",
			blocks:  "technical_context:\n  maintained:\n    paths: [../../fuera/]\n",
			wantErr: []string{"sale de la carpeta maestra"},
		},
		{
			name:    "specs_repository terminado en openspec desplaza a su padre",
			config:  "workspace:\n  name: e\n  topology: multirepo\n  specs_repository: repo-specs/openspec\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n",
			blocks:  "technical_context:\n  maintained:\n    paths: [../repo-backend/docs/]\n  reference:\n    paths: [../../fuera.md]\n",
			wantErr: []string{"'../../fuera.md'", "sale de la carpeta maestra"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := declarationsBase
			if tc.config != "" {
				config = tc.config
			}
			cfg, baseErr, declarations := parseLayered([]byte(config + tc.blocks))
			if baseErr != nil {
				t.Fatalf("la configuración base es válida y no debe fallar: %v", baseErr)
			}
			if tc.valid {
				if declarations.TechnicalContextErr != nil || declarations.BaselineErr != nil {
					t.Fatalf("la declaración debe ser válida: contexto=%v baseline=%v", declarations.TechnicalContextErr, declarations.BaselineErr)
				}
				if declarations.TechnicalContext == nil || cfg.TechnicalContext == nil {
					t.Fatal("una declaración válida debe poblar el contexto técnico")
				}
				return
			}
			wantErrorContains(t, declarations.TechnicalContextErr, tc.wantErr...)
			if declarations.TechnicalContext != nil || cfg.TechnicalContext != nil {
				t.Error("un bloque inválido no debe poblar el contexto técnico ni parcial ni por defecto")
			}
		})
	}
}

// REQ-24.17: las claves desconocidas fuera de los bloques nuevos se siguen ignorando.
func TestDeclarationsUnknownKeysOutsideTheNewBlocksAreIgnored(t *testing.T) {
	data := declarationsBase + "clave_futura: 1\ngovernance:\n  language: es\n  otra: x\n"
	cfg, baseErr, declarations := parseLayered([]byte(data))
	if baseErr != nil || declarations.BaselineErr != nil || declarations.TechnicalContextErr != nil || declarations.ParseErr != nil {
		t.Fatalf("las claves desconocidas ajenas a los bloques nuevos no deben fallar: %v %+v", baseErr, declarations)
	}
	if cfg == nil || cfg.Governance.Language != "es" {
		t.Fatalf("la configuración debe cargarse: %+v", cfg)
	}
	if _, err := ParseConfig([]byte(data)); err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
}

// D-03: ParseErr solo es no nulo cuando existe un axiom.yaml que no se puede
// leer como YAML; sin contenido útil conserva el valor cero.
func TestDeclarationsParseErrOnlyForUnreadableYAML(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		parseErr bool
	}{
		{"yaml ilegible", "workspace: [sin cerrar\n", true},
		{"vacio", "", false},
		{"solo espacios", "  \n\n", false},
		{"solo comentarios", "# nada\n", false},
		{"base valida", declarationsBase, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, baseErr, declarations := parseLayered([]byte(tc.data))
			if (declarations.ParseErr != nil) != tc.parseErr {
				t.Fatalf("ParseErr = %v, se esperaba no nulo = %v", declarations.ParseErr, tc.parseErr)
			}
			if tc.parseErr && baseErr == nil {
				t.Error("un YAML ilegible también es un error de la configuración base")
			}
			if declarations.Baseline != nil || declarations.TechnicalContext != nil {
				t.Errorf("sin bloques legibles no hay declaraciones: %+v", declarations)
			}
		})
	}
}

// D-03, los cuatro casos de la tabla: cada canal de error es independiente.
func TestLayeredErrorChannels(t *testing.T) {
	invalidBaseline := "baseline:\n  documents: []\n"
	validBaseline := baselineDoc(proseDoc("a", "a.md"))
	const noName = "workspace:\n  topology: monorepo-embedded\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n"

	t.Run("YAML ilegible", func(t *testing.T) {
		cfg, baseErr, declarations := parseLayered([]byte("workspace: [sin cerrar\n"))
		if cfg != nil || baseErr == nil || declarations.ParseErr == nil {
			t.Fatalf("cfg=%v baseErr=%v ParseErr=%v", cfg, baseErr, declarations.ParseErr)
		}
		if _, err := ParseConfig([]byte("workspace: [sin cerrar\n")); err == nil {
			t.Error("ParseConfig debe fallar, como hasta ahora")
		}
	})

	t.Run("base invalida con declaracion valida", func(t *testing.T) {
		cfg, baseErr, declarations := parseLayered([]byte(noName + validBaseline))
		wantErrorContains(t, baseErr, "'workspace.name' es obligatorio")
		if cfg != nil {
			t.Error("una base inválida no devuelve configuración")
		}
		if declarations.BaselineErr != nil || declarations.Baseline == nil {
			t.Errorf("la declaración válida se evalúa aunque la base falle: %+v", declarations)
		}
		if declarations.ParseErr != nil {
			t.Errorf("ParseErr solo se refiere al YAML ilegible: %v", declarations.ParseErr)
		}
		if _, err := ParseConfig([]byte(noName + validBaseline)); err == nil || !strings.Contains(err.Error(), "'workspace.name'") {
			t.Errorf("ParseConfig debe fallar con el error de la base: %v", err)
		}
	})

	t.Run("declaracion invalida con base valida", func(t *testing.T) {
		cfg, baseErr, declarations := parseLayered([]byte(declarationsBase + invalidBaseline))
		if baseErr != nil {
			t.Fatalf("la base es válida: %v", baseErr)
		}
		if cfg == nil || cfg.Workspace.Name != "ejemplo" || cfg.Baseline != nil {
			t.Fatalf("la configuración base se lee sin el bloque inválido: %+v", cfg)
		}
		wantErrorContains(t, declarations.BaselineErr, "requiere al menos un documento")
		_, err := ParseConfig([]byte(declarationsBase + invalidBaseline))
		wantErrorContains(t, err, "requiere al menos un documento")
	})

	t.Run("declaracion invalida y base invalida", func(t *testing.T) {
		cfg, baseErr, declarations := parseLayered([]byte(noName + invalidBaseline))
		if cfg != nil || baseErr == nil || declarations.BaselineErr == nil {
			t.Fatalf("cfg=%v baseErr=%v BaselineErr=%v", cfg, baseErr, declarations.BaselineErr)
		}
		_, err := ParseConfig([]byte(noName + invalidBaseline))
		wantErrorContains(t, err, "'workspace.name'", "requiere al menos un documento")
	})

	t.Run("ParseConfig acumula los dos bloques", func(t *testing.T) {
		_, err := ParseConfig([]byte(declarationsBase + invalidBaseline + "technical_context: {}\n"))
		wantErrorContains(t, err, "requiere al menos un documento", "requiere al menos el modo 'maintained' o 'reference'")
	})
}

func TestDeclarationsLoadConfigRejectsAnInvalidDeclaration(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "axiom.yaml")
	if err := os.WriteFile(file, []byte(declarationsBase+baselineDoc(proseDoc("a", "../fuera.md"))), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(file)
	wantErrorContains(t, err, "el dominio 'a'", "sale de workspaceRoot")

	if err := os.WriteFile(file, []byte(declarationsBase+baselineDoc(proseDoc("a", "a.md"))), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(file)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Baseline == nil || len(cfg.Baseline.Documents) != 1 {
		t.Fatalf("LoadConfig debe incluir el bloque válido: %+v", cfg)
	}
}
