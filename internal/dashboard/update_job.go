package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/IGutierrezZ/axiom/v3/internal/app"
)

// Fases del trabajo de actualización que expone GET /api/system/update/status.
const (
	updatePhaseIdle       = "idle" // todavía no se ha lanzado ninguna actualización
	updatePhaseChecking   = "checking"
	updatePhaseUpgrading  = "upgrading"
	updatePhaseRestarting = "restarting"
	updatePhaseSyncing    = "syncing"
	updatePhaseDone       = "done"
	updatePhaseFailed     = "failed"
)

// manualUpdateHint es el comando manual que se ofrece cuando la actualización
// automática no puede completarse y la herramienta no aporta uno propio.
const manualUpdateHint = "go install github.com/IGutierrezZ/axiom/v3/cmd/axiom@latest"

// UpdateSyncSummary es el resultado de la sincronización del workspace asociada a
// la actualización. Executed es false cuando se omitió (y SkippedReason dice por qué).
type UpdateSyncSummary struct {
	Executed      bool     `json:"executed"`
	Success       bool     `json:"success"`
	SkippedReason string   `json:"skipped_reason"`
	Output        []string `json:"output,omitempty"`
	Error         string   `json:"error,omitempty"`
}

// UpdateJobStatus es la respuesta de GET /api/system/update/status.
type UpdateJobStatus struct {
	Phase      string `json:"phase"`
	Running    bool   `json:"running"`
	Message    string `json:"message"`
	ManualHint string `json:"manual_hint"`
	// FromVersion es la versión que servía el dashboard al lanzar el trabajo: la UI
	// espera a que el servidor responda con otra distinta para recargar.
	FromVersion string             `json:"from_version"`
	Sync        *UpdateSyncSummary `json:"sync"`
}

// updateJobState guarda el único trabajo de actualización del proceso. Vive en el
// servicio, no en la petición HTTP: la actualización puede tardar más que el
// WriteTimeout del servidor y debe seguir aunque el navegador cierre la conexión.
type updateJobState struct {
	mu      sync.Mutex
	status  UpdateJobStatus
	running bool
	done    chan struct{} // se cierra al terminar el trabajo en curso
	// pendingSync: la sincronización posterior a la actualización está anunciada
	// (MarkPostUpdateSyncPending) pero aún no arrancada (StartPostUpdateSync).
	pendingSync bool
}

func (j *updateJobState) snapshot() UpdateJobStatus {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.snapshotLocked()
}

func (j *updateJobState) snapshotLocked() UpdateJobStatus {
	st := j.status
	if st.Phase == "" {
		st.Phase = updatePhaseIdle
	}
	st.Running = j.running
	if j.status.Sync != nil {
		cp := *j.status.Sync
		cp.Output = append([]string(nil), cp.Output...)
		st.Sync = &cp
	}
	return st
}

// setPhase cambia de fase sin tocar el resto del estado.
func (j *updateJobState) setPhase(phase, message string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.status.Phase = phase
	j.status.Message = message
}

// setRestarting entra en la fase de reinicio y deja el comando manual a la vista por
// si la versión nueva no llega a arrancar.
func (j *updateJobState) setRestarting(message, manualHint string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.status.Phase = updatePhaseRestarting
	j.status.Message = message
	j.status.ManualHint = manualHint
}

// finish cierra el trabajo: fase terminal (done o failed) y fin de «en curso».
func (j *updateJobState) finish(phase, message, manualHint string, sync *UpdateSyncSummary) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.status.Phase = phase
	j.status.Message = message
	j.status.ManualHint = manualHint
	if sync != nil {
		j.status.Sync = sync
	}
	j.running = false
	if j.done != nil {
		close(j.done)
		j.done = nil
	}
}

// StartUpdate arranca la actualización en segundo plano. Hay un único trabajo a la
// vez: si ya hay uno en curso (en cualquier fase no terminal) devuelve su estado y
// no lanza otro. Devuelve el estado y si se ha lanzado un trabajo nuevo.
func (s *Service) StartUpdate() (UpdateJobStatus, bool) {
	j := &s.update
	j.mu.Lock()
	if j.running {
		st := j.snapshotLocked()
		j.mu.Unlock()
		return st, false
	}
	j.running = true
	j.done = make(chan struct{})
	j.status = UpdateJobStatus{Phase: updatePhaseChecking, Message: "Comprobando la última versión de Axiom…", FromVersion: s.versions.current()}
	st := j.snapshotLocked()
	j.mu.Unlock()

	go s.runUpdateJob()
	return st, true
}

// UpdateStatus devuelve el estado del trabajo de actualización.
func (s *Service) UpdateStatus() UpdateJobStatus {
	return s.update.snapshot()
}

// runUpdateJob es el cuerpo del trabajo: comprobar, actualizar y, si el binario
// no se ha reemplazado, sincronizar. Nunca propaga un pánico: lo convierte en una
// fase failed con el comando manual, y el dashboard sigue sirviendo.
func (s *Service) runUpdateJob() {
	j := &s.update
	defer func() {
		if r := recover(); r != nil {
			j.finish(updatePhaseFailed, fmt.Sprintf("La actualización se interrumpió: %v", r), manualUpdateHint, nil)
		}
	}()

	// 1. Comprobar. El trabajo no depende del contexto de la petición.
	info := s.versions.Get(context.Background(), true)
	switch info.Status {
	case versionStatusUpdateAvailable:
	case versionStatusUpToDate:
		j.finish(updatePhaseDone, "Axiom ya está en la última versión.", "", nil)
		return
	case versionStatusDevBuild:
		j.finish(updatePhaseDone, "Esta es una compilación de desarrollo: no se actualiza automáticamente.", "", nil)
		return
	default:
		j.finish(updatePhaseFailed, "No se pudo comprobar si hay una versión nueva.", manualUpdateHint, nil)
		return
	}

	// 2. Actualizar con la misma cadena que el botón de Ecosistema (upgrade -> sync).
	j.setPhase(updatePhaseUpgrading, fmt.Sprintf("Actualizando Axiom de %s a %s…", info.Current, info.Latest))
	resp, err := s.RunUpgradeSequence()
	if err != nil || resp == nil || resp.Phases == nil {
		msg := "La actualización no produjo resultado."
		if err != nil {
			msg = err.Error()
		}
		j.finish(updatePhaseFailed, msg, manualUpdateHint, nil)
		return
	}
	upgrade := resp.Phases.Upgrade
	if !upgrade.Success {
		msg := upgrade.Error
		if msg == "" {
			msg = resp.Message
		}
		j.finish(updatePhaseFailed, msg, hintOrDefault(upgrade.ManualHint), nil)
		return
	}
	// «skipped»: la herramienta propia terminó en un relevo manual y no se reemplazó.
	if upgrade.Status == app.UpgradeStatusSkipped && !upgrade.RestartRequired {
		j.finish(updatePhaseFailed, "Axiom no se pudo actualizar automáticamente.", hintOrDefault(upgrade.ManualHint), nil)
		return
	}

	sync := summarizeUpdateSync(resp.Phases.Sync)
	if upgrade.RestartRequired {
		// El binario en disco ya es el nuevo; este proceso sigue siendo el viejo y no
		// sincroniza (ResolveSyncSkip): lo hará el proceso nuevo tras relanzarse.
		// Si el relanzamiento falla, la UI ya tiene el comando manual.
		j.setRestarting("Axiom se ha actualizado. Reiniciando el dashboard con la versión nueva…", manualRelaunchHint(s.getRootPath()))
		s.afterSelfUpgrade(sync)
		return
	}
	j.finish(updatePhaseDone, resp.Message, "", sync)
}

// afterSelfUpgrade se ejecuta cuando el binario propio ya se ha reemplazado. El
// servidor registra restartFn (relaunchAfterUpdate); sin servidor no hay a quién
// relanzar y se pide el reinicio manual.
func (s *Service) afterSelfUpgrade(sync *UpdateSyncSummary) {
	if s.restartFn != nil {
		s.restartFn(sync)
		return
	}
	s.update.finish(updatePhaseDone, "Axiom se ha actualizado. Reinicia el dashboard para usar la versión nueva.", "", sync)
}

func hintOrDefault(hint string) string {
	if hint != "" {
		return hint
	}
	return manualUpdateHint
}

func summarizeUpdateSync(p SyncPhaseReport) *UpdateSyncSummary {
	return &UpdateSyncSummary{
		Executed:      p.Executed,
		Success:       p.Success,
		SkippedReason: p.SkippedReason,
		Output:        p.Output,
		Error:         p.Error,
	}
}

// handleSystemUpdate responde POST /api/system/update: arranca el trabajo (o
// devuelve el que ya está en curso) y contesta enseguida, sin esperar a que termine.
func (s *Server) handleSystemUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	if !s.isLocalRequest(r) {
		s.respondJSON(w, http.StatusForbidden, map[string]string{"error": "Petición rechazada: solo se admiten peticiones del propio dashboard"})
		return
	}
	st, _ := s.service.StartUpdate()
	s.respondJSON(w, http.StatusAccepted, st)
}

// handleSystemUpdateStatus responde GET /api/system/update/status.
func (s *Server) handleSystemUpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	s.respondJSON(w, http.StatusOK, s.service.UpdateStatus())
}

// manualSyncHint es el comando manual cuando la sincronización posterior falla.
const manualSyncHint = "axiom sync --scope workspace"

// StartPostUpdateSync lo llama el proceso relanzado tras una actualización
// (`axiom ui --post-update-sync`): sincroniza el workspace abierto con los assets del
// binario NUEVO, que es lo que el proceso viejo no podía hacer (ResolveSyncSkip).
//
// Deja el estado en `syncing` de forma síncrona (la UI que recarga lo ve ya) y
// ejecuta la sincronización en segundo plano. El proceso NO tiene el workspace como
// directorio de trabajo (runUI lo deja en un directorio neutro, ServerWorkDir): la raíz
// sincronizada es la de --path (getRootPath), porque RunSync entra en ella con un
// os.Chdir temporal y restaura el directorio anterior al terminar. Ese cambio es del
// proceso entero: mientras dura, otras acciones que dependan del cwd lo verían en el
// workspace (riesgo previo de RunSync, anotado como seguimiento). Un POST /api/system/update durante la sincronización devuelve este mismo
// trabajo en lugar de lanzar otro. Si ya hay un trabajo en curso no hace nada.
func (s *Service) StartPostUpdateSync() {
	j := &s.update
	j.mu.Lock()
	if j.pendingSync {
		j.pendingSync = false
		j.mu.Unlock()
		go s.runPostUpdateSync()
		return
	}
	if j.running {
		j.mu.Unlock()
		return
	}
	j.running = true
	j.done = make(chan struct{})
	j.status = UpdateJobStatus{Phase: updatePhaseSyncing, Message: "Sincronizando el workspace con la versión nueva de Axiom…"}
	j.mu.Unlock()

	go s.runPostUpdateSync()
}

// MarkPostUpdateSyncPending anuncia la sincronización posterior a la actualización
// (estado `syncing`) sin ejecutarla todavía. Se llama antes de enlazar el puerto para
// que la UI nunca vea «idle»; la sincronización real la arranca StartPostUpdateSync
// una vez enlazado, de modo que un fallo de enlace (os.Exit) no la corte a medias.
func (s *Service) MarkPostUpdateSyncPending() {
	j := &s.update
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.running {
		return
	}
	j.running = true
	j.pendingSync = true
	j.done = make(chan struct{})
	j.status = UpdateJobStatus{Phase: updatePhaseSyncing, Message: "Sincronizando el workspace con la versión nueva de Axiom…"}
}

func (s *Service) runPostUpdateSync() {
	j := &s.update
	defer func() {
		if r := recover(); r != nil {
			j.finish(updatePhaseFailed, fmt.Sprintf("La sincronización se interrumpió: %v", r), manualSyncHint, nil)
		}
	}()

	version := s.versions.current()
	// Sin axiom.yaml no hay workspace de Axiom que sincronizar: el ámbito workspace
	// escribe en la carpeta, y no se hace en un directorio cualquiera.
	if !fileExists(filepath.Join(s.getRootPath(), "axiom.yaml")) {
		j.finish(updatePhaseDone, fmt.Sprintf("Axiom se ha actualizado a %s. La carpeta abierta no es un workspace de Axiom: no hay nada que sincronizar.", version), "",
			&UpdateSyncSummary{SkippedReason: "no-workspace"})
		return
	}

	resp, err := s.RunSync("workspace")
	if err != nil || resp == nil {
		msg := "La sincronización no produjo resultado."
		if err != nil {
			msg = err.Error()
		}
		j.finish(updatePhaseFailed, "Axiom se ha actualizado, pero la sincronización del workspace falló: "+msg, manualSyncHint,
			&UpdateSyncSummary{Executed: true, Error: msg})
		return
	}
	sync := &UpdateSyncSummary{Executed: true, Success: resp.Success, Output: resp.Output, Error: resp.Error}
	if !resp.Success {
		j.finish(updatePhaseFailed, "Axiom se ha actualizado, pero la sincronización del workspace falló: "+resp.Error, manualSyncHint, sync)
		return
	}
	j.finish(updatePhaseDone, fmt.Sprintf("Axiom se ha actualizado a %s y el workspace está sincronizado.", version), "", sync)
}
