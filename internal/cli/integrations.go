package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattn/go-isatty"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// `axiom integrations` (C-07-b): recuperación (`sync`), respaldo (`emit`) y ciclo de vida del secreto
// (`secret`) de los plugins de integración configurados en el bloque `integrations:` de axiom.yaml.
// `connect` y `discover` (C-07-c, integrations_ports.go) despachan por los puertos opcionales de
// identidad y descubrimiento.
// Códigos de salida como `axiom branch`: 0 éxito, 1 rechazo, error o algo pendiente, 2 error de uso.

const (
	integrationsExitOK    = 0
	integrationsExitFail  = 1
	integrationsExitUsage = 2

	// maxSecretBytes es el tamaño máximo del valor de `secret set` (4 KiB).
	maxSecretBytes = 4096

	integrationsSyncSchema         = "axiom.integrations-sync/v1"
	integrationsEmitSchema         = "axiom.integrations-emit/v1"
	integrationsSecretStatusSchema = "axiom.integrations-secret-status/v1"
)

const integrationsHelp = `Uso: axiom integrations <subcomando> [argumentos]

Subcomandos:
  sync      Reintenta las entregas pendientes a los trackers configurados en axiom.yaml
  emit      Emite a mano un evento del ciclo de vida de un cambio a los plugins configurados
  secret    Guarda (set, solo por stdin), consulta (status) o borra (delete) un secreto de integración
  connect   Prueba la conexión con el tracker y guarda tu identidad en ~/.axiom (--plugin, --user)
  discover  Imprime el proceso del tracker: tipos, estados, campos, iteraciones y miembros (--plugin, --json)
  tasks     Crea (create) o vincula a mano (bind) la Task hija del work item de origen de cada rol (--change, --estimate, --replace)
  summary   Escribe (push) el resumen de la spec y las etiquetas de requisito en el work item de origen del cambio (--change)

Ejecuta ` + "`axiom integrations <subcomando> --help`" + ` para ver las banderas de cada subcomando.
`

const integrationsSyncHelp = `Uso: axiom integrations sync [--cwd <ruta>] [--change <cambio>] [--retry-abandoned] [--discard <clave>]... [--json]

Reintenta, en orden de encolado, las operaciones pendientes de las colas locales
(.integrations-outbox.yaml) de la carpeta de cada cambio y de las carpetas archivadas, con un plazo
de 30 s por operación. Un fallo no impide las siguientes. Sin --change trata todas las carpetas con
cola; con él, la carpeta ordinaria del cambio y su carpeta archivada más reciente. Sin cola no lee ni
escribe nada.

Banderas:
  --change <cambio>    Limita el sync a un cambio
  --retry-abandoned    Reactiva las operaciones abandonadas (pone sus intentos a 0)
  --discard <clave>    Retira una operación, pendiente o abandonada, sin entregarla (repetible)
  --cwd <ruta>         Directorio de partida (por defecto, el directorio de trabajo)
  --json               Imprime el resultado como ` + integrationsSyncSchema + `

Sale con 0 solo si no queda ninguna operación pendiente ni abandonada; con 1 si queda alguna.
`

const integrationsEmitHelp = `Uso: axiom integrations emit --change <cambio> --event <tipo> [--role <rol>] [--actor <id>] [--cwd <ruta>] [--json]

Emite a mano un evento del ciclo de vida a los plugins configurados que lo escuchan. Los eventos de
compuerta (gate.spec.approved, gate.design.approved, role.apply.approved) exigen que la última
decisión de esa compuerta en gates.yaml sea approved: no se fabrican aprobaciones. Los eventos de rol
(role.start, role.apply.approved) exigen --role de un rol del roster sellado. archive exige la
carpeta archivada del cambio. change.start no se emite a mano en esta versión (lo emitirá C-07-d).

Banderas:
  --change <cambio>   Cambio (obligatorio)
  --event <tipo>      Evento del catálogo (obligatorio)
  --role <rol>        Rol del roster sellado, para los eventos de rol
  --actor <id>        Actor; por defecto el del registro de la compuerta o la identidad git
  --cwd <ruta>        Directorio de partida (por defecto, el directorio de trabajo)
  --json              Imprime el resultado como ` + integrationsEmitSchema + `

Sale con 0 si se entregó todo o si ningún plugin activo escucha el evento; con 1 si algo queda
pendiente o abandonado.
`

const integrationsSecretHelp = `Uso: axiom integrations secret <set|status|delete> --plugin <id> [--name <secreto>]

  set      --plugin <id> --name <secreto>   Guarda el secreto en el almacén del sistema; el valor entra SOLO por stdin
  status   --plugin <id> [--name <secreto>] [--json]   Informa de dónde sale cada secreto (env, keyring, missing, keyring_unavailable); nunca el valor
  delete   --plugin <id> --name <secreto>   Borra el secreto del almacén del sistema (idempotente)

El almacén del sistema solo existe en Windows (Credential Manager) y macOS (llavero). En Linux y en
cualquier otro sistema el único almacén es la variable de entorno
AXIOM_INTEGRATION_<PLUGIN>_<SECRETO> y set y delete se niegan. Axiom nunca escribe un secreto en un
fichero en texto plano.
`

const integrationsSecretSetHelp = `Uso: axiom integrations secret set --plugin <id> --name <secreto> [--cwd <ruta>]

Lee el valor únicamente por stdin (máximo 4 KiB; se quita un único salto de línea final y se rechaza
el valor vacío). No existe --value: un argumento sería visible en la lista de procesos. Si stdin es un
terminal se niega: canaliza el valor, por ejemplo
  Read-Host -MaskInput | axiom integrations secret set --plugin <id> --name <secreto>
  printf '%s' "$VALOR" | axiom integrations secret set --plugin <id> --name <secreto>
`

const integrationsSecretStatusHelp = `Uso: axiom integrations secret status --plugin <id> [--name <secreto>] [--json]

Por cada secreto del manifiesto del plugin informa de su origen (source): env, keyring, missing o
keyring_unavailable. Nunca imprime el valor ni su longitud. Con --json imprime ` + integrationsSecretStatusSchema + `.
`

const integrationsSecretDeleteHelp = `Uso: axiom integrations secret delete --plugin <id> --name <secreto>

Borra el secreto del almacén del sistema. Es idempotente: sin secreto, lo informa y sale con 0. Una
variable de entorno no se borra desde Axiom.
`

func nowUTC() time.Time { return time.Now().UTC() }

// stdinIsTerminal es la costura de prueba sobre la detección de terminal de stdin.
var stdinIsTerminal = func(stdin io.Reader) bool {
	file, ok := stdin.(*os.File)
	return ok && (isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd()))
}

// RunIntegrations ejecuta `axiom integrations`.
func RunIntegrations(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, integrationsHelp)
		return integrationsExitUsage
	}
	switch subcommand, rest := args[0], args[1:]; subcommand {
	case "--help", "-h", "help":
		fmt.Fprint(stdout, integrationsHelp)
		return integrationsExitOK
	case "sync":
		return runIntegrationsSync(rest, stdout, stderr)
	case "emit":
		return runIntegrationsEmit(rest, stdout, stderr)
	case "secret":
		return runIntegrationsSecret(rest, stdin, stdout, stderr)
	case "connect":
		return runIntegrationsConnect(rest, stdout, stderr)
	case "discover":
		return runIntegrationsDiscover(rest, stdout, stderr)
	case "summary":
		return runIntegrationsSummary(rest, stdout, stderr)
	case "tasks":
		return runIntegrationsTasks(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Error: subcomando '%s' no reconocido para integrations; los disponibles son sync, emit, secret, connect, discover, tasks y summary.\n", subcommand)
		fmt.Fprint(stdout, integrationsHelp)
		return integrationsExitUsage
	}
}

// parseIntegrationsFlags analiza las banderas; devuelve (código, true) si hay que terminar ya.
func parseIntegrationsFlags(flags *flag.FlagSet, args []string, help string, stdout, stderr io.Writer) (int, bool) {
	flags.SetOutput(io.Discard)
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, help)
			return integrationsExitOK, true
		}
		fmt.Fprintf(stderr, "Error: %v. Ejecuta con --help para ver las banderas.\n", err)
		return integrationsExitUsage, true
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "Error: argumento inesperado %q. Ejecuta con --help para ver las banderas.\n", flags.Arg(0))
		return integrationsExitUsage, true
	}
	return 0, false
}

func writeIntegrationsJSON(stdout io.Writer, value any) {
	payload, _ := json.MarshalIndent(value, "", "  ")
	fmt.Fprintln(stdout, string(payload))
}

// locateIntegrationsWorkspace resuelve el espacio de trabajo y rechaza un bloque inválido: son verbos
// explícitos de integración, así que es un error y no un aviso.
func locateIntegrationsWorkspace(cwd string, stderr io.Writer) (workspace.Location, bool) {
	loc, err := workspace.Locate(cwd)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return workspace.Location{}, false
	}
	if loc.Declarations.IntegrationsErr != nil {
		fmt.Fprintf(stderr, "Error: el bloque `integrations:` de axiom.yaml no es válido: %v\nCorrígelo con `axiom workspace validate`.\n", loc.Declarations.IntegrationsErr)
		return workspace.Location{}, false
	}
	return loc, true
}

// ---------------------------------------------------------------- sync

type integrationsSyncChangeJSON struct {
	Change    string                         `json:"change"`
	Archived  bool                           `json:"archived,omitempty"`
	Delivered []integrations.OperationReport `json:"delivered"`
	Pending   []integrations.OperationReport `json:"pending"`
	Abandoned []integrations.OperationReport `json:"abandoned"`
	Discarded []integrations.OperationReport `json:"discarded"`
	Warnings  []string                       `json:"warnings,omitempty"`
}

type integrationsSyncJSON struct {
	Schema    string                       `json:"schema"`
	Changes   []integrationsSyncChangeJSON `json:"changes"`
	Remaining bool                         `json:"remaining"`
}

func nonNil(reports []integrations.OperationReport) []integrations.OperationReport {
	if reports == nil {
		return []integrations.OperationReport{}
	}
	return reports
}

func runIntegrationsSync(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("integrations sync", flag.ContinueOnError)
	cwd := flags.String("cwd", ".", "directorio de partida")
	change := flags.String("change", "", "limita el sync a un cambio")
	retry := flags.Bool("retry-abandoned", false, "reactiva las operaciones abandonadas")
	var discard branchRepeatedFlag
	flags.Var(&discard, "discard", "retira una operación sin entregarla; repetible")
	asJSON := flags.Bool("json", false, "imprime el resultado como JSON")
	if code, done := parseIntegrationsFlags(flags, args, integrationsSyncHelp, stdout, stderr); done {
		return code
	}
	loc, ok := locateIntegrationsWorkspace(*cwd, stderr)
	if !ok {
		return integrationsExitFail
	}
	dispatcher, err := newIntegrationDispatcher(loc)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return integrationsExitFail
	}

	report, err := dispatcher.Sync(context.Background(), integrations.SyncOptions{Change: *change, RetryAbandoned: *retry, Discard: discard})
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return integrationsExitFail
	}

	if *asJSON {
		out := integrationsSyncJSON{Schema: integrationsSyncSchema, Changes: []integrationsSyncChangeJSON{}, Remaining: report.Remaining}
		for _, c := range report.Changes {
			out.Changes = append(out.Changes, integrationsSyncChangeJSON{
				Change: c.Change, Archived: c.Archived,
				Delivered: nonNil(c.Delivered), Pending: nonNil(c.Pending), Abandoned: nonNil(c.Abandoned), Discarded: nonNil(c.Discarded),
				Warnings: c.Warnings,
			})
		}
		writeIntegrationsJSON(stdout, out)
	} else if len(report.Changes) == 0 {
		fmt.Fprintln(stdout, "No hay colas de integración que sincronizar.")
	} else {
		for _, c := range report.Changes {
			where := "ordinaria"
			if c.Archived {
				where = "archivada"
			}
			fmt.Fprintf(stdout, "Cambio %s (%s): entregadas %d, pendientes %d, abandonadas %d, descartadas %d\n", c.Change, where, len(c.Delivered), len(c.Pending), len(c.Abandoned), len(c.Discarded))
			for _, warning := range c.Warnings {
				fmt.Fprintf(stdout, "  Aviso: %s\n", warning)
			}
		}
	}

	if !report.Remaining {
		return integrationsExitOK
	}
	for _, c := range report.Changes {
		for _, op := range append(append([]integrations.OperationReport(nil), c.Pending...), c.Abandoned...) {
			detail := op.LastError
			if op.Reason != "" {
				detail = op.Reason
			}
			fmt.Fprintf(stderr, "Queda %s (%s, plugin %s, intentos %d)%s\n", op.Key, op.State, op.Plugin, op.Attempts, errorSuffix(detail))
		}
		if c.Skipped {
			fmt.Fprintf(stderr, "Se ha saltado la cola de %s: la guarda de ramas la ha rechazado.\n", c.Change)
		}
	}
	return integrationsExitFail
}

// ---------------------------------------------------------------- emit

type integrationsEmitJSON struct {
	Schema      string                         `json:"schema"`
	Change      string                         `json:"change"`
	Event       string                         `json:"event"`
	Key         string                         `json:"key,omitempty"`
	Delivered   []integrations.OperationReport `json:"delivered"`
	Pending     []integrations.OperationReport `json:"pending"`
	Abandoned   []integrations.OperationReport `json:"abandoned"`
	NothingToDo bool                           `json:"nothing_to_do"`
}

func runIntegrationsEmit(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("integrations emit", flag.ContinueOnError)
	cwd := flags.String("cwd", ".", "directorio de partida")
	change := flags.String("change", "", "cambio")
	eventName := flags.String("event", "", "evento del catálogo")
	role := flags.String("role", "", "rol del roster sellado")
	actorFlag := flags.String("actor", "", "actor")
	asJSON := flags.Bool("json", false, "imprime el resultado como JSON")
	if code, done := parseIntegrationsFlags(flags, args, integrationsEmitHelp, stdout, stderr); done {
		return code
	}
	if strings.TrimSpace(*change) == "" || strings.TrimSpace(*eventName) == "" {
		fmt.Fprintln(stderr, "Error: --change y --event son obligatorios. Ejecuta `axiom integrations emit --help`.")
		return integrationsExitUsage
	}
	if *change == "." || *change == ".." || strings.ContainsAny(*change, `/\`) {
		fmt.Fprintf(stderr, "Error: el nombre de cambio %q no es un nombre de carpeta simple.\n", *change)
		return integrationsExitUsage
	}
	loc, ok := locateIntegrationsWorkspace(*cwd, stderr)
	if !ok {
		return integrationsExitFail
	}

	ev, root, err := buildManualEvent(loc, *change, *eventName, strings.ToLower(strings.TrimSpace(*role)), *actorFlag)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return integrationsExitFail
	}

	out := integrationsEmitJSON{Schema: integrationsEmitSchema, Change: *change, Event: string(ev.Type), Key: ev.IdempotencyKey,
		Delivered: []integrations.OperationReport{}, Pending: []integrations.OperationReport{}, Abandoned: []integrations.OperationReport{}}
	if !integrations.HasListeners(loc.Declarations.Integrations, ev.Type) {
		out.NothingToDo = true
		if *asJSON {
			writeIntegrationsJSON(stdout, out)
		} else {
			fmt.Fprintf(stdout, "Ningún plugin activo escucha el evento %s; no hay nada que emitir.\n", ev.Type)
		}
		return integrationsExitOK
	}

	dispatcher, err := newIntegrationDispatcher(loc)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return integrationsExitFail
	}
	report := dispatcher.Dispatch(context.Background(), root, ev, integrations.DispatchOptions{Durable: true, PersistOutcome: integrations.PolicyFor(ev.Type).PersistOutcome})
	if report.Err != nil {
		fmt.Fprintf(stderr, "Error: no se ha podido emitir el evento %s: %v\n", ev.Type, report.Err)
		return integrationsExitFail
	}
	out.Delivered, out.Pending, out.Abandoned = nonNil(report.Delivered), nonNil(report.Pending), nonNil(report.Abandoned)

	if *asJSON {
		writeIntegrationsJSON(stdout, out)
	} else {
		fmt.Fprintf(stdout, "Evento %s del cambio %s (clave %s): entregadas %d, pendientes %d, abandonadas %d\n", ev.Type, *change, ev.IdempotencyKey, len(report.Delivered), len(report.Pending), len(report.Abandoned))
		for _, warning := range report.Warnings {
			fmt.Fprintf(stdout, "  Aviso: %s\n", warning)
		}
	}
	if undelivered := report.Undelivered(); len(undelivered) > 0 {
		for _, op := range undelivered {
			detail := op.LastError
			if op.Reason != "" {
				detail = op.Reason
			}
			fmt.Fprintf(stderr, "Queda %s (%s, plugin %s, intentos %d)%s; cola: %s\n", op.Key, op.State, op.Plugin, op.Attempts, errorSuffix(detail), report.QueuePath)
		}
		return integrationsExitFail
	}
	return integrationsExitOK
}

// buildManualEvent valida y construye el evento de `emit`. Devuelve el evento y la carpeta donde se
// encola. Los eventos de compuerta no se fabrican: toman digest, huella y actor del último registro
// approved de gates.yaml.
func buildManualEvent(loc workspace.Location, change, eventName, role, actorFlag string) (integrations.Event, string, error) {
	ev, err := integrations.NewEvent(eventName, change)
	if err != nil {
		return integrations.Event{}, "", err
	}
	if ev.Type == manifest.EventChangeStart {
		return integrations.Event{}, "", errors.New("change.start no se emite a mano en esta versión: lo emitirá C-07-d (`axiom change start`)")
	}
	policy := integrations.PolicyFor(ev.Type)

	ordinary := filepath.Join(loc.PlanningHome, "changes", change)
	archived, hasArchived := kickoff.ArchivedChangeRoot(loc.WorkspaceRoot, change)
	var root string
	switch {
	case ev.Type == manifest.EventArchive:
		if !hasArchived {
			return integrations.Event{}, "", fmt.Errorf("archive exige la carpeta archivada del cambio %q y no existe ninguna bajo openspec/changes/archive/; comprueba el estado con `axiom sdd status --change %s`", change, change)
		}
		root = archived
	case isDirectory(ordinary):
		root = ordinary
	case hasArchived:
		return integrations.Event{}, "", fmt.Errorf("el cambio %q está archivado: solo se puede emitir archive; consulta su estado con `axiom sdd status --change %s`", change, change)
	default:
		return integrations.Event{}, "", fmt.Errorf("el cambio %q no existe en %q; ejecuta `axiom sdd status` para ver los cambios activos", change, ordinary)
	}

	if policy.NeedsRole {
		if role == "" {
			return integrations.Event{}, "", fmt.Errorf("el evento %s exige --role con un rol del roster sellado; ejecuta `axiom integrations emit --help`", ev.Type)
		}
		sealed, err := kickoff.Load(root)
		if err != nil {
			return integrations.Event{}, "", err
		}
		if sealed == nil {
			return integrations.Event{}, "", fmt.Errorf("el cambio %q no tiene un kickoff sellado con roster de roles; sella el kickoff con `axiom sdd kickoff seal --change %s --execution-style checkpointed`", change, change)
		}
		roster := rosterFromKickoff(*sealed)
		if !gateKeyInRoster(kickoff.RoleApplyGate(role), roster) {
			return integrations.Event{}, "", fmt.Errorf("el rol %q no pertenece al roster sellado de %q (roster: %s); repite con --role de uno de ellos o consulta el estado con `axiom sdd status --change %s`", role, change, strings.Join(rosterRoleNames(roster), ", "), change)
		}
	}

	actor := actorFlag
	if actor == "" && !policy.NeedsApprovedGate {
		actor, _ = defaultActor(loc.WorkspaceRoot, gateActorTarget)
	}

	switch {
	case policy.NeedsApprovedGate:
		gate := kickoff.GateSpec
		switch ev.Type {
		case manifest.EventGateDesignApproved:
			gate = kickoff.GateDesign
		case manifest.EventRoleApplyApproved:
			gate = kickoff.RoleApplyGate(role)
		}
		record, err := lastGateRecord(root, gate)
		if err != nil {
			return integrations.Event{}, "", err
		}
		built := gateIntegrationEvent(root, ev.Type, role, record)
		if actorFlag != "" {
			built.Actor = actorFlag
		}
		return built, root, nil
	case ev.Type == manifest.EventRoleStart:
		ev.Role, ev.Actor, ev.OccurredAt = role, actor, nowUTC()
		ev.IdempotencyKey = integrations.RoleStartKey(change, role)
	case ev.Type == manifest.EventArchive:
		ev.Actor, ev.OccurredAt = actor, nowUTC()
		ev.IdempotencyKey = integrations.ArchiveKey(change, filepath.Base(root))
	}
	return ev, root, nil
}

// lastGateRecord devuelve la última decisión de la compuerta en gates.yaml; debe ser approved.
func lastGateRecord(changeRoot string, gate kickoff.GateKey) (kickoff.GateRecord, error) {
	ledger, err := kickoff.LoadGates(changeRoot)
	if err != nil {
		return kickoff.GateRecord{}, err
	}
	var last *kickoff.GateRecord
	for i := range ledger.Records {
		if ledger.Records[i].Gate == gate {
			last = &ledger.Records[i]
		}
	}
	if last == nil {
		return kickoff.GateRecord{}, fmt.Errorf("la compuerta %s no tiene ninguna decisión en gates.yaml: emit no fabrica aprobaciones; regístrala con `axiom sdd gate record`", gate)
	}
	if last.Decision != kickoff.DecisionApproved {
		return kickoff.GateRecord{}, fmt.Errorf("la última decisión de la compuerta %s en gates.yaml es %s, no approved: emit no fabrica aprobaciones: regístrala con `axiom sdd gate record --change %s --gate %s --decision approved`", gate, last.Decision, filepath.Base(changeRoot), gate)
	}
	return *last, nil
}

// ---------------------------------------------------------------- secret

type integrationsSecretStatusEntry struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

type integrationsSecretStatusJSON struct {
	Schema  string                          `json:"schema"`
	Plugin  string                          `json:"plugin"`
	Secrets []integrationsSecretStatusEntry `json:"secrets"`
}

func runIntegrationsSecret(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, integrationsSecretHelp)
		return integrationsExitUsage
	}
	switch subcommand, rest := args[0], args[1:]; subcommand {
	case "--help", "-h", "help":
		fmt.Fprint(stdout, integrationsSecretHelp)
		return integrationsExitOK
	case "set":
		return runIntegrationsSecretSet(rest, stdin, stdout, stderr)
	case "status":
		return runIntegrationsSecretStatus(rest, stdout, stderr)
	case "delete":
		return runIntegrationsSecretDelete(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Error: subcomando '%s' no reconocido para integrations secret; los disponibles son set, status y delete.\n", subcommand)
		fmt.Fprint(stdout, integrationsSecretHelp)
		return integrationsExitUsage
	}
}

// resolveSecretTarget comprueba que el plugin está en el catálogo cerrado y, si hay nombre, que el
// manifiesto lo declara; si no, error tipado con la lista de los declarados.
func resolveSecretTarget(plugin, name string, requireName bool) (manifest.Manifest, error) {
	m, ok := manifest.Lookup(plugin)
	if !ok {
		list := "ninguno en esta versión"
		if catalog := manifest.Catalog(); len(catalog) > 0 {
			ids := make([]string, len(catalog))
			for i, c := range catalog {
				ids[i] = c.ID
			}
			list = strings.Join(ids, ", ")
		}
		return manifest.Manifest{}, fmt.Errorf("plugin desconocido %q; plugins admitidos: %s; los plugins configurados salen de axiom.yaml (`axiom workspace validate`)", plugin, list)
	}
	if name == "" {
		if requireName {
			return manifest.Manifest{}, errors.New("--name es obligatorio: ejecuta `axiom integrations secret --help`")
		}
		return m, nil
	}
	declared := make([]string, len(m.Secrets))
	for i, s := range m.Secrets {
		if s.Name == name {
			return m, nil
		}
		declared[i] = s.Name
	}
	list := "ninguno"
	if len(declared) > 0 {
		list = strings.Join(declared, ", ")
	}
	return manifest.Manifest{}, fmt.Errorf("el plugin %s no declara el secreto %q; secretos declarados: %s; consúltalos con `axiom integrations secret status --plugin %s`", plugin, name, list, plugin)
}

func secretFlags(name string) (*flag.FlagSet, *string, *string) {
	flags := flag.NewFlagSet("integrations secret "+name, flag.ContinueOnError)
	flags.String("cwd", ".", "directorio de partida (se admite por uniformidad; el secreto no depende del proyecto)")
	plugin := flags.String("plugin", "", "identificador del plugin")
	secretName := flags.String("name", "", "nombre del secreto")
	return flags, plugin, secretName
}

func runIntegrationsSecretSet(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags, plugin, name := secretFlags("set")
	if code, done := parseIntegrationsFlags(flags, args, integrationsSecretSetHelp, stdout, stderr); done {
		return code
	}
	if *plugin == "" || *name == "" {
		fmt.Fprintln(stderr, "Error: --plugin y --name son obligatorios. Ejecuta `axiom integrations secret set --help`.")
		return integrationsExitUsage
	}
	if _, err := resolveSecretTarget(*plugin, *name, true); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return integrationsExitFail
	}
	if stdinIsTerminal(stdin) {
		fmt.Fprintf(stderr, "Error: stdin es un terminal y el valor nunca se pide por argumento. Canaliza el valor, por ejemplo: Read-Host -MaskInput | axiom integrations secret set --plugin %s --name %s   (o: printf '%%s' \"$VALOR\" | axiom integrations secret set ...)\n", *plugin, *name)
		return integrationsExitFail
	}
	data, err := io.ReadAll(io.LimitReader(stdin, maxSecretBytes+1))
	if err != nil {
		fmt.Fprintf(stderr, "Error: no se pudo leer stdin: %v\n", err)
		return integrationsExitFail
	}
	if len(data) > maxSecretBytes {
		fmt.Fprintf(stderr, "Error: el valor supera el máximo de %d bytes.\n", maxSecretBytes)
		return integrationsExitFail
	}
	value := strings.TrimSuffix(strings.TrimSuffix(string(data), "\n"), "\r")
	if value == "" {
		fmt.Fprintln(stderr, "Error: el valor está vacío; no se guarda nada.")
		return integrationsExitFail
	}
	ref := secrets.Ref{Plugin: *plugin, Name: *name}
	if err := secrets.Default().Set(context.Background(), ref, secrets.NewSecret(value)); err != nil {
		fmt.Fprintf(stderr, "Error: no se pudo guardar el secreto %s: %v\n", ref, err)
		return integrationsExitFail
	}
	fmt.Fprintf(stdout, "Secreto %s guardado en el almacén del sistema.\n", ref)
	return integrationsExitOK
}

func runIntegrationsSecretStatus(args []string, stdout, stderr io.Writer) int {
	flags, plugin, name := secretFlags("status")
	asJSON := flags.Bool("json", false, "imprime el resultado como JSON")
	if code, done := parseIntegrationsFlags(flags, args, integrationsSecretStatusHelp, stdout, stderr); done {
		return code
	}
	if *plugin == "" {
		fmt.Fprintln(stderr, "Error: --plugin es obligatorio. Ejecuta `axiom integrations secret status --help`.")
		return integrationsExitUsage
	}
	m, err := resolveSecretTarget(*plugin, *name, false)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return integrationsExitFail
	}
	out := integrationsSecretStatusJSON{Schema: integrationsSecretStatusSchema, Plugin: m.ID, Secrets: []integrationsSecretStatusEntry{}}
	store := secrets.Default()
	for _, spec := range m.Secrets {
		if *name != "" && spec.Name != *name {
			continue
		}
		_, source, _ := store.Get(context.Background(), secrets.Ref{Plugin: m.ID, Name: spec.Name})
		out.Secrets = append(out.Secrets, integrationsSecretStatusEntry{Name: spec.Name, Source: string(source)})
	}
	if *asJSON {
		writeIntegrationsJSON(stdout, out)
		return integrationsExitOK
	}
	if len(out.Secrets) == 0 {
		fmt.Fprintf(stdout, "El plugin %s no declara secretos.\n", m.ID)
	}
	for _, entry := range out.Secrets {
		fmt.Fprintf(stdout, "%s/%s: %s\n", m.ID, entry.Name, entry.Source)
	}
	return integrationsExitOK
}

func runIntegrationsSecretDelete(args []string, stdout, stderr io.Writer) int {
	flags, plugin, name := secretFlags("delete")
	if code, done := parseIntegrationsFlags(flags, args, integrationsSecretDeleteHelp, stdout, stderr); done {
		return code
	}
	if *plugin == "" || *name == "" {
		fmt.Fprintln(stderr, "Error: --plugin y --name son obligatorios. Ejecuta `axiom integrations secret delete --help`.")
		return integrationsExitUsage
	}
	if _, err := resolveSecretTarget(*plugin, *name, true); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return integrationsExitFail
	}
	ref := secrets.Ref{Plugin: *plugin, Name: *name}
	if err := secrets.Default().Delete(context.Background(), ref); err != nil {
		fmt.Fprintf(stderr, "Error: no se pudo borrar el secreto %s: %v\n", ref, err)
		return integrationsExitFail
	}
	fmt.Fprintf(stdout, "Secreto %s borrado del almacén del sistema (si existía). Una variable de entorno no se borra desde Axiom: si %s está definida, sigue activa.\n", ref, ref.EnvVar())
	return integrationsExitOK
}
