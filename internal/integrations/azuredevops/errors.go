package azuredevops

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

// Centinelas para errors.Is. Cada fallo del cliente es un error tipado; nunca se silencia.
var (
	// ErrConfig indica que integrations.azure-devops no trae un valor utilizable (clave ausente,
	// vacía o que no es una cadena) o que un segmento de ruta no es válido. La carga de axiom.yaml ya
	// valida el bloque: este error cubre a quien llama al plugin con una configuración construida a
	// mano y los nombres que llegan de la API.
	ErrConfig = errors.New("configuración de azure-devops inválida")
	// ErrLifecycleUnsupported marca la entrega de eventos del ciclo de vida: este adaptador todavía
	// no entrega ninguno; llegan con C-07-d y C-07-e.
	ErrLifecycleUnsupported = errors.New("el plugin azure-devops no entrega eventos del ciclo de vida en esta versión; llegan con C-07-d y C-07-e")

	ErrAuth            = errors.New("azure devops rechazó la autenticación")
	ErrNotFound        = errors.New("azure devops no encontró el recurso")
	ErrRateLimited     = errors.New("azure devops limita la tasa de peticiones")
	ErrUnavailable     = errors.New("azure devops no está disponible")
	ErrTransport       = errors.New("fallo de red hacia azure devops")
	ErrBadRequest      = errors.New("azure devops rechazó la petición por inválida")
	ErrServer          = errors.New("azure devops devolvió un error del servidor")
	ErrInvalidResponse = errors.New("respuesta de azure devops inválida")
	ErrRevConflict     = errors.New("el work item cambió desde que se leyó (conflicto de /rev)")
	ErrMissingRev      = errors.New("una modificación de work item exige la revisión (rev) leída")
	// ErrPagination indica que una lista paginada no se pudo leer entera. Nunca se devuelve un
	// resultado truncado como si fuera completo.
	ErrPagination = errors.New("azure devops devolvió una paginación que no se puede completar")
)

// PaginationError indica que la paginación de una lista no terminó de forma fiable: se alcanzó el
// tope de páginas o el servidor devolvió una página llena sin elementos nuevos (ignora $skip). Pages
// es el número de páginas leídas.
type PaginationError struct {
	Operation string
	Pages     int
	Reason    string
}

func (e *PaginationError) Error() string {
	return fmt.Sprintf("la paginación al %s no se puede completar tras %d página(s): %s; no se devuelve un resultado parcial", e.Operation, e.Pages, e.Reason)
}

func (e *PaginationError) Is(target error) bool { return target == ErrPagination }

// AuthError es un rechazo de autenticación o de permisos: 401, 403, 203, una página HTML de inicio
// de sesión en un 200 o una redirección (que no se sigue). Scope es el scope probable del PAT para la
// operación. No se reintenta.
type AuthError struct {
	Status    int
	Scope     string
	Operation string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("Azure DevOps rechazó la autenticación (HTTP %d) al %s: el PAT debe ser de una sola organización y llevar el scope %s; los PAT globales se retiran",
		e.Status, e.Operation, e.Scope)
}

func (e *AuthError) Is(target error) bool { return target == ErrAuth }

// NotFoundError indica un recurso inexistente (404). WorkItemID es el work item cuando lo hay.
type NotFoundError struct {
	Operation  string
	WorkItemID int
}

func (e *NotFoundError) Error() string {
	if e.WorkItemID != 0 {
		return fmt.Sprintf("Azure DevOps no encontró el work item %d al %s", e.WorkItemID, e.Operation)
	}
	return fmt.Sprintf("Azure DevOps no encontró el recurso al %s", e.Operation)
}

func (e *NotFoundError) Is(target error) bool { return target == ErrNotFound }

// RevConflictError indica que se rechazó la prueba test /rev de una modificación: el work item ya
// no está en la revisión leída. AfterRetry es true cuando la petición se reenvió tras un fallo de
// red: la modificación pudo aplicarse en un intento anterior.
type RevConflictError struct {
	WorkItemID int
	Rev        int
	AfterRetry bool
}

func (e *RevConflictError) Error() string {
	msg := fmt.Sprintf("el work item %d ya no está en la revisión %d: relee el work item antes de modificarlo", e.WorkItemID, e.Rev)
	if e.AfterRetry {
		msg += " (la modificación pudo aplicarse en un intento anterior cuya respuesta se perdió)"
	}
	return msg
}

func (e *RevConflictError) Is(target error) bool { return target == ErrRevConflict }

// RateLimitError indica el presupuesto de intentos agotado ante 429 o un Retry-After mayor que el
// techo de espera. Attempts es el número de peticiones enviadas.
type RateLimitError struct {
	Operation  string
	Attempts   int
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	msg := fmt.Sprintf("Azure DevOps limita la tasa de peticiones al %s: %d intento(s) enviado(s)", e.Operation, e.Attempts)
	if e.RetryAfter > 0 {
		msg += fmt.Sprintf("; pide esperar %s", e.RetryAfter.Round(time.Second))
	}
	return msg
}

func (e *RateLimitError) Is(target error) bool { return target == ErrRateLimited }

// TransportError es un fallo de red (conexión, timeout del intento o lectura del cuerpo) tras agotar
// los intentos. Message es el texto de la causa, redactado.
type TransportError struct {
	Operation string
	Attempts  int
	Message   string
	cause     error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("fallo de red hacia Azure DevOps al %s tras %d intento(s): %s", e.Operation, e.Attempts, e.Message)
}

func (e *TransportError) Unwrap() error        { return e.cause }
func (e *TransportError) Is(target error) bool { return target == ErrTransport }

// APIError es una respuesta de error de Azure DevOps que no tiene un tipo propio. Detail es el
// cuerpo de error acotado y redactado.
type APIError struct {
	Status    int
	Attempts  int
	Operation string
	Detail    string
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("Azure DevOps respondió HTTP %d al %s", e.Status, e.Operation)
	if e.Attempts > 1 {
		msg += fmt.Sprintf(" tras %d intento(s)", e.Attempts)
	}
	if e.Detail != "" {
		msg += ": " + e.Detail
	}
	return msg
}

func (e *APIError) Is(target error) bool {
	switch {
	case target == ErrBadRequest:
		return e.Status == 400
	case target == ErrUnavailable:
		return e.Status == 503
	case target == ErrServer:
		return e.Status >= 500 && e.Status != 503
	}
	return false
}

// invalidResponse construye el error de una respuesta de éxito que no se puede interpretar.
func invalidResponse(operation, why string) error {
	return fmt.Errorf("%w al %s: %s", ErrInvalidResponse, operation, why)
}

// scopes probables del PAT por familia de operación.
const (
	scopeWorkWrite = "vso.work_write" // work items, lotes, PATCH, tipos, estados y campos
	scopeWork      = "vso.work"       // iteraciones
	scopeProject   = "vso.project"    // proyectos, equipos y miembros
	scopeCode      = "vso.code"       // repositorios
)

// errorBody es el cuerpo de error JSON de Azure DevOps.
type errorBody struct {
	Message string `json:"message"`
	TypeKey string `json:"typeKey"`
}

// parseErrorBody interpreta el cuerpo como JSON de error de Azure DevOps. Si no lo es, el mensaje es
// el texto del cuerpo.
func parseErrorBody(body []byte) errorBody {
	var parsed errorBody
	if err := json.Unmarshal(body, &parsed); err == nil && (parsed.Message != "" || parsed.TypeKey != "") {
		return parsed
	}
	return errorBody{Message: string(body)}
}

const (
	testOperationTypeKey = "TestOperationFailedException"
	testOperationMessage = "test operation"
)

// isRevTestRejection indica si la respuesta es el rechazo de la prueba test /rev de un PATCH. El
// contrato exacto no está verificado contra Azure DevOps real (la exploración lo marca): las marcas
// viven aquí, en un único sitio, para corregirlas.
//
//   - 412 siempre.
//   - 400 o 409 cuando typeKey es TestOperationFailedException o el mensaje menciona «test
//     operation», sin distinguir mayúsculas.
//   - Cualquier otro 400 (por ejemplo TF51535: Cannot find field) no es un rechazo del test.
func isRevTestRejection(status int, body errorBody) bool {
	switch status {
	case 412:
		return true
	case 400, 409:
		return strings.EqualFold(body.TypeKey, testOperationTypeKey) ||
			strings.Contains(strings.ToLower(body.Message), testOperationMessage)
	}
	return false
}

// minRedactLen es la longitud mínima de un valor para redactarlo: uno más corto borraría texto
// legítimo (igual que minTrackedSecretLen en el marco).
const minRedactLen = 4

// redactor sustituye el PAT y sus formas codificadas por [oculto] en todo texto que entra en un
// error del cliente.
type redactor struct{ values []string }

// newRedactor construye el redactor de una petición: el PAT, base64(":"+PAT) y "Basic "+base64,
// ordenados del más largo al más corto para que la forma larga se redacte entera.
func newRedactor(pat string) redactor {
	b64 := base64.StdEncoding.EncodeToString([]byte(":" + pat))
	var values []string
	for _, v := range []string{pat, b64, "Basic " + b64} {
		if len(v) >= minRedactLen {
			values = append(values, v)
		}
	}
	sort.SliceStable(values, func(i, j int) bool { return len(values[i]) > len(values[j]) })
	return redactor{values: values}
}

func (r redactor) redact(s string) string {
	for _, v := range r.values {
		s = strings.ReplaceAll(s, v, "[oculto]")
	}
	return s
}

// maxLen es la longitud del valor más largo a redactar.
func (r redactor) maxLen() int {
	if len(r.values) == 0 {
		return 0
	}
	return len(r.values[0])
}

// detailOf prepara un texto del servidor para incluirlo en un error: redactado, en una sola línea y
// recortado a maxErrorDetail runas.
func (r redactor) detailOf(text string) string {
	flat := strings.Join(strings.Fields(r.redact(text)), " ")
	if utf8.RuneCountInString(flat) <= maxErrorDetail {
		return flat
	}
	runes := []rune(flat)
	return string(runes[:maxErrorDetail]) + "…"
}

// ForeignURLError indica que una URL de recurso (adjunto o imagen) no recibe la credencial: queda
// fuera del esquema, el host o la organización configurados, o lleva userinfo o segmentos «.» o «..».
// Reason es un valor cerrado (scheme, userinfo, host, organization, path); el error nunca repite la
// URL, que es texto del tracker. La decisión se toma sin red y sin resolver el PAT.
type ForeignURLError struct{ Reason string }

func (e *ForeignURLError) Error() string {
	return fmt.Sprintf("la URL del recurso queda fuera del host y la organización configurados (motivo: %s)", e.Reason)
}

// TooLargeError indica que el cuerpo de una descarga supera el límite fijado. Se detecta leyendo con
// un lector limitado, sin depender de Content-Length.
type TooLargeError struct {
	Operation string
	Limit     int64
}

func (e *TooLargeError) Error() string {
	return fmt.Sprintf("el cuerpo supera el límite de %d bytes al %s", e.Limit, e.Operation)
}

func (e *ForeignURLError) Is(target error) bool { return target == integrations.ErrResourceURLRejected }
func (e *TooLargeError) Is(target error) bool   { return target == integrations.ErrAttachmentTooLarge }
