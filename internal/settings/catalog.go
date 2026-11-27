package settings

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// ValueType es el tipo de valor de una clave del catálogo.
type ValueType string

const (
	TypeString     ValueType = "string"
	TypeBool       ValueType = "bool"
	TypeInt        ValueType = "int"
	TypeEnum       ValueType = "enum"
	TypeStringList ValueType = "stringList"
	TypeYAML       ValueType = "yaml" // solo lectura: subárbol serializado (technical_context)
)

// Entry es una clave concreta del catálogo para un documento.
type Entry struct {
	Key            string    // "git.promotion", "repositories[repo-a].promotion", "roles[web].gate_policy"
	Section        string    // workspace, git, repositories, workflow, delivery, sdd, acceptance_tests, roles, technical_context, governance
	Label          string    //
	Description    string    //
	Effect         string    // efecto descrito de editar la clave
	Type           ValueType //
	Allowed        []string  // enum, ya filtrado por ISO
	Topologies     []string  // topologías en las que aplica
	Editable       bool      //
	ReadOnlyReason string    //
	Required       bool      //
	Default        string    // descripción del valor por defecto del motor
	Declared       bool      //
	Value          any       // string, bool, int, []string o nil
	Plugin         string    // id del plugin de integración; vacío fuera de integrations[<plugin>]
	Format         string    // formato declarado por el manifiesto (string y string_list)
	Min            *int      // cota inferior inclusiva de un entero del manifiesto
	Max            *int      // cota superior inclusiva de un entero del manifiesto
	Line           int       // línea de la clave en axiom.yaml (solo lectura declarada); 0 si no se conoce
}

// onlyNewChanges es el aviso común de toda clave editable: ajustar el proyecto no
// reescribe lo que ya está en curso.
const onlyNewChanges = "Solo afecta a cambios nuevos."

const (
	topologyMonorepo = string(workspace.TopologyMonorepoEmbedded)
	topologyMulti    = string(workspace.TopologyMultirepo)
)

// scopeKind indica cómo se expande una plantilla de clave con un documento.
type scopeKind int

const (
	scopeBlock  scopeKind = iota // una única clave, p. ej. git.promotion
	scopeRepo                    // repositories[<ruta>].<campo>
	scopeRole                    // roles[<id>].<campo>
	scopePlugin                  // integrations[<plugin>].<campo>, derivado del manifiesto
)

// keySpec es una fila de la tabla del catálogo. La plantilla de las claves por
// elemento lleva `[]` en lugar del selector: `repositories[].promotion`.
type keySpec struct {
	key           string
	section       string
	label         string
	description   string
	effect        string
	typ           ValueType
	allowed       func() []string // vocabulario cerrado sin filtrar por ISO
	isoDrop       []string        // valores que un proyecto ISO no admite
	required      bool
	engineChecked bool // el vocabulario lo valida el decodificador del motor, no el parche
	def           func() string
	multirepoOnly bool
	scope         scopeKind
	plugin        string // id del plugin (scopePlugin)
	format        string
	min, max      *int
	uiRequired    bool // obligatoriedad del manifiesto, solo informativa
}

// vocabulary devuelve el vocabulario cerrado de la clave, sin filtrar por ISO, o
// nil si la clave no es de vocabulario cerrado.
func (s keySpec) vocabulary() []string {
	if s.allowed == nil {
		return nil
	}
	return s.allowed()
}

func staticDefault(text string) func() string { return func() string { return text } }

func stringsOf[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

// editableSpecs es la tabla de claves editables. Los vocabularios cerrados salen de
// los accesores de su paquete dueño: añadir un valor al esquema lo publica aquí sin
// editar el catálogo.
var editableSpecs = []keySpec{
	{key: "workspace.name", section: "workspace", label: "Nombre del proyecto", description: "Nombre con el que se identifica el proyecto.", effect: "Cambia el nombre que muestran las pantallas y los informes. " + onlyNewChanges, typ: TypeString, required: true},
	{key: "git.integration_branch", section: "git", label: "Rama de integración", description: "Rama base de la que nacen los cambios y a la que vuelven.", effect: "Las ramas de cambio nuevas parten de esta rama. " + onlyNewChanges, typ: TypeString, def: staticDefault(workspace.DefaultIntegrationBranch)},
	{key: "git.branch_prefix", section: "git", label: "Prefijo de rama", description: "Prefijo de las ramas de cambio (por ejemplo, feat).", effect: "Las ramas de cambio nuevas usan este prefijo. " + onlyNewChanges, typ: TypeString, def: staticDefault(workspace.DefaultBranchPrefix)},
	{key: "git.main_branch", section: "git", label: "Rama principal", description: "Rama de producción a la que se promueve lo integrado, si es distinta de la de integración.", effect: "Habilita la promoción hacia esta rama. " + onlyNewChanges, typ: TypeString, def: staticDefault("sin rama principal")},
	{key: "git.promotion", section: "git", label: "Promoción", description: "Cómo se promueve un cambio integrado hacia la rama principal.", effect: "Decide si la promoción se hace por rama, por versión o se pregunta. " + onlyNewChanges, typ: TypeEnum, engineChecked: true, allowed: func() []string { return workspace.PromotionValues() }, def: staticDefault(workspace.PromotionBranch)},
	{key: "git.branch_template", section: "git", label: "Plantilla de nombre de rama", description: "Plantilla del nombre de las ramas de cambio, con variables entre llaves (por ejemplo, {prefix}/{ref}-{slug}).", effect: "Las ramas de cambio nuevas se nombran con esta plantilla. " + onlyNewChanges, typ: TypeString, def: staticDefault("sin plantilla")},
	{key: "workflow.lane", section: "workflow", label: "Carril de trabajo", description: "Carril con el que se aborda el trabajo sustancial: ágil (odd), formal (sdd) o preguntar cada vez (ask).", effect: "Los trabajos nuevos usan este carril. " + onlyNewChanges, typ: TypeEnum, engineChecked: true, allowed: func() []string { return stringsOf(workspace.WorkflowLanes()) }, isoDrop: []string{string(workspace.WorkflowLaneODD)}, def: staticDefault(string(workspace.DefaultWorkflowLane))},
	{key: "delivery.split_hint_lines", section: "delivery", label: "Umbral de aviso de cambio grande", description: "Número de líneas a partir del cual se avisa de que conviene dividir la entrega.", effect: "Los avisos de tamaño de cambios nuevos usan este umbral. " + onlyNewChanges, typ: TypeInt, def: staticDefault("sin aviso")},
	{key: "sdd.iso", section: "sdd", label: "Proyecto ISO", description: "Marca el proyecto como ISO.", effect: "Todo el trabajo pasa por SDD y verify es obligatorio antes de archivar. " + onlyNewChanges, typ: TypeBool, def: staticDefault("false")},
	{key: "sdd.id_pattern", section: "sdd", label: "Patrón de identificadores de requisito", description: "Plantilla de los identificadores de requisito de este proyecto.", effect: "Solo los requisitos nuevos usan la plantilla; los existentes no se renombran automáticamente. " + onlyNewChanges, typ: TypeString, def: func() string { return reqid.Default().Template() }},
	{key: "sdd.explainer_html", section: "sdd", label: "Explicador HTML", description: "Si el cambio genera un explicador HTML: preguntar, siempre o nunca.", effect: "Los cambios nuevos siguen este modo. " + onlyNewChanges, typ: TypeEnum, engineChecked: true, allowed: func() []string { return stringsOf(workspace.ExplainerHTMLModes()) }, def: staticDefault(string(workspace.ExplainerHTMLAsk))},
	{key: "acceptance_tests.enabled", section: "acceptance_tests", label: "Pruebas de aceptación", description: "Activa las pruebas de aceptación entre repositorios.", effect: "Los cambios nuevos exigen o no sus pruebas de aceptación. " + onlyNewChanges, typ: TypeBool, multirepoOnly: true, def: staticDefault("sin declarar")},
	{key: "acceptance_tests.environments", section: "acceptance_tests", label: "Entornos de aceptación", description: "Entornos en los que se ejecutan las pruebas de aceptación (uno por línea).", effect: "Los cambios nuevos usan estos entornos. " + onlyNewChanges, typ: TypeStringList, multirepoOnly: true, def: staticDefault("sin entornos")},
	{key: "repositories[].integration_branch", section: "repositories", label: "Rama de integración del repositorio", description: "Rama de integración propia de este repositorio.", effect: "Las ramas de cambio nuevas de este repositorio parten de ella. " + onlyNewChanges, typ: TypeString, multirepoOnly: true, scope: scopeRepo, def: staticDefault("la de git.integration_branch")},
	{key: "repositories[].branch_prefix", section: "repositories", label: "Prefijo de rama del repositorio", description: "Prefijo de rama propio de este repositorio.", effect: "Las ramas de cambio nuevas de este repositorio usan este prefijo. " + onlyNewChanges, typ: TypeString, multirepoOnly: true, scope: scopeRepo, def: staticDefault("el de git.branch_prefix")},
	{key: "repositories[].promotion", section: "repositories", label: "Promoción del repositorio", description: "Modo de promoción propio de este repositorio.", effect: "La promoción de este repositorio sigue este modo. " + onlyNewChanges, typ: TypeEnum, engineChecked: true, allowed: func() []string { return workspace.PromotionValues() }, multirepoOnly: true, scope: scopeRepo, def: staticDefault("el de git.promotion")},
	{key: "repositories[].branch_template", section: "repositories", label: "Plantilla de rama del repositorio", description: "Plantilla del nombre de las ramas de cambio propia de este repositorio.", effect: "Las ramas de cambio nuevas de este repositorio se nombran con esta plantilla. " + onlyNewChanges, typ: TypeString, multirepoOnly: true, scope: scopeRepo, def: staticDefault("la de git.branch_template")},
	{key: "roles[].gate_policy", section: "roles", label: "Política de compuerta del rol", description: "Exigencia del rol frente a la barrera de sincronización: bloqueante, diferida u opcional.", effect: "Los cambios nuevos aplican esta política al rol. " + onlyNewChanges, typ: TypeEnum, allowed: func() []string { return stringsOf(multirole.GatePolicies()) }, multirepoOnly: true, scope: scopeRole, def: staticDefault(string(multirole.PolicyBlocking))},
	{key: "roles[].branch_template", section: "roles", label: "Plantilla de rama del rol", description: "Plantilla del nombre de las ramas de cambio propia de este rol.", effect: "Las ramas de cambio nuevas de este rol se nombran con esta plantilla. " + onlyNewChanges, typ: TypeString, multirepoOnly: true, scope: scopeRole, def: staticDefault("la del repositorio o la de git.branch_template")},
}

// readOnlySpecs son las claves que el catálogo muestra sin permitir editarlas.
var readOnlySpecs = []keySpec{
	{key: "workspace.topology", section: "workspace", label: "Topología", description: "Organización de repositorios del proyecto.", typ: TypeString},
	{key: "workspace.specs_repository", section: "workspace", label: "Repositorio de specs", description: "Carpeta o repositorio donde viven las especificaciones.", typ: TypeString},
	{key: "workspace.root", section: "workspace", label: "Raíz del proyecto", description: "Ruta base del proyecto.", typ: TypeString},
	{key: "workspace.skill_roots", section: "workspace", label: "Carpetas de skills", description: "Carpetas versionadas con skills canónicas.", typ: TypeStringList},
	{key: "technical_context", section: "technical_context", label: "Contexto técnico", description: "Documentación técnica mantenida del proyecto.", typ: TypeYAML},
}

const readOnlyReason = "Esta clave no se edita desde Ajustes: edítala a mano en axiom.yaml."

// governanceLabels da nombre legible a las claves conocidas del bloque `governance`; una
// clave desconocida se muestra con su ruta.
var governanceLabels = map[string]string{
	"language":      "Idioma",
	"shared_memory": "Memoria compartida",
}

// governanceSpec describe una clave declarada del bloque `governance`.
func governanceSpec(name string) keySpec {
	label := governanceLabels[name]
	if label == "" {
		label = "governance." + name
	}
	return keySpec{key: "governance." + name, section: "governance", label: label, description: "Gobierno transversal del proyecto.", typ: TypeString}
}

// normalizeRepoPath compara rutas de repositorio como el motor: `\` pasa a `/` y se
// limpia con path.Clean.
func normalizeRepoPath(p string) string {
	return path.Clean(strings.ReplaceAll(strings.TrimSpace(p), `\`, "/"))
}

// parsedKey es una clave del parche analizada: la plantilla del catálogo, el
// selector de los elementos (ruta del repositorio o id del rol) y el campo.
type parsedKey struct {
	template string
	selector string
	field    string
	scope    scopeKind
}

// parseKey analiza una clave del parche. Las claves por elemento son
// `repositories[<ruta>].<campo>` y `roles[<id>].<campo>`; el selector va desde el
// primer `[` hasta el último `].`, de modo que un `]` dentro de la ruta no rompe el
// análisis. Cualquier otra clave se devuelve tal cual como clave de bloque.
func parseKey(key string) parsedKey {
	for _, prefix := range []struct {
		name  string
		scope scopeKind
	}{{"repositories[", scopeRepo}, {"roles[", scopeRole}, {"integrations[", scopePlugin}} {
		if !strings.HasPrefix(key, prefix.name) {
			continue
		}
		end := strings.LastIndex(key, "].")
		if end < len(prefix.name) {
			break
		}
		field := key[end+2:]
		return parsedKey{template: prefix.name + "]." + field, selector: key[len(prefix.name):end], field: field, scope: prefix.scope}
	}
	return parsedKey{template: key, scope: scopeBlock}
}

// findSpec devuelve la fila del catálogo editable de una plantilla.
func findSpec(template string) (keySpec, bool) {
	i := slices.IndexFunc(editableSpecs, func(s keySpec) bool { return s.key == template })
	if i < 0 {
		return keySpec{}, false
	}
	return editableSpecs[i], true
}

// pluginTemplate es la plantilla de las claves de un plugin: `integrations[].<campo>`.
func pluginTemplate(field string) string { return "integrations[]." + field }

// pluginKey es la clave concreta de un campo de un plugin.
func pluginKey(plugin, field string) string { return "integrations[" + plugin + "]." + field }

// pluginSpecs deriva del manifiesto las filas de un plugin: `enabled` y una por cada campo
// de su configuración. La obligatoriedad es condicional a `enabled`, así que el parche no la
// exige (required false) y el decodificador estricto la revalida sobre el documento
// resultante. Los secretos del manifiesto no son campos de Config y nunca aparecen aquí.
func pluginSpecs(m manifest.Manifest) []keySpec {
	specs := []keySpec{{
		key: pluginTemplate(manifest.ReservedConfigKey), section: "integrations", label: manifest.ReservedConfigKey,
		description: "Activa o desactiva el plugin " + m.ID + ".", effect: "Activa o desactiva la integración. " + onlyNewChanges,
		typ: TypeBool, engineChecked: true, scope: scopePlugin, plugin: m.ID,
		def: staticDefault("false"),
	}}
	for _, f := range m.Config {
		spec := keySpec{
			key: pluginTemplate(f.Key), section: "integrations", label: f.Key, description: f.Description,
			effect:        "Cambia la configuración de la integración. " + onlyNewChanges,
			engineChecked: true, uiRequired: f.Required, scope: scopePlugin, plugin: m.ID, format: string(f.Format), min: f.Min, max: f.Max,
			def: staticDefault(describeDefault(f.Default)),
		}
		switch f.Type {
		case manifest.FieldBool:
			spec.typ = TypeBool
		case manifest.FieldInt:
			spec.typ = TypeInt
		case manifest.FieldStringList:
			spec.typ = TypeStringList
		default:
			spec.typ = TypeString
			if len(f.Enum) > 0 {
				spec.typ = TypeEnum
			}
		}
		if len(f.Enum) > 0 {
			enum := slices.Clone(f.Enum)
			spec.allowed = func() []string { return slices.Clone(enum) }
		}
		specs = append(specs, spec)
	}
	return specs
}

// describeDefault describe el valor por defecto de un campo del manifiesto.
func describeDefault(v any) string {
	switch d := v.(type) {
	case nil:
		return ""
	case []string:
		return strings.Join(d, ", ")
	case []any:
		parts := make([]string, len(d))
		for i, e := range d {
			parts[i] = fmt.Sprint(e)
		}
		return strings.Join(parts, ", ")
	}
	return fmt.Sprint(v)
}

// findPluginSpec devuelve la fila de un campo de un plugin del catálogo de manifiestos.
// Un plugin ajeno al catálogo, un campo que el manifiesto no declara y el nombre de un
// secreto devuelven false.
func findPluginSpec(plugin, field string) (keySpec, bool) {
	m, ok := manifest.Lookup(plugin)
	if !ok || slices.ContainsFunc(m.Secrets, func(s manifest.SecretSpec) bool { return s.Name == field }) {
		return keySpec{}, false
	}
	for _, spec := range pluginSpecs(m) {
		if spec.key == pluginTemplate(field) {
			return spec, true
		}
	}
	return keySpec{}, false
}
