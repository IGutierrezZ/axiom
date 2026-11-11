package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/livingdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/semantic"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// init registra los tipos MIME de los recursos estáticos del dashboard.
//
// Go resuelve el Content-Type por extensión y, en Windows, lo toma del registro
// del sistema: un .js asociado a text/plain haría que el navegador rechace los
// módulos ES (<script type="module">). Se fijan explícitamente. .woff2 no lo
// conoce el paquete mime y se registra para cuando se embeban las fuentes
// (assets/fonts/, paso posterior).
func init() {
	for ext, typ := range map[string]string{
		".js":    "text/javascript; charset=utf-8",
		".mjs":   "text/javascript; charset=utf-8",
		".css":   "text/css; charset=utf-8",
		".svg":   "image/svg+xml",
		".woff2": "font/woff2",

		".webmanifest": "application/manifest+json",
	} {
		_ = mime.AddExtensionType(ext, typ)
	}
}

// Server representa el servidor HTTP local para el dashboard web de Axiom.
type Server struct {
	service    *Service
	mux        *http.ServeMux
	httpServer *http.Server
	listener   net.Listener
	port       int
	version    string
}

// NewServer inicializa el servidor HTTP y configura las rutas de la API y assets estáticos.
func NewServer(svc *Service) *Server {
	s := &Server{
		service: svc,
		mux:     http.NewServeMux(),
	}
	svc.restartFn = s.relaunchAfterUpdate
	s.registerRoutes()
	return s
}

// Router retorna el ServeMux para pruebas unitarias con httptest.
func (s *Server) Router() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() {
	// 1. Endpoints de la API REST
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/projects", s.handleProjects)
	s.mux.HandleFunc("/api/projects/switch", s.handleProjectsSwitch)
	s.mux.HandleFunc("/api/projects/add", s.handleProjectsAdd)
	s.mux.HandleFunc("/api/projects/init", s.handleProjectsInit)
	s.mux.HandleFunc("/api/projects/active/worktree-prefs", s.handleActiveWorktreePrefs)
	s.mux.HandleFunc("/api/fs/directories", s.handleFSDirectories)
	s.mux.HandleFunc("/api/fs/native-picker", s.handleFSNativePicker)
	s.mux.HandleFunc("/api/workspace", s.handleWorkspace)
	s.mux.HandleFunc("/api/workspace/specs/sync-status", s.handleSpecsSyncStatus)
	s.mux.HandleFunc("/api/workspace/specs/pull", s.handleSpecsPull)
	s.mux.HandleFunc("/api/increments", s.handleIncrements)
	s.mux.HandleFunc("/api/increments/continue", s.handleIncrementContinue)
	s.mux.HandleFunc("/api/increments/verify", s.handleIncrementVerify)
	s.mux.HandleFunc("/api/increments/migrate-cumulative", s.handleMigrateCumulative)
	s.mux.HandleFunc("/api/increments/", s.handleIncrementDetail)
	s.mux.HandleFunc("/api/roles", s.handleRoles)
	s.mux.HandleFunc("/api/handoffs", s.handleHandoffs)
	s.mux.HandleFunc("/api/skills", s.handleSkills)
	s.mux.HandleFunc("/api/skills/inbox", s.handleSkillsInbox)
	s.mux.HandleFunc("/api/skills/scan", s.handleSkillsScan)
	s.mux.HandleFunc("/api/skills/approve", s.handleSkillsApprove)
	s.mux.HandleFunc("/api/skills/reject", s.handleSkillsReject)
	s.mux.HandleFunc("/api/semantic/status", s.handleSemanticStatus)
	s.mux.HandleFunc("/api/semantic/symbols", s.handleSemanticSymbols)
	s.mux.HandleFunc("/api/semantic/dependencies", s.handleSemanticDependencies)
	s.mux.HandleFunc("/api/semantic/reindex", s.handleSemanticReindex)
	s.mux.HandleFunc("/api/archive/specs", s.handleArchiveSpecs)
	s.mux.HandleFunc("/api/archive/specs/", s.handleArchiveSpecDetail)
	s.mux.HandleFunc("/api/archive/sync", s.handleArchiveSync)
	s.mux.HandleFunc("/api/ecosystem/doctor", s.handleEcosystemDoctor)
	s.mux.HandleFunc("/api/ecosystem/sync", s.handleEcosystemSync)
	s.mux.HandleFunc("/api/ecosystem/upgrade", s.handleEcosystemUpgrade)
	s.mux.HandleFunc("/api/ecosystem/backups", s.handleEcosystemBackups)
	s.mux.HandleFunc("/api/ecosystem/backups/create", s.handleEcosystemBackupCreate)
	s.mux.HandleFunc("/api/ecosystem/backups/restore", s.handleEcosystemBackupRestore)
	s.mux.HandleFunc("/api/ecosystem/models", s.handleEcosystemModels)
	s.mux.HandleFunc("/api/system/version", s.handleSystemVersion)
	s.mux.HandleFunc("/api/system/update", s.handleSystemUpdate)
	s.mux.HandleFunc("/api/system/update/status", s.handleSystemUpdateStatus)

	// Iconos PNG de la aplicación, rasterizados en Go (sin binarios en el repo).
	s.mux.HandleFunc("/icons/icon-192.png", iconHandler(192, false))
	s.mux.HandleFunc("/icons/icon-512.png", iconHandler(512, false))
	s.mux.HandleFunc("/icons/icon-maskable-512.png", iconHandler(512, true))

	// 2. Servidor de Archivos Estáticos Embebidos
	subFS, err := fs.Sub(AssetsFS, "assets")
	if err == nil {
		fileServer := http.FileServer(http.FS(subFS))
		s.mux.Handle("/", fileServer)
	} else {
		s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Assets embebidos no disponibles", http.StatusInternalServerError)
		})
	}
}

// ListenAndServe escucha exactamente en 127.0.0.1:port. No reasigna el puerto:
// si está ocupado devuelve un error que envuelve ErrPortInUse.
func (s *Server) ListenAndServe(port int) (int, error) {
	return s.ListenAndServeWithRetry(port, 0)
}

// ListenAndServeWithRetry es ListenAndServe con reintentos mientras el puerto siga
// ocupado, hasta retryFor. Lo usa solo el proceso que se relanza tras una
// actualización (--post-update-sync): el dashboard viejo aún está liberando el puerto
// y sondear otro rompería la URL que el navegador (o la PWA instalada) ya tiene. Con
// retryFor <= 0 no reintenta. Un fallo distinto de «puerto ocupado» no se reintenta.
// Si el puerto no queda libre a tiempo, el error envuelve ErrPortInUse.
func (s *Server) ListenAndServeWithRetry(port int, retryFor time.Duration) (int, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(retryFor)
	for {
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			s.serve(listener, port)
			return port, nil
		}
		if !isAddrInUse(err) {
			return 0, err
		}
		if !time.Now().Before(deadline) {
			return 0, fmt.Errorf("puerto %d: %w", port, ErrPortInUse)
		}
		time.Sleep(listenRetryInterval)
	}
}

// listenRetryInterval es la pausa entre reintentos de ListenAndServeWithRetry.
const listenRetryInterval = 100 * time.Millisecond

// serve arranca el servidor HTTP sobre un listener ya enlazado.
func (s *Server) serve(listener net.Listener, port int) {
	s.listener = listener
	s.port = listener.Addr().(*net.TCPAddr).Port

	s.httpServer = &http.Server{
		Handler:      s.withHeaders(s.mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		_ = s.httpServer.Serve(listener)
	}()
}

// Shutdown detiene el servidor de forma ordenada.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// Port retorna el puerto donde está escuchando el servidor.
func (s *Server) Port() int {
	return s.port
}

// Códigos de Winsock para un puerto ocupado (WSAEADDRINUSE) o reservado
// (WSAEACCES). Windows no los traduce a syscall.EADDRINUSE y su texto va
// localizado, así que se comparan por número.
const (
	wsaEAddrInUse = syscall.Errno(10048)
	wsaEAccess    = syscall.Errno(10013)
)

// isAddrInUse distingue «puerto ocupado» de otros fallos de escucha.
func isAddrInUse(err error) bool {
	if errors.Is(err, syscall.EADDRINUSE) || errors.Is(err, wsaEAddrInUse) || errors.Is(err, wsaEAccess) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "only one usage of each socket address") ||
		strings.Contains(msg, "forbidden by its access permissions")
}

func (s *Server) withHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		// La SPA lee los avisos de los listados (REQ-25.34) de esta cabecera.
		w.Header().Set("Access-Control-Expose-Headers", notesHeaderName)

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	projects, err := s.service.GetProjects()
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, projects)
}

func (s *Server) handleProjectsSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var req ProjectSwitchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	target := req.Path
	if target == "" {
		target = req.ID
	}
	if target == "" {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Debes especificar 'id' o 'path'"})
		return
	}

	// Si target no es un directorio físico directo, buscarlo en el Hub por ID o nombre
	if !dirExists(target) && s.service.GetHubManager() != nil {
		rec, err := s.service.GetHubManager().FindWorkspace(target)
		if err == nil && rec != nil {
			target = rec.Path
		}
	}

	ws, err := s.service.SwitchWorkspace(target)
	if err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	s.respondJSON(w, http.StatusOK, ws)
}

func (s *Server) handleProjectsAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var req ProjectAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	if err := validateWorktreePrefs(req.WorktreeMode, req.WorktreeRoot); err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	rec, err := s.service.AddProject(req)
	if err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	s.respondJSON(w, http.StatusOK, rec)
}

func (s *Server) handleProjectsInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var req ProjectInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	if err := validateWorktreePrefs(req.WorktreeMode, req.WorktreeRoot); err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	res, err := s.service.InitProject(req)
	if err != nil {
		// Una configuración inválida es un error del usuario: 400 con todas las
		// violaciones, para que el modal las muestre una a una. Solo lo inesperado
		// es un 500.
		var invalid *hub.InitValidationError
		if errors.As(err, &invalid) {
			s.respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":      "La configuración del proyecto no es válida.",
				"violations": invalid.Violations,
			})
			return
		}
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.respondJSON(w, http.StatusOK, res)
}

func (s *Server) handleFSDirectories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	targetPath := r.URL.Query().Get("path")
	res, err := s.service.BrowseDirectories(targetPath)
	if err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, res)
}

func (s *Server) handleFSNativePicker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var req NativePickerRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	selected, err := s.service.PickFolderNative(r.Context(), req.InitialPath)
	if err != nil {
		s.respondJSON(w, http.StatusOK, NativePickerResult{
			Canceled: true,
			Error:    err.Error(),
		})
		return
	}

	s.respondJSON(w, http.StatusOK, NativePickerResult{
		Path:     selected,
		Canceled: selected == "",
	})
}

// handleWorkspace responde `GET /api/workspace` con los metadatos del workspace y su política de
// ramas (REQ-1.1). Es solo lectura: cualquier otro método responde 405 y ninguna ruta del servidor
// escribe axiom.yaml (REQ-25.33).
func (s *Server) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	dto, err := s.service.GetWorkspace()
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, dto)
}

func (s *Server) handleSpecsSyncStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	dto, err := s.service.GetSpecsSyncStatus(r.Context())
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, dto)
}

func (s *Server) handleSpecsPull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	dto, err := s.service.PullSpecsRepository(r.Context())
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, dto)
		return
	}
	s.respondJSON(w, http.StatusOK, dto)
}

func (s *Server) handleIncrements(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, notes, err := s.service.GetIncrements()
		if err != nil {
			s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		// REQ-25.34: los avisos informativos (cambios pendientes de archivar, worktrees desaparecidos,
		// falta de la referencia de integración) viajan en la cabecera, no en el cuerpo, que sigue
		// siendo un array. Sin avisos no hay cabecera.
		if header := notesHeader(notes); header != "" {
			w.Header().Set(notesHeaderName, header)
		}
		s.respondJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var req CreateIncrementRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido: " + err.Error()})
			return
		}
		res, err := s.service.CreateIncrement(req)
		if err != nil {
			// REQ-25.32: un rechazo tipado de `ensure` o de la guarda A es un 409 con su código y su
			// continuación (o, en `decision_required`, con la decisión), no un 400 genérico.
			var refusal *branch.RefusalError
			if errors.As(err, &refusal) {
				s.respondJSON(w, http.StatusConflict, conflictBody(refusal))
				return
			}
			s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		s.respondJSON(w, http.StatusCreated, res)
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleIncrementContinue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var req IncrementActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido: " + err.Error()})
		return
	}
	res, err := s.service.ContinueIncrement(req.Name)
	if err != nil {
		s.respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, res)
}

func (s *Server) handleIncrementVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var req IncrementActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido: " + err.Error()})
		return
	}
	res, err := s.service.VerifyIncrement(req.Name)
	if err != nil {
		s.respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, res)
}

func (s *Server) handleIncrementDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/api/increments/")
	name = strings.TrimSpace(name)
	if name == "" {
		http.Error(w, "Nombre de incremento requerido", http.StatusBadRequest)
		return
	}

	dto, err := s.service.GetIncrementDetail(name)
	if err != nil {
		s.respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, dto)
}

func (s *Server) handleMigrateCumulative(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var req MigrateCumulativeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}
	if strings.TrimSpace(req.Change) == "" || strings.TrimSpace(req.Role) == "" {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Campos 'change' y 'role' son requeridos"})
		return
	}

	count, err := s.service.MigrateIncompleteTasksToCumulative(req.Change, req.Role)
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"migrated": count,
		"change":   req.Change,
		"role":     req.Role,
	})
}

func (s *Server) handleRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	change := r.URL.Query().Get("change")
	if change == "" {
		http.Error(w, "Parámetro 'change' es requerido", http.StatusBadRequest)
		return
	}

	barrier, err := s.service.GetRoleStatus(change)
	if err != nil {
		s.respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, NewBarrierDTO(barrier))
}

func (s *Server) handleHandoffs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		change := r.URL.Query().Get("change")
		if change == "" {
			http.Error(w, "Parámetro 'change' es requerido", http.StatusBadRequest)
			return
		}

		ho, err := s.service.GetHandoff(change)
		if err != nil {
			s.respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		s.respondJSON(w, http.StatusOK, NewHandoffDTO(ho))
	case http.MethodPost:
		var req CreateHandoffRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido: " + err.Error()})
			return
		}
		res, err := s.service.CreateHandoff(req)
		if err != nil {
			s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		s.respondJSON(w, http.StatusCreated, res)
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	skills, err := s.service.GetSkills()
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, skills)
}

func (s *Server) handleSkillsInbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	proposals, err := s.service.GetSkillsInbox()
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, proposals)
}

func (s *Server) handleSkillsScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req SkillActionDTO
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	report, err := s.service.ScanSkills(r.Context(), req.Role, req.Offline)
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, report)
}

func (s *Server) handleSkillsApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req SkillActionDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "Campo 'name' es requerido en el cuerpo JSON", http.StatusBadRequest)
		return
	}

	warning, err := s.service.ApproveSkill(req.Name)
	if err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	message := fmt.Sprintf("Skill '%s' aprobada e instalada con éxito", req.Name)
	if warning != "" {
		// Warning, never an error: the promotion stands (REQ-22.13).
		message += " (aviso: " + warning + ")"
	}
	s.respondJSON(w, http.StatusOK, map[string]string{
		"status":  "approved",
		"message": message,
	})
}

func (s *Server) handleSkillsReject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req SkillActionDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "Campo 'name' es requerido en el cuerpo JSON", http.StatusBadRequest)
		return
	}

	if err := s.service.RejectSkill(req.Name); err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, map[string]string{
		"status":  "rejected",
		"message": fmt.Sprintf("Propuesta '%s' descartada del buzón", req.Name),
	})
}

func (s *Server) handleSemanticStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	status, err := s.service.GetSemanticStatus(r.Context())
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, status)
}

func (s *Server) handleSemanticSymbols(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	q := semantic.SemanticQuery{
		Query: r.URL.Query().Get("query"),
		Kind:  semantic.SymbolKind(r.URL.Query().Get("kind")),
		Role:  r.URL.Query().Get("role"),
	}
	symbols, err := s.service.FindSemanticSymbols(q)
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if symbols == nil {
		symbols = make([]semantic.SymbolItem, 0)
	}
	s.respondJSON(w, http.StatusOK, symbols)
}

func (s *Server) handleSemanticDependencies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	role := r.URL.Query().Get("role")
	deps, err := s.service.InspectSemanticDependencies(role)
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if deps == nil {
		deps = make([]semantic.DependencyRelation, 0)
	}
	s.respondJSON(w, http.StatusOK, deps)
}

func (s *Server) handleSemanticReindex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	res, err := s.service.ReindexCodeGraph(r.Context())
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, res)
		return
	}
	s.respondJSON(w, http.StatusOK, res)
}

func (s *Server) handleArchiveSpecs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	catalog, err := s.service.GetLivingSpecs(r.Context())
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, catalog)
}

func (s *Server) handleArchiveSpecDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	domain := strings.TrimPrefix(r.URL.Path, "/api/archive/specs/")
	domain = strings.TrimSpace(domain)
	if domain == "" {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Dominio de especificación requerido"})
		return
	}
	entry, content, err := s.service.GetLivingSpecDetail(domain)
	if err != nil {
		// Un dominio de la baseline externa del proyecto no se indexa: es un error
		// de cliente que nombra la baseline y la ruta de su documento (REQ-24.25).
		var external *livingdoc.ExternalBaselineError
		if errors.As(err, &external) {
			s.respondJSON(w, http.StatusNotFound, map[string]interface{}{
				"error":             err.Error(),
				"external_baseline": true,
				"domain":            external.Domain,
				"path":              external.Path,
			})
			return
		}
		s.respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"entry":   entry,
		"content": content,
	})
}

func (s *Server) handleArchiveSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	report, err := s.service.SyncLivingDocs(r.Context())
	if err != nil {
		// Con una baseline del proyecto, o indeterminada, el servicio rechaza sin
		// escribir: 409 con el mensaje, que nombra la baseline y la continuación
		// (REQ-24.23).
		var second *workspace.SecondBaselineError
		if errors.As(err, &second) {
			s.respondJSON(w, http.StatusConflict, map[string]string{
				"error":     err.Error(),
				"ownership": string(second.Ownership),
			})
			return
		}
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, report)
}

func (s *Server) handleEcosystemDoctor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	report, err := s.service.GetDoctorDiagnostics()
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, report)
}

func (s *Server) handleEcosystemSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var req EcosystemSyncRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	scope := "workspace"
	if strings.TrimSpace(req.Scope) != "" {
		scope = strings.TrimSpace(req.Scope)
	}
	resp, err := s.service.RunSync(scope)
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, resp)
		return
	}
	s.respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleEcosystemUpgrade(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var req EcosystemUpgradeRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	resp, err := s.service.RunUpgradeSequence(req.Channel)
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, resp)
		return
	}
	s.respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleEcosystemBackups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	backups, err := s.service.GetBackups()
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, backups)
}

func (s *Server) handleEcosystemBackupCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var req BackupActionRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	item, err := s.service.CreateBackup(req.Description)
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusCreated, item)
}

func (s *Server) handleEcosystemBackupRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var req BackupActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Identificador de respaldo (name) requerido"})
		return
	}
	resp, err := s.service.RestoreBackup(req.Name)
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, resp)
		return
	}
	s.respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleEcosystemModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	models, err := s.service.GetModelAssignments()
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, models)
}

// decisionRequiredBody es el cuerpo del 409 de `decision_required` (REQ-25.32): el estado con su
// etiqueta, el código y la decisión que la interfaz presenta antes de repetir la creación con `mode`.
type decisionRequiredBody struct {
	Status      string           `json:"status"`
	StatusLabel string           `json:"statusLabel"`
	Code        branch.Code      `json:"code"`
	Decision    *branch.Decision `json:"decision"`
}

// refusalBody es el cuerpo del 409 de cualquier otro rechazo de `ensure` o de la guarda A: el
// código tipado, el mensaje y la continuación ejecutable.
type refusalBody struct {
	Code         branch.Code `json:"code"`
	Error        string      `json:"error"`
	Continuation string      `json:"continuation"`
}

// conflictBody traduce un rechazo tipado al cuerpo del 409: `{status, statusLabel, code, decision}`
// para `decision_required` y `{code, error, continuation}` para el resto.
func conflictBody(refusal *branch.RefusalError) any {
	if refusal.Code == branch.CodeDecisionRequired && refusal.Decision != nil {
		status := string(branch.StatusDecisionRequired)
		return decisionRequiredBody{
			Status:      status,
			StatusLabel: branch.Label(status),
			Code:        refusal.Code,
			Decision:    refusal.Decision,
		}
	}
	return refusalBody{Code: refusal.Code, Error: refusal.Message, Continuation: refusal.Continuation}
}

// isLocalRequest protege las rutas que cambian estado frente a CSRF y DNS rebinding:
// withHeaders permite cualquier origen para lectura, así que aquí se exige que Host y,
// si viene, Origin sean el propio dashboard (127.0.0.1 o localhost en el puerto
// realmente enlazado). Sin servidor escuchando (puerto 0, pruebas con httptest) se
// acepta cualquier puerto, pero solo con esos nombres de host.
func (s *Server) isLocalRequest(r *http.Request) bool {
	allowed := func(hostport string) bool {
		host, port, err := net.SplitHostPort(hostport)
		if err != nil {
			host, port = hostport, ""
		}
		if host != "127.0.0.1" && host != "localhost" {
			return false
		}
		return s.port == 0 || port == strconv.Itoa(s.port)
	}
	if !allowed(r.Host) {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		rest, ok := strings.CutPrefix(origin, "http://")
		if !ok || !allowed(rest) {
			return false
		}
	}
	return true
}

func (s *Server) respondJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

// notesHeaderName es la cabecera de `GET /api/increments` con los avisos informativos de los
// listados (REQ-25.34, design D-20): un array JSON de frases en castellano.
const notesHeaderName = "X-Axiom-Notes"

// notesHeader codifica los avisos como un array JSON en ASCII: un valor de cabecera HTTP no debe
// llevar bytes fuera de ASCII, de modo que los acentos y el resto de caracteres no ASCII (también los
// de las rutas) viajan escapados como `\uXXXX`, que cualquier decodificador JSON restituye. Sin avisos
// devuelve la cadena vacía y la cabecera no se emite.
func notesHeader(notes []string) string {
	if len(notes) == 0 {
		return ""
	}
	encoded, err := json.Marshal(notes)
	if err != nil {
		return ""
	}
	var header strings.Builder
	for _, r := range string(encoded) {
		switch {
		case r < 0x7f:
			header.WriteRune(r)
		case r > 0xffff:
			high, low := utf16.EncodeRune(r)
			fmt.Fprintf(&header, `\u%04x\u%04x`, high, low)
		default:
			fmt.Fprintf(&header, `\u%04x`, r)
		}
	}
	return header.String()
}
