package dashboard

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
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
	// versionStatusRestartRequired: el binario instalado es distinto del que sirve el
	// dashboard y no va por detrás de la última versión; basta con reiniciar
	// (RF-UPD-09).
	versionStatusRestartRequired = "restart_required"
)

// installedVersionTimeout acota la lectura de la versión del binario instalado.
const installedVersionTimeout = 3 * time.Second

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
	// Installed es la versión del binario que arrancaría un reinicio; vacía si no se
	// pudo leer. RestartRequired dice que es distinta de Current.
	Installed       string `json:"installed"`
	RestartRequired bool   `json:"restart_required"`
	// Instance identifica el proceso servidor que responde. Cambia en cada reinicio,
	// también cuando la versión es la misma.
	Instance string `json:"instance"`
}

// newServerInstance devuelve un identificador del proceso servidor, distinto en cada
// arranque. Lleva una parte aleatoria: el reloj de Windows puede repetir valores
// entre dos arranques seguidos.
func newServerInstance() string {
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	}
	return fmt.Sprintf("%d-%s", os.Getpid(), hex.EncodeToString(nonce[:]))
}

// versionCheckFn consulta la última versión publicada de Axiom. Es la costura
// que sustituyen las pruebas; la implementación real envuelve update.CheckFiltered.
type versionCheckFn func(ctx context.Context, current string) VersionInfo

// installedVersionFn lee la versión del binario que arrancaría un reinicio. Es una
// costura del paquete: las pruebas la sustituyen para no ejecutar el binario de
// pruebas.
var installedVersionFn = readInstalledAxiomVersion

// versionChecker comprueba la versión con una caché en memoria con TTL. El reloj, el
// comprobador y la lectura de la versión instalada son inyectables.
type versionChecker struct {
	mu        sync.Mutex
	ttl       time.Duration
	now       func() time.Time
	current   func() string
	check     versionCheckFn
	installed func(ctx context.Context) (string, error)

	cached    VersionInfo
	cachedFor string // versión con la que se hizo la comprobación cacheada
	cachedAt  time.Time
	hasCache  bool
}

func newVersionChecker() *versionChecker {
	return &versionChecker{
		ttl:       versionCacheTTL,
		now:       time.Now,
		current:   func() string { return app.Version },
		check:     checkLatestAxiomVersion,
		installed: func(ctx context.Context) (string, error) { return installedVersionFn(ctx) },
	}
}

// Get devuelve la información de versión. Con refresh omite la caché. Solo se
// cachean las comprobaciones concluyentes: un fallo de red se reintenta en la
// siguiente consulta en lugar de ocultar el aviso durante horas.
//
// La comprobación contra la última versión se hace con el binario que arrancaría un
// reinicio, no con el proceso: si otro `go install` ya lo sustituyó, lo que hace
// falta es reiniciar, no actualizar (RF-UPD-09). Sin esa versión, se usa la del
// proceso, como antes.
func (c *versionChecker) Get(ctx context.Context, refresh bool) VersionInfo {
	c.mu.Lock()
	defer c.mu.Unlock()

	current := c.current()
	installed := c.installedVersion(ctx)
	base := current
	if installed != "" {
		base = installed
	}
	if !refresh && c.hasCache && c.cachedFor == base && c.now().Sub(c.cachedAt) < c.ttl {
		return withInstalled(c.cached, current, installed)
	}

	info := c.check(ctx, base)
	if info.Status == versionStatusCheckFailed {
		// Un fallo no sustituye a una comprobación anterior válida del mismo
		// binario: se informa del fallo, pero la caché previa se conserva.
		return withInstalled(info, current, installed)
	}
	c.cached, c.cachedFor, c.cachedAt, c.hasCache = info, base, c.now(), true
	return withInstalled(info, current, installed)
}

func (c *versionChecker) installedVersion(ctx context.Context) string {
	if c.installed == nil {
		return ""
	}
	v, err := c.installed(ctx)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

// withInstalled completa la respuesta con la versión instalada. Con un binario
// instalado distinto del proceso, el estado pasa a restart_required salvo que haya
// una versión más reciente que instalar: entonces sigue siendo update_available, y
// la actualización reinicia después.
func withInstalled(info VersionInfo, current, installed string) VersionInfo {
	info.Current = current
	info.Installed = installed
	info.RestartRequired = installed != "" && installed != current
	if info.RestartRequired && info.Status != versionStatusUpdateAvailable {
		info.Status = versionStatusRestartRequired
		info.UpdateAvailable = false
	}
	return info
}

// readInstalledAxiomVersion ejecuta `<binario> version` sobre el ejecutable que
// lanzaría un reinicio (resolveRelaunchExecutable). Se ejecuta en lugar de leer la
// información de compilación porque las releases inyectan la versión con -ldflags.
func readInstalledAxiomVersion(ctx context.Context) (string, error) {
	exe, err := resolveRelaunchExecutable()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, installedVersionTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, exe, "version").Output()
	if err != nil {
		return "", fmt.Errorf("%s version: %w", exe, err)
	}
	return parseVersionOutput(string(out))
}

// parseVersionOutput extrae la versión de la salida de `axiom version`:
// `<plataforma> version <versión> (<os>/<arch>) commit:<commit>`.
func parseVersionOutput(out string) (string, error) {
	fields := strings.Fields(out)
	for i := 0; i+1 < len(fields); i++ {
		if fields[i] == "version" {
			return fields[i+1], nil
		}
	}
	return "", fmt.Errorf("salida de `version` no reconocida: %q", strings.TrimSpace(out))
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
	results := update.CheckFiltered(ctx, current, detection.System.Profile, []string{update.SelfToolName})
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
	info := s.versions.Get(ctx, refresh)
	info.Instance = s.instance
	return info
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
