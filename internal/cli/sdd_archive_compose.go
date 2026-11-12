package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// RunSDDArchiveCompose composes an OpenSpec canonical spec with a delta spec
// (#4119). It replaces the sdd-archive skill's model-driven Read/Edit merge
// for the "main spec exists" case: on success the composed bytes go to
// --output (default stdout); on an unapplied delta, nothing is written and
// the error names the offending section and requirement.
//
// The only mode is the native one (--canonical --delta, REQ-30.22): it is
// byte for byte the composition that every accepted delta always produced. The
// document and change modes (--document, --change, --in-place) belonged to the
// foreign `baseline:` and were retired with it: their use is rejected with a
// *ComposeModeRetiredError that names the continuation. When the composed spec
// goes to a file, the workspace of that destination is resolved and a residual
// `baseline:` stops the command without writing (*workspace.BaselineRetiredError,
// REQ-30.21 d).
func RunSDDArchiveCompose(args []string, stdout io.Writer) error {
	return runSDDArchiveCompose(args, stdout)
}

func runSDDArchiveCompose(args []string, stdout io.Writer) error {
	return runSDDArchiveComposeWithStderr(args, stdout, os.Stderr)
}

// ComposeModeRetiredError is the typed rejection of the retired modes of
// `axiom sdd archive-compose` (`--document`, `--change`, `--in-place`).
type ComposeModeRetiredError struct {
	// Flags are the retired options that were given, in a stable order.
	Flags []string
}

// Error names the retired options and the continuation.
func (e *ComposeModeRetiredError) Error() string {
	return fmt.Sprintf("sdd-archive-compose: %s ya no existen: pertenecían a la baseline ajena, que se retiró. La composición es `axiom sdd archive-compose --canonical <ruta> --delta <ruta> [--output <ruta|->]`; para el contexto técnico usa `axiom archive context prepare|verify` y para migrar una baseline `axiom archive adopt scan --path <ruta> --change <cambio>`",
		strings.Join(e.Flags, ", "))
}

func runSDDArchiveComposeWithStderr(args []string, stdout, _ io.Writer) error {
	if hasSDDArchiveComposeHelp(args) {
		return renderSDDArchiveComposeHelp(stdout)
	}
	flags := flag.NewFlagSet("sdd-archive-compose", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	canonicalPath := flags.String("canonical", "", "Path to the existing canonical openspec/specs/<domain>/spec.md")
	deltaPath := flags.String("delta", "", "Path to the change's delta openspec/changes/<change>/specs/<domain>/spec.md")
	outputPath := flags.String("output", "-", "Where to write the composed canonical spec; use - for stdout")
	supersede := flags.Bool("supersede", false, "Marca los requisitos en REMOVED con [SUPERSEDED / DEPRECADO] en lugar de borrarlos físicamente")
	supersededReqs := flags.String("superseded-requirements", "", "Lista de requisitos adicionales separados por comas para marcar como [SUPERSEDED / DEPRECADO]")
	cwd := flags.String("cwd", "", "Directorio desde el que resolver el workspaceRoot; con él, las rutas relativas se resuelven contra el workspaceRoot")
	// Retired modes: declared only to recognise them and reject them with a typed error.
	flags.String("document", "", "Retirado")
	flags.String("change", "", "Retirado")
	flags.Bool("in-place", false, "Retirado")
	if err := flags.Parse(args); err != nil {
		return err
	}
	var retired []string
	flags.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "document", "change", "in-place":
			retired = append(retired, "--"+f.Name)
		}
	})
	if len(retired) > 0 {
		return &ComposeModeRetiredError{Flags: retired}
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected sdd-archive-compose argument %q; rerun `axiom sdd-archive-compose --canonical <path> --delta <path> [--output <path|->]` with no positional arguments", flags.Arg(0))
	}
	if strings.TrimSpace(*canonicalPath) == "" {
		return errors.New("sdd-archive-compose requires --canonical; rerun `axiom sdd-archive-compose --canonical openspec/specs/<domain>/spec.md --delta openspec/changes/<change>/specs/<domain>/spec.md`")
	}
	if strings.TrimSpace(*deltaPath) == "" {
		return errors.New("sdd-archive-compose requires --delta; rerun `axiom sdd-archive-compose --canonical openspec/specs/<domain>/spec.md --delta openspec/changes/<change>/specs/<domain>/spec.md`")
	}

	// With --cwd the workspace is resolved with the planning-home primitive and
	// relative paths resolve against its workspaceRoot (REQ-24.47); without it they
	// keep resolving against the process working directory.
	var loc workspace.Location
	resolve := func(path string) string { return path }
	if strings.TrimSpace(*cwd) != "" {
		located, err := workspace.Locate(*cwd)
		if err != nil {
			return err
		}
		loc = located
		resolve = func(path string) string {
			if path == "" || path == "-" || filepath.IsAbs(path) {
				return path
			}
			return filepath.Join(loc.WorkspaceRoot, path)
		}
	}

	// Guarda A (REQ-25.19): componer un delta de `changes/<c>/` a un fichero solo se hace en la
	// rama de archivado. La previsualización por la salida estándar no escribe y no se guarda.
	if output := resolve(*outputPath); output != "-" && strings.TrimSpace(output) != "" {
		// REQ-30.21 d: un `baseline:` residual en el workspace del destino detiene
		// la escritura de la spec viva.
		if err := refuseRetiredBaselineAtDestination(output); err != nil {
			return err
		}
		if err := guardComposeLooseDelta(resolve(*deltaPath)); err != nil {
			return err
		}
	}

	// REQ-29.10 y REQ-29.11 (D13): un cambio ISO que no cumple la regla del archivado no se
	// compone; se niega antes de leer la canónica y sin escribir.
	if err := refuseISOCompose(loc, resolve(*deltaPath), nil); err != nil {
		return err
	}

	canonicalBytes, err := os.ReadFile(resolve(*canonicalPath))
	if err != nil {
		return fmt.Errorf("read canonical spec: %w", err)
	}
	deltaBytes, err := os.ReadFile(resolve(*deltaPath))
	if err != nil {
		return fmt.Errorf("read delta spec: %w", err)
	}
	options := sddstatus.ComposeOptions{SupersedeRemoved: *supersede, ExplicitSuperseded: splitSupersededRequirements(*supersededReqs)}
	native, err := sddstatus.ComposeOpenSpecCanonicalSpecWithOptions(string(canonicalBytes), string(deltaBytes), options)
	if err != nil {
		return err
	}
	composed := []byte(native)

	output := resolve(*outputPath)
	if output == "-" || strings.TrimSpace(output) == "" {
		_, err := stdout.Write(composed)
		return err
	}
	if err := os.WriteFile(output, composed, 0o600); err != nil {
		return fmt.Errorf("write composed canonical spec: %w", err)
	}
	return nil
}

// refuseRetiredBaselineAtDestination resolves the workspace of the output file
// and rejects the write with the typed retirement error when its axiom.yaml
// still declares `baseline:` (REQ-30.21 d). An output outside any workspace is
// not a spec of a project and is not guarded.
func refuseRetiredBaselineAtDestination(output string) error {
	absolute, err := filepath.Abs(output)
	if err != nil {
		return nil
	}
	loc, err := workspace.Locate(filepath.Dir(absolute))
	if err != nil {
		return nil
	}
	return loc.GuardSpecsWrite("sdd archive-compose")
}

func splitSupersededRequirements(list string) []string {
	var explicit []string
	for _, s := range strings.Split(list, ",") {
		if trimmed := strings.TrimSpace(s); trimmed != "" {
			explicit = append(explicit, trimmed)
		}
	}
	return explicit
}

// changeRootOfDelta tells whether the delta lives in
// <planning home>/changes/<change>/specs/<domain>/ and returns the change folder.
func changeRootOfDelta(loc workspace.Location, deltaPath, domain string) (string, bool) {
	absolute, err := filepath.Abs(deltaPath)
	if err != nil {
		return "", false
	}
	changes := filepath.Join(loc.PlanningHome, "changes")
	if resolved, err := filepath.EvalSymlinks(absolute); err == nil {
		absolute = resolved
	}
	if resolved, err := filepath.EvalSymlinks(changes); err == nil {
		changes = resolved
	}
	relative, err := filepath.Rel(changes, absolute)
	if err != nil {
		return "", false
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) < 4 || parts[0] == ".." || parts[1] != "specs" || parts[2] != domain {
		return "", false
	}
	return filepath.Join(changes, parts[0]), true
}

func hasSDDArchiveComposeHelp(args []string) bool {
	for _, argument := range args {
		if argument == "--help" || argument == "-h" {
			return true
		}
	}
	return false
}

func renderSDDArchiveComposeHelp(stdout io.Writer) error {
	_, _ = fmt.Fprintln(stdout, "Usage: axiom sdd-archive-compose --canonical <path> --delta <path> [--cwd <dir>] [--output <path|->] [--supersede] [--superseded-requirements <req1,req2>]")
	_, _ = fmt.Fprintln(stdout, "Merges an OpenSpec delta spec into a canonical spec ("+sddstatus.OpenSpecComposeSchema+").")
	_, _ = fmt.Fprintln(stdout, "  --cwd                       Directorio desde el que se resuelve el workspaceRoot; las rutas relativas se resuelven contra él.")
	_, _ = fmt.Fprintln(stdout, "Los modos --document, --change e --in-place pertenecían a la baseline ajena, retirada: se rechazan con un motivo tipado.")
	_, _ = fmt.Fprintln(stdout, "Con salida a fichero, un `baseline:` residual en el axiom.yaml del destino detiene la composición sin escribir.")
	_, _ = fmt.Fprintln(stdout, "Opciones de Poda Progresiva (ODD-4.1):")
	_, _ = fmt.Fprintln(stdout, "  --supersede                 Marca los requisitos de REMOVED con [SUPERSEDED / DEPRECADO] en vez de eliminarlos.")
	_, _ = fmt.Fprintln(stdout, "  --superseded-requirements   Lista de requisitos adicionales separados por coma a marcar como superados.")
	_, _ = fmt.Fprintln(stdout, "On an unapplied delta, writes nothing and fails naming the section and requirement.")
	return nil
}
