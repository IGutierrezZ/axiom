package cli

import (
	"bytes"
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

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
)

// `axiom autofetch` (C-05-d, RF-HUB-13, RF-BRA-39, RF-BRA-40, design D-03, D-09 y D-10): el verbo
// propio del autofetch del repositorio de specs. Es un verbo de primer nivel, no un subcomando
// de `axiom branch` (RF-BRA-01: esa familia conserva sus cinco verbos). `enable`, `disable` y
// `status` editan y leen la preferencia personal del hub; `run` ejecuta el ciclo del motor y, con
// `--agent`, es el hook de sesión de Claude Code y Codex: nunca falla, termina siempre con 0 y
// solo escribe cuando hay avisos.

const autofetchHelp = `Uso: axiom autofetch <subcomando> [banderas]

Subcomandos:
  enable    Enciende el autofetch del proyecto (preferencia personal del hub)
  disable   Apaga el autofetch del proyecto
  status    Informa del estado efectivo del interruptor (solo lectura, sin red)
  run       Trae origin del repositorio de specs y avisa de lo relevante

El autofetch es una preferencia personal del hub (~/.axiom/workspaces.json), nunca de
axiom.yaml, y está apagado por defecto. Encenderlo es la autorización explícita y
persistente del fetch de origin en ese proyecto.

Banderas comunes:
  --cwd <ruta>       Ruta absoluta de partida (por defecto, el directorio de trabajo del proceso)
  --agent <runtime>  claude-code o codex: modo hook de run; en los demás subcomandos se acepta y no cambia nada
  --json             Imprime el resultado como JSON

Ejecuta ` + "`axiom autofetch <subcomando> --help`" + ` para ver el detalle de cada subcomando.
`

const autofetchEnableHelp = `Uso: axiom autofetch enable [--cwd <ruta>] [--agent <runtime>] [--json]

Guarda autofetch: true en la entrada del proyecto del hub. Con el interruptor encendido,
` + "`axiom autofetch run`" + ` y ` + "`axiom branch ensure`" + ` traen origin sin que tengas que pasar --fetch. Es idempotente:
repetirla deja workspaces.json byte a byte igual. El proyecto debe estar registrado
(axiom project add "<carpeta maestra>"); si no lo está, o si el hub es ilegible, se rechaza
sin escribir nada y termina con 1. Nunca modifica axiom.yaml.

Salida --json: axiom.autofetch-preference/v1.
`

const autofetchDisableHelp = `Uso: axiom autofetch disable [--cwd <ruta>] [--agent <runtime>] [--json]

Apaga el autofetch del proyecto y retira la clave de su entrada del hub. Es idempotente. Sobre
un proyecto no registrado no hace nada: lo informa («no hay nada que apagar»), no crea el hub
y termina con 0. Con el hub ilegible se rechaza sin modificarlo y termina con 1. Nunca
modifica axiom.yaml.

Salida --json: axiom.autofetch-preference/v1.
`

const autofetchStatusHelp = `Uso: axiom autofetch status [--cwd <ruta>] [--agent <runtime>] [--json]

Informa del estado efectivo del interruptor (enabled o disabled) para el proyecto. No hace
red, no lanza git y no escribe. Con el proyecto sin registrar o con el hub ilegible informa de
disabled con un aviso y termina con 0.

Salida --json: axiom.autofetch-preference/v1.
`

const autofetchRunHelp = `Uso: axiom autofetch run [--cwd <ruta>] [--agent claude-code|codex] [--json]

Con el interruptor encendido, trae origin del repositorio de specs (acotado a unos 8 s y al
intervalo mínimo entre fetch), avanza la rama de integración solo por fast-forward estricto y
avisa únicamente de lo relevante, con la orden exacta que puedes ejecutar. Nunca hace push ni
prune. Con el interruptor apagado no hace red ni escribe y lo informa. Termina con 0 aunque el
remoto no responda; solo un error de uso termina con 2.

Sin --json imprime un texto en castellano; con --json, axiom.autofetch-result/v1.

Con --agent claude-code o --agent codex es el hook SessionStart de ese runtime: escribe
{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"…"}} solo cuando hay
avisos, no escribe nada en cualquier otro caso y termina siempre con 0. Su directorio sale de
--cwd o, si falta, del campo cwd de la carga JSON que el runtime envía por la entrada estándar
(hasta 256 KiB); si tampoco está, del directorio del proceso.
`

// Cotas de la lectura de la entrada estándar del hook: la carga del runtime es pequeña; más de
// 256 KiB se descarta y el hook sigue con el directorio del proceso (falla en abierto).
const autofetchStdinLimit = 256 << 10

// autofetchStdinWait es lo máximo que el hook espera a la carga del runtime: es una variable de
// paquete para que las pruebas la reduzcan.
var autofetchStdinWait = 2 * time.Second

// AutofetchPreferenceSchema es el esquema de la salida --json de `enable`, `disable` y `status`.
const AutofetchPreferenceSchema = "axiom.autofetch-preference/v1"

// autofetchPreference es la salida --json de `enable`, `disable` y `status` (design D-09).
type autofetchPreference struct {
	Schema     string   `json:"schema"`
	Operation  string   `json:"operation"`
	Autofetch  string   `json:"autofetch"`
	Project    string   `json:"project"`
	Registered bool     `json:"registered"`
	Changed    bool     `json:"changed"`
	Warnings   []string `json:"warnings"`
	Message    string   `json:"message"`
}

// autofetchHookOutput es la salida del modo hook: el canal de contexto adicional de
// `SessionStart` de Claude Code y Codex.
type autofetchHookOutput struct {
	HookSpecificOutput autofetchHookSpecific `json:"hookSpecificOutput"`
}

type autofetchHookSpecific struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// autofetchOptions son las banderas comunes de los cuatro subcomandos.
type autofetchOptions struct {
	cwd      string
	cwdGiven bool
	agent    string
	asJSON   bool
}

// RunAutofetch ejecuta `axiom autofetch <subcomando>` con el hub del usuario y devuelve el código
// de salida. stdin solo lo lee `run --agent` (la carga del hook).
func RunAutofetch(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return runAutofetch(args, stdin, stdout, stderr, "")
}

// runAutofetch es RunAutofetch con la ruta del hub inyectada ("" es el hub del usuario): las
// pruebas nunca tocan ~/.axiom/workspaces.json.
func runAutofetch(args []string, stdin io.Reader, stdout, stderr io.Writer, hubPath string) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, autofetchHelp)
		return branch.ExitUsage
	}
	switch subcommand, rest := args[0], args[1:]; subcommand {
	case "--help", "-h", "help":
		fmt.Fprint(stdout, autofetchHelp)
		return branch.ExitReady
	case "enable":
		return runAutofetchPreference("enable", rest, stdout, stderr, hubPath)
	case "disable":
		return runAutofetchPreference("disable", rest, stdout, stderr, hubPath)
	case "status":
		return runAutofetchPreference("status", rest, stdout, stderr, hubPath)
	case "run":
		if autofetchHookRequested(rest) {
			return runAutofetchHook(rest, stdin, stdout, hubPath)
		}
		return runAutofetchRun(rest, stdout, stderr, hubPath)
	default:
		fmt.Fprintf(stderr, "Error: subcomando '%s' no reconocido para autofetch; los disponibles son enable, disable, status y run. Ejecuta `axiom autofetch --help`.\n", subcommand)
		return branch.ExitUsage
	}
}

// autofetchHelpFor es la ayuda de un subcomando.
func autofetchHelpFor(subcommand string) string {
	switch subcommand {
	case "enable":
		return autofetchEnableHelp
	case "disable":
		return autofetchDisableHelp
	case "status":
		return autofetchStatusHelp
	}
	return autofetchRunHelp
}

// parseAutofetchFlags lee las banderas comunes de un subcomando. Una ayuda pedida devuelve
// flag.ErrHelp.
func parseAutofetchFlags(subcommand string, args []string) (autofetchOptions, error) {
	flags := flag.NewFlagSet("autofetch "+subcommand, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var options autofetchOptions
	flags.StringVar(&options.cwd, "cwd", "", "ruta absoluta de partida")
	flags.StringVar(&options.agent, "agent", "", "runtime del hook: claude-code o codex")
	flags.BoolVar(&options.asJSON, "json", false, "imprime el resultado como JSON")
	if err := flags.Parse(args); err != nil {
		return options, err
	}
	if flags.NArg() > 0 {
		return options, fmt.Errorf("argumento inesperado '%s'", flags.Arg(0))
	}
	flags.Visit(func(given *flag.Flag) {
		if given.Name == "cwd" {
			options.cwdGiven = true
		}
	})
	return options, nil
}

// resolveAutofetchCWD devuelve el directorio de partida: --cwd, que debe ser absoluta (una ruta
// relativa es un error de uso), o el directorio del proceso.
func resolveAutofetchCWD(options autofetchOptions) (string, error) {
	if options.cwdGiven {
		if strings.TrimSpace(options.cwd) == "" || !filepath.IsAbs(options.cwd) {
			return "", fmt.Errorf("--cwd '%s' debe ser una ruta absoluta", options.cwd)
		}
		return options.cwd, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("no se pudo determinar el directorio de trabajo (%v): indica una ruta absoluta con --cwd", err)
	}
	return dir, nil
}

// autofetchUsage imprime un error de uso y devuelve el código 2.
func autofetchUsage(stderr io.Writer, subcommand string, err error) int {
	fmt.Fprintf(stderr, "Error: %v. Ejecuta `axiom autofetch %s --help`.\n", err, subcommand)
	return branch.ExitUsage
}

// autofetchHookRequested dice si `run` se invoca como hook: con la bandera --agent presente.
func autofetchHookRequested(args []string) bool {
	for _, arg := range args {
		name := strings.TrimLeft(arg, "-")
		if name != arg && (name == "agent" || strings.HasPrefix(name, "agent=")) {
			return true
		}
		if arg == "--" {
			return false
		}
	}
	return false
}

// runAutofetchRun ejecuta `run` a mano: texto o --json, código 0 siempre salvo un error de uso.
func runAutofetchRun(args []string, stdout, stderr io.Writer, hubPath string) int {
	options, err := parseAutofetchFlags("run", args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, autofetchRunHelp)
			return branch.ExitReady
		}
		return autofetchUsage(stderr, "run", err)
	}
	cwd, err := resolveAutofetchCWD(options)
	if err != nil {
		return autofetchUsage(stderr, "run", err)
	}
	result := engine.DefaultWithHubPath(hubPath).Autofetch(context.Background(), engine.AutofetchRequest{CWD: cwd})
	if options.asJSON {
		if err := writeIndentedJSON(stdout, result); err != nil {
			fmt.Fprintf(stderr, "Error: no se pudo escribir el resultado: %v. Repite `axiom autofetch run --json`.\n", err)
		}
		return branch.ExitReady
	}
	result.WriteText(stdout)
	return branch.ExitReady
}

// runAutofetchHook es `run --agent`: el hook de sesión. Falla en abierto y termina siempre con 0:
// un error de uso, una carga ilegible o un pánico acaban en silencio. Solo escribe cuando hay
// avisos que dar; con el interruptor apagado, el proyecto sin registrar, el hub ilegible o fuera
// de un proyecto Axiom no escribe nada (RF-BRA-40).
func runAutofetchHook(args []string, stdin io.Reader, stdout io.Writer, hubPath string) (code int) {
	defer func() {
		if recover() != nil {
			code = branch.ExitReady
		}
	}()
	options, err := parseAutofetchFlags("run", args)
	if err != nil || !validAutofetchAgent(options.agent) {
		return branch.ExitReady
	}
	var cwd string
	if options.cwdGiven {
		if cwd, err = resolveAutofetchCWD(options); err != nil {
			return branch.ExitReady
		}
	} else if cwd = autofetchStdinCWD(stdin); cwd == "" {
		if cwd, err = os.Getwd(); err != nil {
			return branch.ExitReady
		}
	}
	result := engine.DefaultWithHubPath(hubPath).Autofetch(context.Background(), engine.AutofetchRequest{CWD: cwd})
	if result.Autofetch != branch.AutofetchStateEnabled || len(result.Notices) == 0 {
		return branch.ExitReady
	}
	var text strings.Builder
	text.WriteString("Axiom autofetch:\n")
	for _, notice := range result.Notices {
		fmt.Fprintf(&text, "- %s\n", notice.Message)
		if notice.Command != "" {
			fmt.Fprintf(&text, "  Orden: %s\n", notice.Command)
		}
	}
	_ = writeIndentedJSON(stdout, autofetchHookOutput{HookSpecificOutput: autofetchHookSpecific{
		HookEventName:     "SessionStart",
		AdditionalContext: strings.TrimRight(text.String(), "\n"),
	}})
	return branch.ExitReady
}

// validAutofetchAgent dice si el valor de --agent es un runtime con hook de sesión.
func validAutofetchAgent(agent string) bool {
	switch agent {
	case "claude-code", "claude", "codex":
		return true
	}
	return false
}

// autofetchStdinCWD lee el campo cwd de la carga JSON del runtime. Falla en abierto: sin entrada,
// en una terminal, con una carga ilegible, de más de 256 KiB o que no llega a tiempo devuelve ""
// y el hook sigue con el directorio del proceso.
func autofetchStdinCWD(stdin io.Reader) string {
	if stdin == nil {
		return ""
	}
	if file, ok := stdin.(*os.File); ok {
		if info, err := file.Stat(); err != nil || info.Mode()&os.ModeCharDevice != 0 {
			return ""
		}
	}
	loaded := make(chan []byte, 1)
	go func() {
		data, _ := io.ReadAll(io.LimitReader(stdin, autofetchStdinLimit+1))
		loaded <- data
	}()
	select {
	case data := <-loaded:
		if len(data) > autofetchStdinLimit {
			return ""
		}
		var payload struct {
			CWD string `json:"cwd"`
		}
		if json.Unmarshal(data, &payload) != nil || !filepath.IsAbs(payload.CWD) {
			return ""
		}
		return payload.CWD
	case <-time.After(autofetchStdinWait):
		return ""
	}
}

// runAutofetchPreference ejecuta `enable`, `disable` y `status` (design D-03 y D-09). La
// identidad del proyecto se resuelve solo con ficheros: ningún proceso de git.
func runAutofetchPreference(operation string, args []string, stdout, stderr io.Writer, hubPath string) int {
	options, err := parseAutofetchFlags(operation, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, autofetchHelpFor(operation))
			return branch.ExitReady
		}
		return autofetchUsage(stderr, operation, err)
	}
	cwd, err := resolveAutofetchCWD(options)
	if err != nil {
		return autofetchUsage(stderr, operation, err)
	}
	eng := engine.DefaultWithHubPath(hubPath)
	project, refusal := eng.AutofetchProject(cwd)
	if refusal != nil {
		fmt.Fprintf(stderr, "Error: %s. %s.\n", strings.TrimRight(refusal.Message, ". "), strings.TrimRight(refusal.Continuation, ". "))
		return branch.ExitRefused
	}
	preference := autofetchPreference{
		Schema:    AutofetchPreferenceSchema,
		Operation: "autofetch " + operation,
		Autofetch: branch.AutofetchStateDisabled,
		Project:   project.Master,
		Warnings:  []string{},
	}
	switch operation {
	case "status":
		enabled, state, warning := eng.AutofetchPreference(project.Master)
		preference.Registered = state == branch.AutofetchEnabled || state == branch.AutofetchDisabled
		if warning != "" {
			preference.Warnings = append(preference.Warnings, warning)
		}
		if enabled {
			preference.Autofetch = branch.AutofetchStateEnabled
			preference.Message = fmt.Sprintf("Autofetch encendido para %s.", project.Master)
		} else {
			preference.Message = fmt.Sprintf("Autofetch apagado para %s.", project.Master)
		}
	default:
		if code := setAutofetchPreference(operation == "enable", project.Master, hubPath, &preference, stderr); code != branch.ExitReady {
			return code
		}
	}
	return writeAutofetchPreference(preference, options.asJSON, stdout, stderr)
}

// setAutofetchPreference aplica `enable` o `disable` en el hub y rellena la salida. Devuelve un
// código distinto de 0 cuando rechaza (proyecto sin registrar al encender, hub ilegible): en ese
// caso no ha escrito nada.
func setAutofetchPreference(enable bool, master, hubPath string, preference *autofetchPreference, stderr io.Writer) int {
	manager, err := hub.NewManager(hubPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error: no se pudo abrir el hub (%v): no se modifica nada.\n", err)
		return branch.ExitRefused
	}
	_, changed, err := manager.SetAutofetch(master, enable)
	switch {
	case errors.Is(err, hub.ErrProjectNotRegistered) && !enable:
		preference.Message = fmt.Sprintf("No hay nada que apagar: el proyecto %s no está registrado en el hub, así que el autofetch ya estaba apagado.", master)
		return branch.ExitReady
	case errors.Is(err, hub.ErrProjectNotRegistered):
		fmt.Fprintf(stderr, "Error: el proyecto %s no está registrado en el hub, así que no se puede encender el autofetch. Regístralo con `axiom project add \"%s\"` y repite `axiom autofetch enable`.\n", master, master)
		return branch.ExitRefused
	case err != nil:
		fmt.Fprintf(stderr, "Error: no se pudo leer o escribir el hub (%v): no se modifica nada. Repara o retira ese fichero y repite `axiom autofetch %s`.\n", err, map[bool]string{true: "enable", false: "disable"}[enable])
		return branch.ExitRefused
	}
	preference.Registered, preference.Changed = true, changed
	switch {
	case enable && changed:
		preference.Autofetch = branch.AutofetchStateEnabled
		preference.Message = fmt.Sprintf("Autofetch encendido para %s: es la autorización explícita y persistente del fetch de origin en este proyecto.", master)
	case enable:
		preference.Autofetch = branch.AutofetchStateEnabled
		preference.Message = fmt.Sprintf("El autofetch ya estaba encendido para %s.", master)
	case changed:
		preference.Message = fmt.Sprintf("Autofetch apagado para %s.", master)
	default:
		preference.Message = fmt.Sprintf("El autofetch ya estaba apagado para %s.", master)
	}
	return branch.ExitReady
}

// writeAutofetchPreference imprime la preferencia como JSON o como texto en castellano.
func writeAutofetchPreference(preference autofetchPreference, asJSON bool, stdout, stderr io.Writer) int {
	if asJSON {
		if err := writeIndentedJSON(stdout, preference); err != nil {
			fmt.Fprintf(stderr, "Error: no se pudo escribir el resultado: %v. Repite `axiom %s --json`.\n", err, preference.Operation)
			return branch.ExitRefused
		}
		return branch.ExitReady
	}
	fmt.Fprintln(stdout, preference.Message)
	for _, warning := range preference.Warnings {
		fmt.Fprintf(stdout, "Aviso: %s\n", warning)
	}
	return branch.ExitReady
}

// writeIndentedJSON escribe value como JSON con sangría, sin escapar `<`, `>` ni `&` (las órdenes
// de los avisos llevan comillas y rutas), y con un salto de línea final.
func writeIndentedJSON(w io.Writer, value any) error {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return err
	}
	_, err := w.Write(buffer.Bytes())
	return err
}
