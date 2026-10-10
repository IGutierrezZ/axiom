package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/specbaseline"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// RunSDDArchiveCompose composes an OpenSpec canonical spec with a delta spec
// (#4119). It replaces the sdd-archive skill's model-driven Read/Edit merge
// for the "main spec exists" case: on success the composed bytes go to
// --output (default stdout); on an unapplied delta, nothing is written and
// the error names the offending section and requirement.
//
// Two modes share the command (inc-24, REQ-24.38 and REQ-24.47). The native mode
// (--canonical --delta) is unchanged. The document mode (--cwd --document <domain>
// --delta) resolves the document and its requirement profile from the
// workspace's axiom.yaml: without a declared baseline it composes
// <planning home>/specs/<domain>/spec.md with the same native engine, and with a
// requirement profile it composes with specbaseline.ComposeDocument. The
// non-blocking warnings of the profile engine go to the standard error, and so does
// the correspondence of provisional to definitive requirement IDs (REQ-24.43,
// REQ-24.59): it is computed once for the whole change when the delta lives in
// <planning home>/changes/<change>/specs/<domain>/, and for the loose delta
// otherwise.
func RunSDDArchiveCompose(args []string, stdout io.Writer) error {
	return runSDDArchiveCompose(args, stdout)
}

func runSDDArchiveCompose(args []string, stdout io.Writer) error {
	return runSDDArchiveComposeWithStderr(args, stdout, os.Stderr)
}

// repeatedFlag is a string flag that may be given more than once.
type repeatedFlag []string

func (r *repeatedFlag) String() string { return strings.Join(*r, ",") }

func (r *repeatedFlag) Set(value string) error {
	*r = append(*r, value)
	return nil
}

func runSDDArchiveComposeWithStderr(args []string, stdout, stderr io.Writer) error {
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
	var documents repeatedFlag
	flags.Var(&documents, "document", "Dominio de la baseline sobre el que componer (modo documento); excluyente con --canonical")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected sdd-archive-compose argument %q; rerun `axiom sdd-archive-compose --canonical <path> --delta <path> [--output <path|->]` with no positional arguments", flags.Arg(0))
	}
	documentMode := len(documents) > 0
	if documentMode && strings.TrimSpace(*canonicalPath) != "" {
		return errors.New("sdd-archive-compose: --document y --canonical son excluyentes: usa --canonical --delta para componer sobre una especificación OpenSpec concreta, o --cwd --document <dominio> --delta para resolver el documento desde axiom.yaml; vuelve a ejecutar `axiom sdd-archive-compose --canonical <ruta> --delta <ruta>` o `axiom sdd-archive-compose --cwd <dir> --document <dominio> --delta <ruta>`")
	}
	if !documentMode && strings.TrimSpace(*canonicalPath) == "" {
		return errors.New("sdd-archive-compose requires --canonical or --document; rerun `axiom sdd-archive-compose --canonical openspec/specs/<domain>/spec.md --delta openspec/changes/<change>/specs/<domain>/spec.md` or `axiom sdd-archive-compose --cwd <workspace> --document <domain> --delta <path>`")
	}
	if strings.TrimSpace(*deltaPath) == "" {
		return errors.New("sdd-archive-compose requires --delta; rerun `axiom sdd-archive-compose --canonical openspec/specs/<domain>/spec.md --delta openspec/changes/<change>/specs/<domain>/spec.md`")
	}
	if len(documents) > 1 {
		return fmt.Errorf("sdd-archive-compose: --document admite un solo dominio junto con --delta (se indicaron %d: %s); ejecuta `axiom sdd-archive-compose --cwd <dir> --document <dominio> --delta <ruta>` una vez por dominio", len(documents), strings.Join(documents, ", "))
	}

	// With --cwd, and in document mode, the workspace is resolved with the
	// planning-home primitive. With --cwd, relative paths resolve against its
	// workspaceRoot (REQ-24.47); without it they keep resolving against the
	// process working directory.
	var loc workspace.Location
	resolve := func(path string) string { return path }
	if strings.TrimSpace(*cwd) != "" || documentMode {
		start := *cwd
		if strings.TrimSpace(start) == "" {
			start = "."
		}
		located, err := workspace.Locate(start)
		if err != nil {
			return err
		}
		loc = located
		if strings.TrimSpace(*cwd) != "" {
			resolve = func(path string) string {
				if path == "" || path == "-" || filepath.IsAbs(path) {
					return path
				}
				return filepath.Join(loc.WorkspaceRoot, path)
			}
		}
	}

	var canonicalBytes []byte
	if !documentMode {
		var err error
		if canonicalBytes, err = os.ReadFile(resolve(*canonicalPath)); err != nil {
			return fmt.Errorf("read canonical spec: %w", err)
		}
	}
	deltaBytes, err := os.ReadFile(resolve(*deltaPath))
	if err != nil {
		return fmt.Errorf("read delta spec: %w", err)
	}
	options := sddstatus.ComposeOptions{SupersedeRemoved: *supersede, ExplicitSuperseded: splitSupersededRequirements(*supersededReqs)}

	var composed []byte
	if documentMode {
		composition, err := composeDocumentMode(loc, documents[0], resolve(*deltaPath), deltaBytes, options)
		if err != nil {
			return err
		}
		composed = composition.Output
		for _, warning := range composition.Warnings {
			_, _ = fmt.Fprintf(stderr, "Aviso: %s\n", warning)
		}
		for _, assignment := range composition.IDs {
			_, _ = fmt.Fprintf(stderr, "Correspondencia de IDs: %s -> %s (dominio %q)\n", assignment.Provisional, assignment.Definitive, assignment.Domain)
		}
	} else {
		native, err := sddstatus.ComposeOpenSpecCanonicalSpecWithOptions(string(canonicalBytes), string(deltaBytes), options)
		if err != nil {
			return err
		}
		composed = []byte(native)
	}

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

func splitSupersededRequirements(list string) []string {
	var explicit []string
	for _, s := range strings.Split(list, ",") {
		if trimmed := strings.TrimSpace(s); trimmed != "" {
			explicit = append(explicit, trimmed)
		}
	}
	return explicit
}

// documentComposition is the result of the document mode: the composed bytes, the
// non-blocking warnings of the profile engine and the correspondence of
// provisional to definitive IDs of the change (empty without ADDED requirements
// with an `id` identity).
type documentComposition struct {
	Output   []byte
	Warnings []string
	IDs      []specbaseline.IDAssignment
}

// composeDocumentMode composes the delta over the document of one baseline
// domain (REQ-24.47, REQ-24.48). It never writes: it returns the composed bytes,
// the non-blocking warnings and the ID correspondence. deltaPath is the
// resolved path of the delta, which decides the scope of that correspondence.
func composeDocumentMode(loc workspace.Location, domain, deltaPath string, deltaBytes []byte, options sddstatus.ComposeOptions) (documentComposition, error) {
	if err := checkDocumentDomain(domain); err != nil {
		return documentComposition{}, err
	}
	declarations := loc.Declarations
	if declarations.ParseErr != nil {
		return documentComposition{}, fmt.Errorf("sdd-archive-compose: no se puede leer %s para resolver la baseline: %w; corrige el YAML y vuelve a ejecutar el comando", loc.ConfigPath, declarations.ParseErr)
	}
	if declarations.BaselineErr != nil {
		return documentComposition{}, fmt.Errorf("sdd-archive-compose: la baseline declarada en %s no es válida: %w; corrígela (`axiom workspace validate --path %s`) y vuelve a ejecutar el comando", loc.ConfigPath, declarations.BaselineErr, loc.Master)
	}

	if declarations.Baseline == nil {
		// Without a declared baseline the document is the OpenSpec one and the
		// composition is the native engine, byte for byte (REQ-24.38).
		canonicalBytes, err := os.ReadFile(filepath.Join(loc.PlanningHome, "specs", domain, "spec.md"))
		if err != nil {
			return documentComposition{}, fmt.Errorf("read canonical spec: %w", err)
		}
		composed, err := sddstatus.ComposeOpenSpecCanonicalSpecWithOptions(string(canonicalBytes), string(deltaBytes), options)
		if err != nil {
			return documentComposition{}, err
		}
		return documentComposition{Output: []byte(composed)}, nil
	}

	var declared *workspace.BaselineDocument
	for i := range declarations.Baseline.Documents {
		if declarations.Baseline.Documents[i].Domain == domain {
			declared = &declarations.Baseline.Documents[i]
			break
		}
	}
	if declared == nil {
		return documentComposition{}, fmt.Errorf("sdd-archive-compose: el dominio %q no figura en 'baseline.documents' de %s: no se integra ni se promueve, y nunca se crea %s. Continuación: declara el dominio en 'baseline.documents' de axiom.yaml (`axiom workspace validate --path %s` comprueba la declaración) o retira la delta del cambio",
			domain, loc.ConfigPath, filepath.Join(loc.PlanningHome, "specs", domain), loc.Master)
	}
	if declared.Requirement == nil {
		// refusal:by-design world-action: a prose document has no requirement profile to compose; it is integrated by directed edit or registry-only, not by this command.
		return documentComposition{}, fmt.Errorf("sdd-archive-compose: el dominio %q es de prosa (sin perfil de requisito) y no se compone: su documento se integra por edición dirigida o solo registro, según su 'integration' (%s)", domain, declared.EffectiveIntegration())
	}
	target, err := plannedBaselineDocument(loc, *declared, deltaBytes)
	if err != nil {
		return documentComposition{}, err
	}

	// The correspondence of IDs is computed once for the whole change when the
	// delta lives in its folder, and for the loose delta otherwise (REQ-24.59).
	planned := []specbaseline.PlannedDocument{target}
	if changeRoot, ok := changeRootOfDelta(loc, deltaPath, domain); ok {
		if planned, err = plannedChangeDocuments(loc, declarations.Baseline.Documents, changeRoot, target); err != nil {
			return documentComposition{}, err
		}
	}
	ids, err := specbaseline.PlanIDs(planned)
	if err != nil {
		return documentComposition{}, err
	}

	result, err := specbaseline.ComposeDocument(specbaseline.ComposeInput{
		Document:           target,
		IDs:                ids,
		Supersede:          options.SupersedeRemoved,
		ExplicitSuperseded: options.ExplicitSuperseded,
	})
	if err != nil {
		return documentComposition{}, err
	}
	return documentComposition{Output: result.Output, Warnings: result.Warnings, IDs: ids.Assignments()}, nil
}

// plannedBaselineDocument reads a baseline document with a requirement profile
// and pairs it with the bytes of its delta.
func plannedBaselineDocument(loc workspace.Location, declared workspace.BaselineDocument, deltaBytes []byte) (specbaseline.PlannedDocument, error) {
	profile, err := specbaseline.CompileProfile(declared)
	if err != nil {
		return specbaseline.PlannedDocument{}, fmt.Errorf("sdd-archive-compose: %w", err)
	}
	documentPath := filepath.Join(loc.WorkspaceRoot, filepath.FromSlash(declared.Path))
	original, err := os.ReadFile(documentPath)
	if err != nil {
		return specbaseline.PlannedDocument{}, fmt.Errorf("read baseline document of domain %q: %w", declared.Domain, err)
	}
	return specbaseline.PlannedDocument{
		Domain:   declared.Domain,
		Path:     documentPath,
		RelPath:  filepath.ToSlash(filepath.Clean(filepath.FromSlash(declared.Path))),
		Profile:  profile,
		Original: original,
		Delta:    deltaBytes,
	}, nil
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

// plannedChangeDocuments gathers, in the order of `baseline.documents`, the
// documents with a requirement profile that the change has a delta for: the
// target one with the delta already read, and the others with
// <change>/specs/<domain>/spec.md. Domains without a delta do not take part.
func plannedChangeDocuments(loc workspace.Location, documents []workspace.BaselineDocument, changeRoot string, target specbaseline.PlannedDocument) ([]specbaseline.PlannedDocument, error) {
	var planned []specbaseline.PlannedDocument
	for _, declared := range documents {
		if declared.Domain == target.Domain {
			planned = append(planned, target)
			continue
		}
		if declared.Requirement == nil {
			continue
		}
		if err := checkDocumentDomain(declared.Domain); err != nil {
			return nil, err
		}
		deltaBytes, err := os.ReadFile(filepath.Join(changeRoot, "specs", declared.Domain, "spec.md"))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read delta spec of domain %q: %w", declared.Domain, err)
		}
		other, err := plannedBaselineDocument(loc, declared, deltaBytes)
		if err != nil {
			return nil, err
		}
		planned = append(planned, other)
	}
	return planned, nil
}

// checkDocumentDomain rejects a --document value that cannot be a domain name:
// without a declared baseline it becomes a path segment of
// <planning home>/specs/<domain>.
func checkDocumentDomain(domain string) error {
	if strings.TrimSpace(domain) != domain || domain == "" || domain == "." || domain == ".." || strings.ContainsAny(domain, `/\:`) {
		return fmt.Errorf("sdd-archive-compose: --document %q no es un nombre de dominio válido; indica el dominio de 'baseline.documents' (o el directorio de <planning home>/specs/<dominio>), sin rutas, y vuelve a ejecutar `axiom sdd-archive-compose --cwd <dir> --document <dominio> --delta <ruta>`", domain)
	}
	return nil
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
	_, _ = fmt.Fprintln(stdout, "Usage: axiom sdd-archive-compose --canonical <path> --delta <path> [--output <path|->] [--supersede] [--superseded-requirements <req1,req2>]")
	_, _ = fmt.Fprintln(stdout, "       axiom sdd-archive-compose --cwd <dir> --document <domain> --delta <path> [--output <path|->] [--supersede] [--superseded-requirements <req1,req2>]")
	_, _ = fmt.Fprintln(stdout, "Merges an OpenSpec delta spec into a canonical spec ("+sddstatus.OpenSpecComposeSchema+").")
	_, _ = fmt.Fprintln(stdout, "Modo documento (--document): resuelve el documento y su perfil de requisito desde la baseline de axiom.yaml del workspace;")
	_, _ = fmt.Fprintln(stdout, "sin baseline declarada compone <planning home>/specs/<dominio>/spec.md con el motor nativo. Es excluyente con --canonical.")
	_, _ = fmt.Fprintln(stdout, "  --cwd                       Directorio desde el que se resuelve el workspaceRoot; las rutas relativas se resuelven contra él.")
	_, _ = fmt.Fprintln(stdout, "  --document                  Dominio de la baseline (un solo dominio junto con --delta).")
	_, _ = fmt.Fprintln(stdout, "                              Los ADDED con identidad id declaran su ubicación con (Placement: ...) y reciben su ID definitivo;")
	_, _ = fmt.Fprintln(stdout, "                              la correspondencia de IDs provisionales a definitivos se imprime por la salida de error.")
	_, _ = fmt.Fprintln(stdout, "Opciones de Poda Progresiva (ODD-4.1):")
	_, _ = fmt.Fprintln(stdout, "  --supersede                 Marca los requisitos de REMOVED con [SUPERSEDED / DEPRECADO] en vez de eliminarlos.")
	_, _ = fmt.Fprintln(stdout, "  --superseded-requirements   Lista de requisitos adicionales separados por coma a marcar como superados.")
	_, _ = fmt.Fprintln(stdout, "On an unapplied delta, writes nothing and fails naming the section and requirement.")
	return nil
}
