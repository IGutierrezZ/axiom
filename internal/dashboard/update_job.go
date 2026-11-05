package dashboard

import (
	"context"
	"fmt"
	"net/http"
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
	Phase      string             `json:"phase"`
	Running    bool               `json:"running"`
	Message    string             `json:"message"`
	ManualHint string             `json:"manual_hint"`
	Sync       *UpdateSyncSummary `json:"sync"`
}

// updateJobState guarda el único trabajo de actualización del proceso. Vive en el
// servicio, no en la petición HTTP: la actualización puede tardar más que el
// WriteTimeout del servidor y debe seguir aunque el navegador cierre la conexión.
type updateJobState struct {
	mu      sync.Mutex
	status  UpdateJobStatus
	running bool
	done    chan struct{} // se cierra al terminar el trabajo en curso
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
	j.status = UpdateJobStatus{Phase: updatePhaseChecking, Message: "Comprobando la última versión de Axiom…"}
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
		// sincroniza (ResolveSyncSkip). El relanzamiento lo gestiona la fase siguiente.
		j.setPhase(updatePhaseRestarting, "Axiom se ha actualizado. Reiniciando el dashboard con la versión nueva…")
		s.afterSelfUpgrade(sync)
		return
	}
	j.finish(updatePhaseDone, resp.Message, "", sync)
}

// afterSelfUpgrade se ejecuta cuando el binario propio ya se ha reemplazado.
func (s *Service) afterSelfUpgrade(sync *UpdateSyncSummary) {
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
