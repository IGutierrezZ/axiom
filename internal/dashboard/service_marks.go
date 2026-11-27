package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests"
	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests/marks"
	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/fileprim"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Marcas de ejecución de pruebas.md por entorno (C-06-d, design D5 a D14). Este fichero no ejecuta
// git ni importa sddpublish (prov:WEB:3): la identidad y la publicación pasan por MarksPublisher.

// Códigos de rechazo propios del dashboard (los de validación son los de marks y los de la
// publicación los de sddpublish, reenviados tal cual).
const (
	codeRequestNotLocal      = "request_not_local"
	codeMarksContentType     = "marks_content_type"
	codeMarksBodyInvalid     = "marks_body_invalid"
	codeMarksConfirmMissing  = "marks_confirm_missing"
	codeIdentityMissing      = "identity_missing"
	codeMarksFileInvalid     = "marks_file_invalid"
	codeMarksBatteryUnread   = "marks_battery_unreadable"
	codeMarksBusy            = "marks_busy"
	codeMarksPublisherAbsent = "marks_publisher_missing"
	codeMarksTopology        = "marks_topology_unsupported"
	codeMarksNotSDD          = "marks_not_sdd"
	codeMarksArchived        = "marks_change_archived"
	codeMarksNotActive       = "marks_not_active"
	codeMarksBatteryMissing  = "marks_battery_missing"
	codeMarksWorkspace       = "marks_workspace_unreadable"
	codeChangeNotFound       = "change_not_found"
)

const (
	marksStatusRefused     = "refused"
	marksStatusUnavailable = "unavailable"
	marksRemote            = "origin"
)

// MarksRefusal es un rechazo tipado de las rutas de marcas. Todo rechazo lleva código, mensaje en
// castellano y continuación.
type MarksRefusal struct {
	HTTPStatus   int
	Status       string // "refused" o "unavailable"
	Code         string
	Label        string
	Message      string
	Continuation string
	Details      map[string]any
}

// body es el cuerpo JSON de la respuesta de rechazo.
func (r *MarksRefusal) body() map[string]any {
	out := map[string]any{"status": r.Status, "code": r.Code, "error": r.Message, "continuation": r.Continuation}
	if r.Label != "" {
		out["label"] = r.Label
	}
	if len(r.Details) > 0 {
		out["details"] = r.Details
	}
	return out
}

func (r *MarksRefusal) dto() *MarksRefusalDTO {
	return &MarksRefusalDTO{Code: r.Code, Error: r.Message, Continuation: r.Continuation}
}

func marksRefuse(httpStatus int, code, message, continuation string) *MarksRefusal {
	return &MarksRefusal{HTTPStatus: httpStatus, Status: marksStatusRefused, Code: code, Message: message, Continuation: continuation}
}

func marksUnavailable(code, message, continuation string) *MarksRefusal {
	return &MarksRefusal{HTTPStatus: http.StatusConflict, Status: marksStatusUnavailable, Code: code, Message: message, Continuation: continuation}
}

// MarkRequest es el cuerpo de `POST /api/increments/marks`.
type MarkRequest struct {
	Change      string `json:"change"`
	Case        string `json:"case"`
	Environment string `json:"environment"`
	Result      string `json:"result"`
	Commit      string `json:"commit,omitempty"`
	Note        string `json:"note,omitempty"`
}

// PublishMarksRequest es el cuerpo de `POST /api/increments/marks/publish`: Branch es la rama que
// la pantalla mostró, de modo que un cambio de rama entre la vista y el clic da `wrong_branch`.
type PublishMarksRequest struct {
	Change  string `json:"change"`
	Confirm bool   `json:"confirm"`
	Branch  string `json:"branch"`
}

// MarkRecorded es la respuesta de una marca escrita.
type MarkRecorded struct {
	Entry MarkEntryDTO        `json:"entry"`
	Marks *AcceptanceMarksDTO `json:"marks"`
}

// MarksPublished es la respuesta de la publicación: `published`, `up_to_date` o
// `nothing_to_publish`.
type MarksPublished struct {
	Status          string              `json:"status"`
	Remote          string              `json:"remote"`
	Branch          string              `json:"branch"`
	Commit          string              `json:"commit,omitempty"`
	Pushed          string              `json:"pushed,omitempty"`
	Replayed        bool                `json:"replayed"`
	UncommittedDocs []string            `json:"uncommittedDocs"`
	Marks           *AcceptanceMarksDTO `json:"marks"`
}

// marksClockState es el reloj de las marcas: `at` en UTC con milisegundos y nunca repetido ni
// retrocedido dentro del proceso (D3). now es inyectable en pruebas.
type marksClockState struct {
	mu   sync.Mutex
	now  func() time.Time
	last time.Time
}

func (c *marksClockState) next() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now
	if c.now != nil {
		now = c.now
	}
	t := now().UTC().Truncate(time.Millisecond)
	if !c.last.IsZero() && !t.After(c.last) {
		t = c.last.Add(time.Millisecond)
	}
	c.last = t
	return t.Format(marks.AtLayout)
}

// marksLock es el cerrojo de un cambio compartido por marcar y publicar (D7).
type marksLock struct {
	mu         sync.Mutex
	publishing atomic.Bool
}

func (s *Service) marksLockFor(changeRoot string) *marksLock {
	lock, _ := s.marksLocks.LoadOrStore(changeRoot, &marksLock{})
	return lock.(*marksLock)
}

func marksBusy() *MarksRefusal {
	return marksRefuse(http.StatusConflict, codeMarksBusy,
		"hay otra marca o una publicación de marcas en curso sobre este cambio",
		"espera a que termine y repite")
}

// validMarksChange rechaza todo nombre que no sea un único componente de ruta, antes de componer
// ninguna ruta con él.
func validMarksChange(change string) bool {
	if change == "" || change == "." || change == ".." || change != strings.TrimSpace(change) {
		return false
	}
	if strings.ContainsAny(change, "/\\:\x00") || filepath.IsAbs(change) || filepath.VolumeName(change) != "" {
		return false
	}
	return filepath.Base(change) == change
}

func marksBodyInvalid(message string) *MarksRefusal {
	return jsonBodyInvalid(codeMarksBodyInvalid, message)
}

// jsonBodyInvalid es el rechazo 400 de un cuerpo inválido con el código de su familia de rutas.
func jsonBodyInvalid(code, message string) *MarksRefusal {
	return marksRefuse(http.StatusBadRequest, code, message,
		`envía un JSON con los campos documentados y "change" con el nombre de un cambio, sin rutas`)
}

// marksContext es lo que resuelve la disponibilidad: el workspace, la carpeta del cambio en el
// repositorio de specs in situ (D6) y el estado de la batería.
type marksContext struct {
	loc        workspace.Location
	change     string
	changeRoot string
	tests      sddstatus.AcceptanceTestsResult
}

// marksAvailability localiza el workspace y evalúa la disponibilidad (D14).
func (s *Service) marksAvailability(change string) (marksContext, *MarksRefusal) {
	loc, err := workspace.Locate(s.getRootPath())
	if err != nil {
		return marksContext{}, marksUnavailable(codeMarksWorkspace,
			"no se puede localizar el workspace: "+err.Error(),
			"corrígelo con `axiom workspace validate --path "+pathquote.Quote(s.getRootPath())+"`")
	}
	return s.marksAvailabilityAt(loc, change)
}

// marksAvailabilityAt evalúa D14 con el motivo de topología antes que `marks_publisher_missing`:
// topología, publicador, tipo de cambio (ODD, archivado), función activa y batería.
func (s *Service) marksAvailabilityAt(loc workspace.Location, change string) (marksContext, *MarksRefusal) {
	if !validMarksChange(change) {
		return marksContext{}, marksBodyInvalid("el nombre del cambio no es válido")
	}
	if topology, _ := loc.GoverningTopology(); topology != workspace.TopologyMultirepo {
		return marksContext{}, marksUnavailable(codeMarksTopology,
			"las marcas de pruebas solo se admiten en la topología multirepo",
			"declara `workspace.topology: multirepo` con un repositorio de specs propio en axiom.yaml")
	}
	if s.marksPublisher() == nil {
		return marksContext{}, marksUnavailable(codeMarksPublisherAbsent,
			"este dashboard no tiene publicador de marcas",
			"arranca el dashboard con `axiom dashboard` desde el binario de Axiom")
	}
	changeRoot := filepath.Join(loc.PlanningHome, "changes", change)
	if !dirExists(changeRoot) {
		return marksContext{}, s.marksMissingChange(loc, change)
	}
	tests := sddstatus.AcceptanceTestsFor(loc, changeRoot)
	if !tests.Active {
		message := "la batería de pruebas de aceptación no está activa en este proyecto"
		if tests.DeclarationError != "" {
			message += ": " + tests.DeclarationError
		}
		return marksContext{}, marksUnavailable(codeMarksNotActive, message,
			"activa `acceptance_tests.enabled: true` con sus entornos en axiom.yaml del repositorio de specs")
	}
	if !tests.Present {
		return marksContext{}, marksUnavailable(codeMarksBatteryMissing,
			"el cambio no tiene pruebas.md",
			"genera pruebas.md con la fase de tareas del cambio")
	}
	return marksContext{loc: loc, change: change, changeRoot: changeRoot, tests: tests}, nil
}

// missingKind es el motivo por el que un cambio no está en el repositorio de specs in situ.
type missingKind int

const (
	missingAbsent   missingKind = iota // no existe (404)
	missingODD                         // es una feature ODD
	missingArchived                    // está archivado
)

// classifyMissingChange explica por qué el cambio no está activo: es una feature ODD, está
// archivado o no existe.
func (s *Service) classifyMissingChange(loc workspace.Location, change string) missingKind {
	odd, _ := oddIncrements(loc)
	for _, feature := range odd {
		if feature.Name == change {
			return missingODD
		}
	}
	if found, err := s.locateIncrement(change); err == nil && found.kind == "archived" {
		return missingArchived
	}
	return missingAbsent
}

// changeNotFound es el 404 de un cambio que no existe, común a marcas y puertas.
func changeNotFound(change string) *MarksRefusal {
	return marksRefuse(http.StatusNotFound, codeChangeNotFound,
		"el cambio «"+change+"» no existe en el repositorio de specs de este proyecto",
		"comprueba el nombre con `axiom sdd status` o prepáralo con `axiom branch ensure --change "+change+"`")
}

// marksMissingChange proyecta classifyMissingChange con los códigos de marcas.
func (s *Service) marksMissingChange(loc workspace.Location, change string) *MarksRefusal {
	switch s.classifyMissingChange(loc, change) {
	case missingODD:
		return marksUnavailable(codeMarksNotSDD,
			"«"+change+"» es una feature ODD, no un cambio SDD con pruebas.md",
			"las marcas solo existen en cambios SDD de openspec/changes")
	case missingArchived:
		return marksUnavailable(codeMarksArchived,
			"el cambio «"+change+"» está archivado y ya no admite marcas",
			"las marcas se registran antes de archivar el cambio")
	}
	return changeNotFound(change)
}

// marksSnapshot es la lectura única de las marcas de un cambio: la de sddstatus
// (AcceptanceMarksFor), más lo que esa lectura no calcula cuando todavía no hay fichero.
type marksSnapshot struct {
	res     sddstatus.AcceptanceMarksResult
	entries []marks.Entry
	digest  string
	summary marks.Summary
	// errCode y errMsg describen un fichero o una batería ilegibles.
	errCode, errMsg string
}

func (s *Service) marksSnapshot(mc marksContext) marksSnapshot {
	var snap marksSnapshot
	if mc.tests.Error != "" {
		snap.errCode, snap.errMsg = codeMarksBatteryUnread, mc.tests.Error
		return snap
	}
	snap.res = sddstatus.AcceptanceMarksFor(context.Background(), mc.loc, mc.changeRoot, mc.change, mc.tests)
	if snap.res.Error != "" {
		snap.errCode, snap.errMsg = codeMarksFileInvalid, snap.res.Error
		return snap
	}
	snap.digest = snap.res.Digest
	if snap.digest == "" {
		digest, err := kickoff.ArtifactDigest([]string{filepath.Join(mc.changeRoot, acceptancetests.FileName)})
		if err != nil {
			snap.errCode, snap.errMsg = codeMarksBatteryUnread, err.Error()
			return snap
		}
		snap.digest = digest
	}
	snap.entries = snap.res.File.Entries
	if snap.res.Applicable {
		snap.summary = snap.res.Summary
	} else {
		snap.summary = marks.Summarize(mc.tests.Report.Battery, mc.tests.Environments, nil, snap.digest)
	}
	return snap
}

func marksEntryDTO(e marks.Entry) MarkEntryDTO {
	return MarkEntryDTO{Case: e.Case, Environment: e.Environment, Result: string(e.Result), By: e.By,
		At: e.At, Commit: e.Commit, PlanDigest: e.PlanDigest, Note: e.Note}
}

func marksEntryPtr(e *marks.Entry) *MarkEntryDTO {
	if e == nil {
		return nil
	}
	dto := marksEntryDTO(*e)
	return &dto
}

// marksDTO es el DTO de un cambio disponible.
func (s *Service) marksDTO(mc marksContext) *AcceptanceMarksDTO {
	dto := &AcceptanceMarksDTO{
		Available: true, Environments: mc.tests.Environments, Pending: []MarkEntryDTO{},
		NoteMax: marks.MaxNoteRunes, CommitMax: marks.MaxCommitRunes,
	}
	snap := s.marksSnapshot(mc)
	if snap.errMsg != "" {
		dto.Error = snap.errMsg
		return dto
	}
	dto.Summary = snap.summary.Environments
	dto.SummaryText = snap.summary.Text()
	dto.Orphans, dto.Invalid = snap.summary.Orphans, snap.summary.Invalid
	for _, cell := range marks.Cells(mc.tests.Report.Battery, mc.tests.Environments, snap.entries, snap.digest) {
		dto.Cells = append(dto.Cells, MarkCellDTO{Case: cell.Case, Environment: cell.Environment, State: cell.State,
			Current: marksEntryPtr(cell.Current), Stale: marksEntryPtr(cell.Stale)})
	}
	switch {
	case snap.res.PendingKnown:
		dto.Remote, dto.Branch = snap.res.Remote, snap.res.Branch
		for _, e := range snap.res.Pending {
			dto.Pending = append(dto.Pending, marksEntryDTO(e))
		}
	case snap.res.PendingRefusal != nil:
		refusal := snap.res.PendingRefusal
		dto.Remote, dto.Branch = marksRemote, refusal.Details["branch"]
		dto.PendingUnavailable = &MarksRefusalDTO{Code: string(refusal.Code), Error: refusal.Message, Continuation: refusal.Continuation}
	}
	return dto
}

// incrementAcceptanceMarks es la parte de marcas del DTO del detalle: disponible con su matriz o no
// disponible con el motivo tipado.
func (s *Service) incrementAcceptanceMarks(loc workspace.Location, change string) *AcceptanceMarksDTO {
	mc, refusal := s.marksAvailabilityAt(loc, change)
	if refusal != nil {
		return &AcceptanceMarksDTO{Unavailable: refusal.dto(), Pending: []MarkEntryDTO{},
			NoteMax: marks.MaxNoteRunes, CommitMax: marks.MaxCommitRunes}
	}
	return s.marksDTO(mc)
}

// RecordMark valida y añade una marca a `pruebas-estado.yaml` del cambio (D5, D6, D7). Solo añade:
// el contenido previo del fichero es prefijo exacto del nuevo.
func (s *Service) RecordMark(ctx context.Context, req MarkRequest) (*MarkRecorded, *MarksRefusal) {
	if !validMarksChange(req.Change) {
		return nil, marksBodyInvalid("el nombre del cambio no es válido")
	}
	mc, refusal := s.marksAvailability(req.Change)
	if refusal != nil {
		return nil, refusal
	}
	lock := s.marksLockFor(mc.changeRoot)
	if lock.publishing.Load() {
		return nil, marksBusy()
	}
	lock.mu.Lock()
	defer lock.mu.Unlock()

	snap := s.marksSnapshot(mc)
	if snap.errMsg != "" {
		return nil, marksFileRefusal(snap.errCode, snap.errMsg, mc.changeRoot)
	}
	entry, rejection := marks.ValidateInput(marks.Input{Case: req.Case, Environment: req.Environment,
		Result: req.Result, Commit: req.Commit, Note: req.Note}, mc.tests.Report.Battery, mc.tests.Environments)
	if rejection != nil {
		return nil, marksRefuse(http.StatusUnprocessableEntity, rejection.Code, rejection.Message, rejection.Continuation)
	}
	actor, missing := s.marksPublisher().MarksIdentity(ctx, mc.changeRoot)
	if actor == "" {
		return nil, marksIdentityRefusal(missing, mc.changeRoot)
	}
	entry.By, entry.At, entry.PlanDigest = actor, s.marksClock.next(), snap.digest

	dest := filepath.Join(mc.changeRoot, marks.FileName)
	current, err := os.ReadFile(dest)
	if err != nil && !os.IsNotExist(err) {
		return nil, marksFileRefusal(codeMarksFileInvalid, "no se puede leer "+dest+": "+err.Error(), mc.changeRoot)
	}
	updated, err := marks.Append(current, entry)
	if err != nil {
		return nil, marksFileRefusal(codeMarksFileInvalid, err.Error(), mc.changeRoot)
	}
	if err := writeFileAtomic(mc.changeRoot, dest, updated); err != nil {
		return nil, marksRefuse(http.StatusInternalServerError, "marks_write_failed",
			"no se pudo escribir "+dest+": "+err.Error(),
			"comprueba los permisos de la carpeta del cambio y repite la marca")
	}
	return &MarkRecorded{Entry: marksEntryDTO(entry), Marks: s.marksDTO(mc)}, nil
}

func marksFileRefusal(code, message, changeRoot string) *MarksRefusal {
	if code == codeMarksBatteryUnread {
		return marksRefuse(http.StatusConflict, code, message, "comprueba que pruebas.md se puede leer")
	}
	return marksRefuse(http.StatusConflict, code, message,
		"corrige "+marks.FileName+" o recupéralo con `git -C "+pathquote.Quote(changeRoot)+" checkout -- "+marks.FileName+"`")
}

func marksIdentityRefusal(missing []string, changeRoot string) *MarksRefusal {
	if len(missing) == 0 {
		missing = []string{"user.name", "user.email"}
	}
	orders := make([]string, 0, len(missing))
	for _, key := range missing {
		orders = append(orders, fmt.Sprintf("`git -C %s config %s \"<valor>\"`", pathquote.Quote(changeRoot), key))
	}
	return marksRefuse(http.StatusConflict, codeIdentityMissing,
		"falta la identidad de git ("+strings.Join(missing, " y ")+") en el repositorio de specs; no se ha escrito nada",
		"fíjala con "+strings.Join(orders, " y ")+" y repite la marca")
}

// writeFileAtomic escribe data en dest con un temporal con punto en dir y un reemplazo atómico.
func writeFileAtomic(dir, dest string, data []byte) error {
	tmp, err := os.CreateTemp(dir, ".pruebas-estado-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	cleanup := func(cause error) error {
		_ = tmp.Close()
		_ = os.Remove(name)
		return cause
	}
	if _, err := tmp.Write(data); err != nil {
		return cleanup(err)
	}
	if err := tmp.Sync(); err != nil {
		return cleanup(err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := fileprim.ReplaceAtomic(name, dest); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}

// PublishMarks publica las marcas pendientes del cambio con el MarksPublisher (D3, D7, D13). La
// confirmación se comprueba antes de leer nada de git; sin pendientes responde
// `nothing_to_publish` sin llamar al publicador.
func (s *Service) PublishMarks(ctx context.Context, req PublishMarksRequest) (*MarksPublished, *MarksRefusal) {
	if !validMarksChange(req.Change) {
		return nil, marksBodyInvalid("el nombre del cambio no es válido")
	}
	if !req.Confirm {
		return nil, marksRefuse(http.StatusUnprocessableEntity, codeMarksConfirmMissing,
			"la publicación de marcas exige confirmación explícita; no se ha tocado nada",
			"confirma la publicación en el diálogo de la pestaña Pruebas")
	}
	mc, refusal := s.marksAvailability(req.Change)
	if refusal != nil {
		return nil, refusal
	}
	lock := s.marksLockFor(mc.changeRoot)
	if !lock.mu.TryLock() {
		return nil, marksBusy()
	}
	lock.publishing.Store(true)
	defer func() {
		lock.publishing.Store(false)
		lock.mu.Unlock()
	}()

	snap := s.marksSnapshot(mc)
	if snap.errMsg != "" {
		return nil, marksFileRefusal(snap.errCode, snap.errMsg, mc.changeRoot)
	}
	if len(snap.entries) == 0 {
		return &MarksPublished{Status: "nothing_to_publish", Remote: marksRemote, UncommittedDocs: []string{}, Marks: s.marksDTO(mc)}, nil
	}
	if snap.res.PendingKnown && len(snap.res.Pending) == 0 {
		return &MarksPublished{Status: "nothing_to_publish", Remote: snap.res.Remote, Branch: snap.res.Branch,
			UncommittedDocs: []string{}, Marks: s.marksDTO(mc)}, nil
	}
	if refusal := snap.res.PendingRefusal; refusal != nil {
		return nil, &MarksRefusal{HTTPStatus: http.StatusConflict, Status: marksStatusRefused, Code: string(refusal.Code),
			Message: refusal.Message, Continuation: refusal.Continuation}
	}

	result := s.marksPublisher().PublishMarks(ctx, cli.MarksPublishRequest{CWD: mc.loc.WorkspaceRoot, Change: req.Change, Branch: req.Branch})
	if result.Refusal != nil {
		return nil, &MarksRefusal{HTTPStatus: http.StatusConflict, Status: marksStatusRefused, Code: result.Refusal.Code,
			Label: result.Refusal.Label, Message: result.Refusal.Message, Continuation: result.Refusal.Continuation,
			Details: result.Refusal.Details}
	}
	docs := result.UncommittedDocs
	if docs == nil {
		docs = []string{}
	}
	return &MarksPublished{Status: result.Status, Remote: result.Remote, Branch: result.Branch, Commit: result.Commit,
		Pushed: result.PushedSHA, Replayed: result.Replayed, UncommittedDocs: docs, Marks: s.marksDTO(mc)}, nil
}
