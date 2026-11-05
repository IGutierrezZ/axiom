package dashboard

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/app"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
	"github.com/IGutierrezZ/axiom/v3/internal/update"
)

// Estados de la comprobación de versión que expone GET /api/system/version.
const (
	versionStatusUpToDate        = "up_to_date"
	versionStatusUpdateAvailable = "update_available"
	versionStatusDevBuild        = "dev_build"
	versionStatusCheckFailed     = "check_failed"
)

// versionCacheTTL es la vigencia de una comprobación correcta en memoria. Una
// consulta de red por cada carga del dashboard sería excesiva para un dato que
// cambia como mucho una vez por release; «Comprobar de nuevo» la omite.
const versionCacheTTL = 6 * time.Hour

// versionCheckTimeout acota la consulta a la red para quedar por debajo del
// WriteTimeout del servidor (15 s).
const versionCheckTimeout = 10 * time.Second

// VersionInfo es la respuesta de GET /api/system/version.
type VersionInfo struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"update_available"`
	Status          string `json:"status"`
	ReleaseURL      string `json:"release_url"`
}

// versionCheckFn consulta la última versión publicada de Axiom. Es la costura
// que sustituyen las pruebas; la implementación real envuelve update.CheckFiltered.
type versionCheckFn func(ctx context.Context, current string) VersionInfo

// versionChecker comprueba la versión con una caché en memoria con TTL. El reloj y
// el comprobador son inyectables.
type versionChecker struct {
	mu      sync.Mutex
	ttl     time.Duration
	now     func() time.Time
	current func() string
	check   versionCheckFn

	cached   VersionInfo
	cachedAt time.Time
	hasCache bool
}

func newVersionChecker() *versionChecker {
	return &versionChecker{
		ttl:     versionCacheTTL,
		now:     time.Now,
		current: func() string { return app.Version },
		check:   checkLatestAxiomVersion,
	}
}

// Get devuelve la información de versión. Con refresh omite la caché. Solo se
// cachean las comprobaciones concluyentes: un fallo de red se reintenta en la
// siguiente consulta en lugar de ocultar el aviso durante horas.
func (c *versionChecker) Get(ctx context.Context, refresh bool) VersionInfo {
	c.mu.Lock()
	defer c.mu.Unlock()

	current := c.current()
	if !refresh && c.hasCache && c.cached.Current == current && c.now().Sub(c.cachedAt) < c.ttl {
		return c.cached
	}

	info := c.check(ctx, current)
	info.Current = current
	if info.Status == versionStatusCheckFailed {
		// Un fallo no sustituye a una comprobación anterior válida del mismo
		// binario: se informa del fallo, pero la caché previa se conserva.
		return info
	}
	c.cached, c.cachedAt, c.hasCache = info, c.now(), true
	return info
}

// checkLatestAxiomVersion es el comprobador real: pregunta por la última release
// de Axiom. No usa CheckAllWithCooldown, que devuelve nil con su propia caché
// vigente y dejaría al aviso sin datos.
func checkLatestAxiomVersion(ctx context.Context, current string) VersionInfo {
	ctx, cancel := context.WithTimeout(ctx, versionCheckTimeout)
	defer cancel()

	detection, err := system.Detect(ctx)
	if err != nil {
		return VersionInfo{Current: current, Status: versionStatusCheckFailed}
	}
	// El registro de herramientas llama "axiom" a la herramienta propia; en las
	// pruebas del paquete conserva su nombre original.
	results := update.CheckFiltered(ctx, current, detection.System.Profile, []string{"axiom", "gentle-ai"})
	for _, r := range results {
		if !update.IsSelfTool(r.Tool) {
			continue
		}
		return versionInfoFromResult(current, r)
	}
	return VersionInfo{Current: current, Status: versionStatusCheckFailed}
}

// versionInfoFromResult traduce el resultado de update al DTO. Todo estado que no
// sea concluyente (no instalado, versión desconocida, error) se reduce a
// check_failed para que la interfaz no muestre aviso.
func versionInfoFromResult(current string, r update.UpdateResult) VersionInfo {
	info := VersionInfo{Current: current, Latest: r.LatestVersion, ReleaseURL: r.ReleaseURL}
	switch r.Status {
	case update.UpdateAvailable:
		info.Status = versionStatusUpdateAvailable
		info.UpdateAvailable = true
	case update.UpToDate:
		info.Status = versionStatusUpToDate
	case update.DevBuild:
		info.Status = versionStatusDevBuild
	default:
		info.Status = versionStatusCheckFailed
	}
	return info
}

// CheckVersion devuelve la información de versión del Axiom que sirve el dashboard.
func (s *Service) CheckVersion(ctx context.Context, refresh bool) VersionInfo {
	return s.versions.Get(ctx, refresh)
}

// handleSystemVersion responde GET /api/system/version. Un fallo de red no es un
// error HTTP: se informa con status "check_failed" y la interfaz no muestra aviso.
func (s *Server) handleSystemVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	refresh := r.URL.Query().Get("refresh") == "1"
	s.respondJSON(w, http.StatusOK, s.service.CheckVersion(r.Context(), refresh))
}
