package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests/marks"
	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/sddpublish"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// errPublishRefused es lo que devuelve RunSDDPublish cuando la publicación se rechaza: el resultado ya
// está escrito (texto o JSON) y el error lleva la línea resumen para que `axiom sdd` salga con 1.
var errPublishRefused = errors.New("publicación rechazada") // refusal:by-design world-action: the typed refusal (code, label, message and continuation) is already written to the output; this error only carries the exit code 1

const sddPublishHelp = `Uso: axiom sdd publish --cwd <ruta> --change <nombre> [--archive] [--branch <rama>] [--json]

Publica los documentos de un cambio SDD: confirma solo las rutas de documentación del cambio
(carpeta del cambio, specs, índice y las rutas de baseline y de contexto mantenido declaradas en
axiom.yaml) y sube solo la rama del cambio a origin, sin forzar y sin saltarse hooks. No sube código,
no abre PR y no hace fetch: si hay commits de código sin subir, los informa y los sube la persona.

  --cwd <ruta>       directorio de partida (por defecto, el actual)
  --change <nombre>  cambio que se publica (obligatorio)
  --archive          publica el archivado del cambio: obligatorio en la rama de archivado
  --branch <rama>    ata la publicación a la rama que autorizó la persona
  --json             imprime el resultado como axiom.sdd-publish/v1, también cuando se rechaza

Salida 0 con published y up_to_date, y 1 con una negativa tipada.
`

// RunSDDPublish es el punto de entrada de `axiom sdd publish` (DD-10): un adaptador de E/S delgado
// sobre sddpublish.Publisher, el único código del producto que confirma y sube (RF-PUB-07). Es el
// único fichero de este paquete que importa sddpublish.
func RunSDDPublish(args []string, stdout io.Writer) error {
	if hasGovernanceHelpFlag(args) {
		_, _ = fmt.Fprint(stdout, sddPublishHelp)
		return nil
	}
	flags := flag.NewFlagSet("sdd publish", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	cwd := flags.String("cwd", ".", "directorio de partida")
	change := flags.String("change", "", "nombre del cambio")
	archive := flags.Bool("archive", false, "publica el archivado del cambio")
	branchName := flags.String("branch", "", "rama autorizada")
	asJSON := flags.Bool("json", false, "imprime el resultado como axiom.sdd-publish/v1")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("%v; ejecuta `axiom sdd publish --help`", err)
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("argumento inesperado %q para sdd publish; ejecuta `axiom sdd publish --help`", flags.Arg(0))
	}
	if strings.TrimSpace(*change) == "" {
		return errors.New("falta --change: indica el cambio que se publica, por ejemplo `axiom sdd publish --change <nombre>`; ejecuta `axiom sdd publish --help`")
	}
	if err := gitref.ValidBranchComponent(*change); err != nil {
		return fmt.Errorf("--change %q no es un nombre de cambio válido: %v; usa letras, dígitos, «.», «_» y «-», por ejemplo `axiom sdd publish --change <nombre>`; ejecuta `axiom sdd publish --help`", *change, err)
	}
	blank := false
	flags.Visit(func(given *flag.Flag) {
		if given.Name == "branch" && strings.TrimSpace(*branchName) == "" {
			blank = true
		}
	})
	if blank {
		return errors.New("--branch no puede estar vacío: indica la rama autorizada con `axiom sdd publish --change <nombre> --branch <rama>` o quita la bandera; ejecuta `axiom sdd publish --help`")
	}

	dir, err := filepath.Abs(*cwd)
	if err != nil {
		return fmt.Errorf("no se pudo resolver --cwd %q (%v): indica una ruta absoluta con `axiom sdd publish --change <nombre> --cwd <ruta>`", *cwd, err)
	}
	loc, err := workspace.Locate(dir)
	if err != nil {
		return err
	}

	result, err := publishChangeAt(context.Background(), loc, *change, *archive, *branchName, func(text string) {
		_, _ = fmt.Fprintln(writerBranchWarnings, "Aviso: "+text)
	})
	if err != nil {
		return err
	}

	if *asJSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(stdout, string(data))
	} else {
		writePublishText(stdout, result)
	}
	// Tras un archivado publicado, el evento archive se encola de forma duradera en la carpeta
	// archivada y se intenta de inmediato; nunca cambia lo que imprime ni el código de salida.
	if *archive && result.Status != sddpublish.StatusRefused {
		if archivedRoot, found := kickoff.ArchivedChangeRoot(loc.WorkspaceRoot, *change); found {
			emitArchiveEvent(loc, archivedRoot)
		}
	}
	if result.Status == sddpublish.StatusRefused {
		return fmt.Errorf("%w: %s (%s)", errPublishRefused, result.Refusal.Label, result.Refusal.Code)
	}
	return nil
}

// publishChangeAt es el cuerpo compartido de `axiom sdd publish` y de PublishChange (design D6):
// aplica la guarda A (RF-BRA-20, publicar escribe en la carpeta del cambio, ordinaria o archivada;
// los avisos de la guarda van a warn) y publica con sddpublish. Una negativa de la guarda vuelve
// como error, sin publicar nada; las de sddpublish viajan tipadas en el resultado.
func publishChangeAt(ctx context.Context, loc workspace.Location, change string, archive bool, branchName string, warn func(string)) (sddpublish.Result, error) {
	writerContext := branch.ContextOrdinary
	if archive {
		writerContext = branch.ContextArchive
	}
	if changeRoot, found := publishChangeRoot(loc, change); found {
		if err := guardChangeWriterCollect(changeRoot, change, "sdd publish", writerContext, warn); err != nil {
			// Sin `--archive`, la rama de archivado es la equivocada para el contexto ordinario: lo que
			// falta es el flag, y la negativa lo nombra.
			if !archive && guardChangeWriterCollect(changeRoot, change, "sdd publish", branch.ContextArchive, warn) == nil {
				return sddpublish.Result{}, fmt.Errorf("%w; la rama actual es la de archivado: repite la orden con `--archive`", err)
			}
			return sddpublish.Result{}, err
		}
	}
	publisher := &sddpublish.Publisher{Git: sddpublish.NewGit(), Targets: engine.Default()}
	return publisher.Run(ctx, sddpublish.Request{
		CWD: loc.WorkspaceRoot, Change: change, Archive: archive, Branch: branchName,
	}), nil
}

// ChangePublishRequest es la petición del dashboard para publicar los documentos de un cambio tras
// aprobar una puerta (C-06-f): CWD es la raíz del espacio de trabajo y Branch la rama que la persona
// vio en pantalla, que siempre se ata.
type ChangePublishRequest struct{ CWD, Change, Branch string }

// ChangePublishRefusal es una negativa tipada de sddpublish, con su código, etiqueta, mensaje y
// continuación.
type ChangePublishRefusal struct {
	Code, Label, Message, Continuation string
	Details                            map[string]any
}

// ChangePublishResult es el resultado de publicar un cambio. Status es `published`, `up_to_date` o
// `refused`; con Refusal, Status es `refused`. Commit es el SHA del commit de documentos y PushedSHA
// el SHA subido. Notices lleva los avisos de la guarda.
type ChangePublishResult struct {
	Status, Remote, Branch, GitRoot, Commit, PushedSHA string
	UncommittedDocs                                    []string
	Resume                                             []string
	Refusal                                            *ChangePublishRefusal
	Notices                                            []GateNotice
}

// PublishChange publica los documentos del cambio en su contexto ordinario con la misma guarda A y el
// mismo sddpublish que `axiom sdd publish`: sin `Only` ni `Replay`, sin fetch y sin forzar, y con la
// rama siempre atada. La negativa de la guarda vuelve como *branch.RefusalError en el error.
func PublishChange(ctx context.Context, req ChangePublishRequest) (ChangePublishResult, error) {
	loc, err := workspace.Locate(req.CWD)
	if err != nil {
		return ChangePublishResult{}, err
	}
	var notices []GateNotice
	result, err := publishChangeAt(ctx, loc, req.Change, false, req.Branch, func(text string) {
		notices = append(notices, GateNotice{Source: "branch_guard", Text: text})
	})
	if err != nil {
		return ChangePublishResult{Notices: notices}, err
	}
	out := ChangePublishResult{
		Status: string(result.Status), Remote: result.Remote, Branch: result.Branch,
		UncommittedDocs: result.UncommittedDocs, Resume: result.Resume, Notices: notices,
	}
	if result.Repository != nil {
		out.GitRoot = result.Repository.GitRoot
	}
	if result.Commit != nil {
		out.Commit = result.Commit.SHA
	}
	if result.Push != nil {
		out.PushedSHA = result.Push.SHA
	}
	if refusal := result.Refusal; refusal != nil {
		out.Refusal = &ChangePublishRefusal{
			Code: string(refusal.Code), Label: refusal.Label, Message: refusal.Message,
			Continuation: refusal.Continuation, Details: refusal.Details,
		}
	}
	return out, nil
}

// publishChangeRoot es la carpeta del cambio sobre la que se aplica la guarda: la del planning
// home o, si ya se movió, la archivada. Falso si no existe ninguna: la publicación lo rechaza con
// su propia negativa tipada.
func publishChangeRoot(loc workspace.Location, change string) (string, bool) {
	changeRoot := filepath.Join(loc.PlanningHome, "changes", change)
	if isDirectory(changeRoot) {
		return changeRoot, true
	}
	return kickoff.ArchivedChangeRoot(loc.WorkspaceRoot, change)
}

// writePublishText escribe el resultado en texto, en castellano.
func writePublishText(stdout io.Writer, result sddpublish.Result) {
	subject := result.Change
	if result.Context == sddpublish.ContextArchive {
		subject = "el archivado de " + result.Change
	}
	target := ""
	if result.Branch != "" {
		target = fmt.Sprintf(" en %s/%s", result.Remote, result.Branch)
	}
	repository := ""
	if result.Repository != nil {
		repository = fmt.Sprintf(" (repositorio %s)", result.Repository.Name)
	}

	switch result.Status {
	case sddpublish.StatusRefused:
		_, _ = fmt.Fprintf(stdout, "Publicación de %s rechazada: %s (%s)\n", subject, result.Refusal.Label, result.Refusal.Code)
		_, _ = fmt.Fprintf(stdout, "  %s\n", result.Refusal.Message)
		_, _ = fmt.Fprintf(stdout, "  Continuación: %s\n", result.Refusal.Continuation)
		if result.Commit != nil {
			_, _ = fmt.Fprintf(stdout, "  El commit %s se conserva en local: %s (%d rutas)\n", publishShortSHA(result.Commit.SHA), result.Commit.Message, len(result.Commit.Paths))
		}
		return
	case sddpublish.StatusUpToDate:
		_, _ = fmt.Fprintf(stdout, "Publicación de %s%s%s: nada que publicar, la rama ya está al día.\n", subject, target, repository)
	default:
		_, _ = fmt.Fprintf(stdout, "Publicación de %s%s%s:\n", subject, target, repository)
		if result.Commit != nil {
			_, _ = fmt.Fprintf(stdout, "  Commit %s: %s (%d rutas)\n", publishShortSHA(result.Commit.SHA), result.Commit.Message, len(result.Commit.Paths))
		}
		if result.Push != nil {
			created := ""
			if result.Push.Created {
				created = " (rama creada)"
			}
			_, _ = fmt.Fprintf(stdout, "  Push: %s -> %s%s\n", result.Push.Refspec, result.Remote, created)
		}
	}
	_, _ = fmt.Fprintf(stdout, "Quien siga: `%s` y `%s`.\n", result.Resume[0], result.Resume[1])
}

func publishShortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// MarksPublishRequest es la petición del dashboard para publicar las marcas de pruebas de un
// cambio (C-06-d): CWD es el workspaceRoot, Change el cambio y Branch la rama que la persona vio
// en la pantalla (vacía si no se ata).
type MarksPublishRequest struct{ CWD, Change, Branch string }

// MarksPublishRefusal es una negativa tipada de la publicación de marcas, con el código, la
// etiqueta, el mensaje y la continuación que da sddpublish.
type MarksPublishRefusal struct {
	Code         string
	Label        string
	Message      string
	Continuation string
	Details      map[string]any
}

// MarksPublishResult es el resultado de publicar las marcas. Status es `published`, `up_to_date`
// o `refused`; con Refusal, Status es `refused`. Commit es el SHA del commit de marcas y
// PushedSHA el SHA subido; GitRoot es el repositorio donde se publicó.
type MarksPublishResult struct {
	Status, Remote, Branch, GitRoot, Commit, PushedSHA string
	Replayed                                           bool
	UncommittedDocs                                    []string
	Refusal                                            *MarksPublishRefusal
}

// MarksPublisher es el publicador de marcas que el dashboard recibe por inyección (C-06-d,
// prov:WEB:3): el dashboard no ejecuta git ni importa sddpublish. Publica solo
// `pruebas-estado.yaml` con la publicación acotada de sddpublish y, ante `push_non_fast_forward`,
// su reproducción única con marks.Merge (prov:PUB:1, prov:PUB:2). Vive en este fichero para que el
// verbo siga siendo el único importador de sddpublish (RF-PUB-07).
type MarksPublisher struct {
	publisher *sddpublish.Publisher
}

// NewMarksPublisher crea el publicador con el puerto git real y el motor de ramas.
func NewMarksPublisher() *MarksPublisher {
	return &MarksPublisher{publisher: &sddpublish.Publisher{Git: sddpublish.NewGit(), Targets: engine.Default()}}
}

// MarksIdentity devuelve el autor `Nombre <email>` de la configuración git de dir o, si falta
// alguna parte, las claves de git que faltan.
func (*MarksPublisher) MarksIdentity(ctx context.Context, dir string) (string, []string) {
	identity := resolveActorIdentity(ctx, dir)
	if actor, ok := identity.Actor(); ok {
		return actor, nil
	}
	return "", identity.Missing()
}

// PublishMarks confirma y sube solo `pruebas-estado.yaml` del cambio. Toda negativa de la
// publicación viaja tipada en Refusal, con su código, etiqueta y continuación.
func (m *MarksPublisher) PublishMarks(ctx context.Context, req MarksPublishRequest) MarksPublishResult {
	result := m.publisher.Run(ctx, sddpublish.Request{
		CWD: req.CWD, Change: req.Change, Branch: req.Branch,
		Only: []string{marks.FileName}, Replay: marks.Merge,
	})
	out := MarksPublishResult{
		Status: string(result.Status), Remote: result.Remote, Branch: result.Branch,
		Replayed: result.Replayed, UncommittedDocs: result.UncommittedDocs,
	}
	if result.Repository != nil {
		out.GitRoot = result.Repository.GitRoot
	}
	if result.Commit != nil {
		out.Commit = result.Commit.SHA
	}
	if result.Push != nil {
		out.PushedSHA = result.Push.SHA
	}
	if refusal := result.Refusal; refusal != nil {
		out.Refusal = &MarksPublishRefusal{
			Code: string(refusal.Code), Label: refusal.Label, Message: refusal.Message,
			Continuation: refusal.Continuation, Details: refusal.Details,
		}
	}
	return out
}
