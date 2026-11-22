package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/builtin"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/identitystore"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// `axiom integrations connect` y `discover` (C-07-c): verbos genéricos sobre los puertos opcionales
// de identidad y descubrimiento de un plugin (internal/integrations/ports.go). No hacen preguntas
// interactivas: lo que falta se pide con una bandera y un código de salida distinto de cero.

const (
	integrationsConnectSchema = "axiom.integrations-connect/v1"

	connectStatusConnected         = "connected"
	connectStatusSelectionRequired = "selection_required"
)

const integrationsConnectHelp = `Uso: axiom integrations connect --plugin <id> [--user <id>] [--cwd <ruta>] [--json]

Prueba la conexión con el tracker con el secreto del almacén (axiom integrations secret), resuelve tu
identidad en él y la guarda en ~/.axiom/integration-identities.json (respeta AXIOM_STATE_DIR),
indexada por plugin y organización. El fichero no contiene secretos y no se versiona.

Sin --user, el plugin intenta resolver la identidad por sí solo (en azure-devops, connectionData). Si
no puede, lista los miembros del equipo con su id y su nombre, no guarda nada y sale con 1: repite el
comando con --user <id>. Con --user no se consulta connectionData. No hay preguntas interactivas.

Banderas:
  --plugin <id>   Plugin del catálogo configurado en integrations: de axiom.yaml (obligatorio)
  --user <id>     Identificador del usuario del tracker, elegido de entre los miembros del equipo
  --cwd <ruta>    Directorio de partida (por defecto, el directorio de trabajo)
  --json          Imprime el resultado como ` + integrationsConnectSchema + `

Sale con 0 si se guardó la identidad; con 1 si falló o hace falta --user; con 2 por un error de uso.
`

const integrationsDiscoverHelp = `Uso: axiom integrations discover --plugin <id> [--cwd <ruta>] [--json]

Imprime el proceso del tracker: tipos de elemento con sus estados (y su categoría tal como la
devuelve el tracker) y sus campos, iteraciones del equipo y miembros. Es de solo lectura: no escribe
ningún fichero. Un fallo en cualquier consulta hace fallar el descubrimiento entero; nunca se imprime
un resultado parcial como si fuera completo.

Banderas:
  --plugin <id>   Plugin del catálogo configurado en integrations: de axiom.yaml (obligatorio)
  --cwd <ruta>    Directorio de partida (por defecto, el directorio de trabajo)
  --json          Imprime el resultado como ` + integrations.DiscoverySchemaV1 + `

Sale con 0 si el descubrimiento terminó; con 1 si falló; con 2 por un error de uso.
`

// Errores tipados de los verbos que despachan por puertos (REQ-45.12 y REQ-45.13). Se comprueban con
// errors.Is; el texto de cada uno dice qué corregir.
var (
	// ErrIntegrationPluginUnknown: --plugin no está en el catálogo cerrado.
	ErrIntegrationPluginUnknown = errors.New("plugin de integración desconocido")
	// ErrIntegrationNotConfigured: axiom.yaml no declara integrations.<plugin>.
	ErrIntegrationNotConfigured = errors.New("plugin de integración no configurado")
	// ErrIntegrationDisabled: integrations.<plugin>.enabled es false.
	ErrIntegrationDisabled = errors.New("plugin de integración desactivado")
	// ErrIntegrationConfigInvalid: el bloque integrations: no es válido (por ejemplo, falta una clave
	// obligatoria de un plugin activo).
	ErrIntegrationConfigInvalid = errors.New("configuración de integraciones inválida")
	// ErrIntegrationNotRegistered: el plugin está en el catálogo pero no está compilado en esta versión.
	ErrIntegrationNotRegistered = errors.New("plugin de integración no registrado")
)

// integrationVerbError es un error tipado de un verbo de integraciones: kind es el centinela y
// message el texto que ve la persona.
type integrationVerbError struct {
	kind    error
	message string
}

func (e *integrationVerbError) Error() string        { return e.message }
func (e *integrationVerbError) Is(target error) bool { return target == e.kind }

func verbError(kind error, format string, args ...any) error {
	return &integrationVerbError{kind: kind, message: fmt.Sprintf(format, args...)}
}

// portTarget es el plugin sobre el que actúa un verbo de puertos, ya validado.
type portTarget struct {
	manifest manifest.Manifest
	plugin   integrations.Plugin
	config   workspace.IntegrationPluginConfig
}

// resolvePortTarget resuelve el plugin de un verbo de puertos. Orden: catálogo (antes de leer el
// workspace), bloque integrations:, plugin declarado y activo, y registro compilado.
func resolvePortTarget(cwd, pluginID string) (portTarget, error) {
	m, err := resolveSecretTarget(pluginID, "", false)
	if err != nil {
		return portTarget{}, verbError(ErrIntegrationPluginUnknown, "%v", err)
	}

	loc, err := workspace.Locate(cwd)
	if err != nil {
		return portTarget{}, err
	}
	if loc.Declarations.IntegrationsErr != nil {
		return portTarget{}, verbError(ErrIntegrationConfigInvalid,
			"el bloque `integrations:` de axiom.yaml no es válido: %v; corrígelo con `axiom workspace validate`", loc.Declarations.IntegrationsErr)
	}

	var declared *workspace.IntegrationPluginConfig
	if loc.Declarations.Integrations != nil {
		for i, p := range loc.Declarations.Integrations.Plugins {
			if p.ID == pluginID {
				declared = &loc.Declarations.Integrations.Plugins[i]
				break
			}
		}
	}
	switch {
	case declared == nil:
		return portTarget{}, verbError(ErrIntegrationNotConfigured,
			"el plugin %s no está configurado en `integrations:` de axiom.yaml: añade el bloque `integrations.%s` (plantilla en axiom.example.yaml)", pluginID, pluginID)
	case !declared.Enabled:
		return portTarget{}, verbError(ErrIntegrationDisabled,
			"el plugin %s está desactivado (`enabled: false` en `integrations.%s` de axiom.yaml)", pluginID, pluginID)
	}

	registry, err := builtin.DefaultRegistry()
	if err != nil {
		return portTarget{}, err
	}
	plugin, ok := registry.Get(pluginID)
	if !ok {
		return portTarget{}, verbError(ErrIntegrationNotRegistered, "el plugin %s está en el catálogo pero no está compilado en esta versión de axiom", pluginID)
	}
	return portTarget{manifest: m, plugin: plugin, config: *declared}, nil
}

// connectResult es lo que devuelve connectIntegration: la identidad guardada, el fichero y, si la
// organización ya tenía una entrada, la que se sustituye.
type connectResult struct {
	Identity     integrations.TrackerIdentity
	ResolvedAt   string
	IdentityFile string
	Replaced     *identitystore.Entry
}

// connectIntegration resuelve la identidad por el puerto y la guarda. La falta de puerto, la
// configuración y el PAT ausente salen antes de abrir ningún fichero.
func connectIntegration(ctx context.Context, cwd, pluginID, user string) (connectResult, error) {
	target, err := resolvePortTarget(cwd, pluginID)
	if err != nil {
		return connectResult{}, err
	}
	identity, err := integrations.ResolveIdentity(ctx, target.plugin, target.config, secrets.Default(), user)
	if err != nil {
		return connectResult{}, err
	}
	path, err := identitystore.DefaultPath()
	if err != nil {
		return connectResult{}, err
	}
	resolvedAt := nowUTC()
	entry := identitystore.Entry{ID: identity.ID, DisplayName: identity.DisplayName, Source: identity.Source, ResolvedAt: resolvedAt}
	store := identitystore.Store{Path: path}
	previous, hadPrevious, err := store.Get(pluginID, identity.Account)
	if err != nil {
		return connectResult{}, err
	}
	if err := store.Put(pluginID, identity.Account, entry); err != nil {
		return connectResult{}, err
	}
	result := connectResult{Identity: identity, ResolvedAt: resolvedAt.Format("2006-01-02T15:04:05Z"), IdentityFile: path}
	if hadPrevious {
		result.Replaced = &previous
	}
	return result, nil
}

// discoverIntegration invoca el puerto de descubrimiento. No escribe ningún fichero.
func discoverIntegration(ctx context.Context, cwd, pluginID string) (integrations.Discovery, error) {
	target, err := resolvePortTarget(cwd, pluginID)
	if err != nil {
		return integrations.Discovery{}, err
	}
	return integrations.Discover(ctx, target.plugin, target.config, secrets.Default())
}

type integrationsConnectIdentityJSON struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Source      string `json:"source"`
	ResolvedAt  string `json:"resolved_at"`
}

type integrationsConnectCandidateJSON struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// integrationsConnectJSON es el documento de `connect --json`. organization repite account con el
// nombre que le da Azure DevOps (REQ-45.12); account es la clave genérica del almacén de identidad.
type integrationsConnectJSON struct {
	Schema       string                             `json:"schema"`
	Plugin       string                             `json:"plugin"`
	Account      string                             `json:"account"`
	Organization string                             `json:"organization"`
	Status       string                             `json:"status"`
	Identity     *integrationsConnectIdentityJSON   `json:"identity,omitempty"`
	IdentityFile string                             `json:"identity_file,omitempty"`
	User         string                             `json:"user,omitempty"`
	Reason       string                             `json:"reason,omitempty"`
	Candidates   []integrationsConnectCandidateJSON `json:"candidates,omitempty"`
}

func runIntegrationsConnect(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("integrations connect", flag.ContinueOnError)
	cwd := flags.String("cwd", ".", "directorio de partida")
	pluginID := flags.String("plugin", "", "plugin del catálogo")
	user := flags.String("user", "", "usuario del tracker elegido entre los miembros del equipo")
	asJSON := flags.Bool("json", false, "imprime el resultado como JSON")
	if code, done := parseIntegrationsFlags(flags, args, integrationsConnectHelp, stdout, stderr); done {
		return code
	}
	if strings.TrimSpace(*pluginID) == "" {
		fmt.Fprintln(stderr, "Error: --plugin es obligatorio. Ejecuta `axiom integrations connect --help`.")
		return integrationsExitUsage
	}

	result, err := connectIntegration(context.Background(), *cwd, *pluginID, strings.TrimSpace(*user))
	if err != nil {
		var selection *integrations.IdentitySelectionError
		if errors.As(err, &selection) {
			return reportSelectionRequired(selection, *pluginID, *asJSON, stdout, stderr)
		}
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return integrationsExitFail
	}

	identity := result.Identity
	if *asJSON {
		writeIntegrationsJSON(stdout, integrationsConnectJSON{
			Schema: integrationsConnectSchema, Plugin: *pluginID, Account: identity.Account, Organization: identity.Account,
			Status: connectStatusConnected,
			Identity: &integrationsConnectIdentityJSON{
				ID: identity.ID, DisplayName: identity.DisplayName, Source: identity.Source, ResolvedAt: result.ResolvedAt,
			},
			IdentityFile: result.IdentityFile,
		})
		return integrationsExitOK
	}
	fmt.Fprintf(stdout, "Conectado a %s (organización %s) como %s (id %s), origen %s. Identidad guardada en %s.\n",
		*pluginID, identity.Account, oneLine(identity.DisplayName), identity.ID, identity.Source, result.IdentityFile)
	if result.Replaced != nil {
		fmt.Fprintf(stdout, "Sustituye la identidad anterior de esa organización (id %s, origen %s).\n", result.Replaced.ID, result.Replaced.Source)
	}
	return integrationsExitOK
}

// reportSelectionRequired imprime la petición de --user con los candidatos tomados del campo
// Candidates (no del texto del error, que el saneado recorta). No se guarda nada.
func reportSelectionRequired(selection *integrations.IdentitySelectionError, pluginID string, asJSON bool, stdout, stderr io.Writer) int {
	reason := oneLine(selection.Reason)
	if selection.User != "" {
		reason = fmt.Sprintf("el usuario %q no coincide con ningún miembro del equipo", oneLine(selection.User))
	}
	if len(selection.Candidates) == 0 {
		fmt.Fprintf(stderr, "Error: no se pudo resolver la identidad en %s (%s) y el equipo no devolvió ningún miembro que elegir con --user: revisa `project` y `team` de integrations.%s en axiom.yaml y los scopes del PAT.\n",
			pluginID, reason, pluginID)
		return integrationsExitFail
	}

	candidates := make([]integrationsConnectCandidateJSON, len(selection.Candidates))
	for i, c := range selection.Candidates {
		candidates[i] = integrationsConnectCandidateJSON{ID: c.ID, DisplayName: oneLine(c.DisplayName)}
	}
	if asJSON {
		writeIntegrationsJSON(stdout, integrationsConnectJSON{
			Schema: integrationsConnectSchema, Plugin: pluginID, Account: selection.Account, Organization: selection.Account,
			Status: connectStatusSelectionRequired, User: selection.User, Reason: reason, Candidates: candidates,
		})
	}
	fmt.Fprintf(stderr, "Error: hace falta elegir tu usuario con --user: %s. No se guarda ninguna identidad.\n", reason)
	if !asJSON {
		fmt.Fprintln(stderr, "Miembros candidatos del equipo (id  nombre):")
		for _, c := range candidates {
			fmt.Fprintf(stderr, "  %s  %s\n", c.ID, c.DisplayName)
		}
	}
	fmt.Fprintf(stderr, "Repite con `axiom integrations connect --plugin %s --user <id>`.\n", pluginID)
	return integrationsExitFail
}

func runIntegrationsDiscover(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("integrations discover", flag.ContinueOnError)
	cwd := flags.String("cwd", ".", "directorio de partida")
	pluginID := flags.String("plugin", "", "plugin del catálogo")
	asJSON := flags.Bool("json", false, "imprime el resultado como JSON")
	if code, done := parseIntegrationsFlags(flags, args, integrationsDiscoverHelp, stdout, stderr); done {
		return code
	}
	if strings.TrimSpace(*pluginID) == "" {
		fmt.Fprintln(stderr, "Error: --plugin es obligatorio. Ejecuta `axiom integrations discover --help`.")
		return integrationsExitUsage
	}

	discovery, err := discoverIntegration(context.Background(), *cwd, *pluginID)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return integrationsExitFail
	}
	if *asJSON {
		writeIntegrationsJSON(stdout, discovery)
		return integrationsExitOK
	}
	printDiscovery(stdout, discovery)
	return integrationsExitOK
}

// printDiscovery imprime el descubrimiento en una forma legible equivalente al JSON.
func printDiscovery(stdout io.Writer, d integrations.Discovery) {
	keys := make([]string, 0, len(d.Scope))
	for k := range d.Scope {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	scope := make([]string, len(keys))
	for i, k := range keys {
		scope[i] = k + " " + oneLine(d.Scope[k])
	}
	fmt.Fprintf(stdout, "%s — %s\n", d.Plugin, strings.Join(scope, ", "))

	fmt.Fprintf(stdout, "Tipos de work item (%d):\n", len(d.ItemTypes))
	for _, t := range d.ItemTypes {
		fmt.Fprintf(stdout, "  %s (%s)\n", oneLine(t.Name), oneLine(t.ReferenceName))
		states := make([]string, len(t.States))
		for i, s := range t.States {
			states[i] = fmt.Sprintf("%s [%s]", oneLine(s.Name), oneLine(s.Category))
		}
		fmt.Fprintf(stdout, "    Estados: %s\n", strings.Join(states, ", "))
		fields := make([]string, len(t.Fields))
		for i, f := range t.Fields {
			fields[i] = oneLine(f.ReferenceName)
		}
		fmt.Fprintf(stdout, "    Campos (%d): %s\n", len(fields), strings.Join(fields, ", "))
	}

	fmt.Fprintf(stdout, "Iteraciones (%d):\n", len(d.Iterations))
	for _, it := range d.Iterations {
		line := "  " + oneLine(it.Name)
		if it.StartDate != "" || it.FinishDate != "" {
			line += fmt.Sprintf("  %s → %s", shortDate(it.StartDate), shortDate(it.FinishDate))
		}
		if it.TimeFrame != "" {
			line += "  " + oneLine(it.TimeFrame)
		}
		fmt.Fprintln(stdout, line)
	}

	fmt.Fprintf(stdout, "Miembros (%d):\n", len(d.Members))
	for _, m := range d.Members {
		fmt.Fprintf(stdout, "  %s  %s\n", m.ID, oneLine(m.DisplayName))
	}
}

// shortDate recorta una marca de tiempo ISO 8601 a su fecha para la salida legible; el JSON conserva
// el texto de la API.
func shortDate(value string) string {
	if date, _, found := strings.Cut(value, "T"); found {
		return date
	}
	return value
}

// oneLine aplana un texto que viene del tracker a una sola línea.
func oneLine(text string) string { return strings.Join(strings.Fields(text), " ") }
