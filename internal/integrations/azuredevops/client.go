package azuredevops

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

const (
	// DefaultEndpoint es la URL base de Azure DevOps Services. axiom.yaml no puede cambiarla.
	DefaultEndpoint = "https://dev.azure.com"
	// APIVersion es la versión de la API REST que usa el cliente en toda petición.
	APIVersion = "7.1"
	// DefaultTimeout es el timeout de cada intento, incluida la lectura del cuerpo.
	DefaultTimeout = 30 * time.Second
	// DefaultMaxAttempts es el presupuesto de intentos por petición lógica (1 + 3 reintentos).
	DefaultMaxAttempts = 4

	backoffBase       = 1 * time.Second  // 1 s, 2 s, 4 s...
	backoffCap        = 16 * time.Second // tope del retroceso exponencial
	maxRetryAfter     = 60 * time.Second // techo de espera por Retry-After
	maxResponseBytes  = 16 << 20         // cuerpo de una respuesta de éxito
	maxErrorBodyBytes = 64 << 10         // cuerpo de error que se lee para clasificar
	maxErrorDetail    = 512              // runas del detalle de error que se incluye

	userAgent     = "axiom"
	patSecretName = "pat"
)

// Clock es el reloj del cliente. Las pruebas lo sustituyen para no dormir.
type Clock interface {
	Now() time.Time
	// Sleep espera d y devuelve ctx.Err() si ctx vence antes.
	Sleep(ctx context.Context, d time.Duration) error
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

func (realClock) Sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Options configura el plugin y los clientes que construye. El valor cero es el de producción.
type Options struct {
	// Endpoint es la URL base del servicio. Solo para pruebas: por defecto DefaultEndpoint y
	// axiom.yaml no puede cambiarla.
	Endpoint string
	// HTTPClient es el cliente HTTP. Por defecto uno nuevo con el transporte por defecto (con el proxy
	// del entorno). El cliente no lo modifica: trabaja sobre una copia que no sigue redirecciones.
	HTTPClient *http.Client
	// Timeout es el timeout de cada intento. Por defecto DefaultTimeout.
	Timeout time.Duration
	// MaxAttempts es el presupuesto de intentos por petición. Por defecto DefaultMaxAttempts.
	MaxAttempts int
	// Clock es el reloj de las esperas. Por defecto el reloj real.
	Clock Clock
}

// RateLimitInfo son las últimas cabeceras X-RateLimit-* leídas. Son informativas: un valor ilegible
// o ausente queda a cero. X-RateLimit-Resource no se lee ni interviene en ningún cálculo.
type RateLimitInfo struct {
	Delay     time.Duration // X-RateLimit-Delay: el retraso que el servidor ya aplicó
	Remaining int           // X-RateLimit-Remaining
	Limit     int           // X-RateLimit-Limit
	Reset     time.Time     // X-RateLimit-Reset (epoch Unix)
}

// orgState es el estado compartido por organización durante la vida del proceso: el ritmo marcado
// por Retry-After, las últimas cabeceras de tasa y la caché de repositorios.
type orgState struct {
	mu        sync.Mutex
	notBefore time.Time
	rate      RateLimitInfo
	repos     map[string][]Repository // caché por proyecto; solo éxitos
}

// Client es el cliente REST de Azure DevOps 7.1 para una organización, un proyecto y un equipo. Todo
// el tráfico pasa por un único punto (do): autenticación, timeout por intento, espera, reintentos,
// clasificación de respuestas y saneado de errores.
type Client struct {
	endpoint    *url.URL
	http        *http.Client
	timeout     time.Duration
	maxAttempts int
	clock       Clock
	target      Target
	secrets     integrations.SecretSource
	state       *orgState
}

// NewClient construye un cliente ligado al estado de la organización del plugin: el ritmo, las
// cabeceras de tasa y la caché de repositorios se comparten entre llamadas durante la vida del
// proceso. cfg es la configuración de integrations.azure-devops. El PAT se resuelve en cada petición
// con src, nunca al construir el cliente.
func (p *Plugin) NewClient(cfg map[string]any, src integrations.SecretSource) (*Client, error) {
	target, err := TargetFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	if err := target.validate(); err != nil {
		return nil, err
	}
	client, err := NewClient(p.opts, target, src)
	if err != nil {
		return nil, err
	}
	client.state = p.session(target.Organization)
	return client, nil
}

// NewClient construye un cliente con su propio estado de organización (ritmo, cabeceras de tasa y
// caché de repositorios). Es la entrada pública de quien no pasa por un Plugin y la que usa
// (*Plugin).NewClient, que además le asigna el estado compartido de la organización. src resuelve el
// secreto pat en cada petición.
func NewClient(opts Options, target Target, src integrations.SecretSource) (*Client, error) {
	if src == nil {
		return nil, fmt.Errorf("%w: falta la fuente de secretos", ErrConfig)
	}
	if err := target.validate(); err != nil {
		return nil, err
	}
	endpoint := opts.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	base, err := url.Parse(endpoint)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" || base.RawQuery != "" || base.Fragment != "" || base.User != nil {
		return nil, fmt.Errorf("%w: la URL base %q no es una URL http(s) válida", ErrConfig, endpoint)
	}
	hc := &http.Client{}
	if opts.HTTPClient != nil {
		copied := *opts.HTTPClient
		hc = &copied
	}
	// Azure DevOps redirige al inicio de sesión cuando la credencial no vale: seguir la redirección
	// convertiría un fallo de autenticación en un 200 con HTML.
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	c := &Client{
		endpoint:    base,
		http:        hc,
		timeout:     opts.Timeout,
		maxAttempts: opts.MaxAttempts,
		clock:       opts.Clock,
		target:      target,
		secrets:     src,
		state:       &orgState{},
	}
	if c.timeout <= 0 {
		c.timeout = DefaultTimeout
	}
	if c.maxAttempts <= 0 {
		c.maxAttempts = DefaultMaxAttempts
	}
	if c.clock == nil {
		c.clock = realClock{}
	}
	return c, nil
}

// Target devuelve el destino del cliente.
func (c *Client) Target() Target { return c.target }

// Endpoint devuelve la URL base efectiva (sin la organización).
func (c *Client) Endpoint() string { return c.endpoint.String() }

// RateLimit devuelve las últimas cabeceras X-RateLimit-* leídas para la organización.
func (c *Client) RateLimit() RateLimitInfo {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	return c.state.rate
}

// request describe una petición lógica.
type request struct {
	method      string
	contentType string
	segments    []string // tras la organización, sin escapar
	query       url.Values
	body        []byte
	scope       string // scope probable del PAT para el error de autenticación
	operation   string // texto de los errores: "leer el work item 42"
	retryable   bool   // GET, POST workitemsbatch y PATCH con test /rev
	revPatch    bool   // activa la clasificación del rechazo de test /rev
	workItemID  int    // para NotFound y RevConflict
	rev         int    // revisión enviada en test /rev
	batchIDs    []int  // identificadores de un lote: un 404 nombra el que falta
}

// buildURL compone la URL: [organización, segmentos...] escapados uno a uno con url.PathEscape. Nunca
// se concatena sin escapar ni se usa JoinPath, que interpretaría la «/» de un nombre como separador.
// Un segmento vacío, «.» o «..» es un error de configuración: ninguna petición sale de
// {endpoint}/{organización}/.
func (c *Client) buildURL(req request) (string, error) {
	segments := append([]string{c.target.Organization}, req.segments...)
	rawParts := make([]string, 0, len(segments))
	escapedParts := make([]string, 0, len(segments))
	for _, s := range segments {
		if err := validateSegment(s); err != nil {
			return "", err
		}
		rawParts = append(rawParts, s)
		escapedParts = append(escapedParts, url.PathEscape(s))
	}
	u := *c.endpoint
	basePath := strings.TrimRight(c.endpoint.Path, "/")
	baseEscaped := strings.TrimRight(c.endpoint.EscapedPath(), "/")
	u.Path = basePath + "/" + strings.Join(rawParts, "/")
	u.RawPath = baseEscaped + "/" + strings.Join(escapedParts, "/")
	query := url.Values{}
	for k, vs := range req.query {
		for _, v := range vs {
			query.Add(k, v)
		}
	}
	query.Set("api-version", APIVersion)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

// validateSegment rechaza los segmentos de ruta que podrían sacar la petición de la organización.
func validateSegment(s string) error {
	switch strings.TrimSpace(s) {
	case "", ".", "..":
		return fmt.Errorf("%w: el segmento de ruta %q no es válido (vacío, «.» o «..»)", ErrConfig, s)
	}
	return nil
}

// validate comprueba la organización, el proyecto y el equipo.
func (t Target) validate() error {
	for _, field := range []struct{ name, value string }{
		{"organization", t.Organization}, {"project", t.Project}, {"team", t.Team},
	} {
		if err := validateSegment(field.value); err != nil {
			return fmt.Errorf("integrations.azure-devops.%s: %w", field.name, err)
		}
	}
	return nil
}

// attemptResult es lo que devuelve un intento HTTP.
type attemptResult struct {
	status      int
	header      http.Header
	contentType string
	body        []byte
	truncated   bool
	err         error // fallo de transporte (conexión, timeout del intento, lectura del cuerpo)
}

// do ejecuta una petición lógica y decodifica la respuesta de éxito en out. Secuencia:
//
//  1. Resuelve el PAT una vez por petición lógica (los reintentos lo reutilizan). Sin PAT no hay HTTP.
//  2. Respeta el ritmo marcado por un Retry-After anterior.
//  3. Envía el intento con su propio timeout.
//  4. Lee las cabeceras de tasa y Retry-After de cualquier respuesta.
//  5. Reintenta solo 429, 503 y errores de red, con retroceso exponencial con tope y sin esperar
//     menos que Retry-After; si Retry-After supera el techo, no reintenta.
//  6. Clasifica el resto y sanea los errores con el redactor del PAT.
func (c *Client) do(ctx context.Context, req request, out any) error {
	secret, err := c.secrets.Secret(ctx, patSecretName)
	if err != nil {
		return err
	}
	pat := secret.Reveal()
	if pat == "" {
		return &secrets.MissingError{Ref: secrets.Ref{Plugin: manifest.AzureDevOpsID, Name: patSecretName}}
	}
	red := newRedactor(pat)
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(":"+pat))

	target, err := c.buildURL(req)
	if err != nil {
		return err
	}

	attempts := 0
	for {
		if err := c.pace(ctx, req, attempts); err != nil {
			return err
		}
		attempts++
		res := c.attempt(ctx, req, target, auth, red)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("%s: %w", req.operation, ctxErr)
		}

		if res.err != nil {
			if !req.retryable || attempts >= c.maxAttempts {
				return &TransportError{Operation: req.operation, Attempts: attempts, Message: red.detailOf(res.err.Error()), cause: res.err}
			}
			if err := c.clock.Sleep(ctx, backoff(attempts)); err != nil {
				return fmt.Errorf("%s: %w", req.operation, err)
			}
			continue
		}

		retryAfter, hasRetryAfter := c.observe(res.header)
		if res.status == http.StatusTooManyRequests || res.status == http.StatusServiceUnavailable {
			if !req.retryable || attempts >= c.maxAttempts || (hasRetryAfter && retryAfter > maxRetryAfter) {
				return exhausted(req, res.status, attempts, retryAfter, red, res)
			}
			wait := backoff(attempts)
			if retryAfter > wait {
				wait = retryAfter
			}
			if err := c.clock.Sleep(ctx, wait); err != nil {
				return fmt.Errorf("%s: %w", req.operation, err)
			}
			continue
		}
		return c.classify(req, res, attempts, red, out)
	}
}

// backoff es el retroceso exponencial tras el intento n (1 s, 2 s, 4 s...) con tope.
func backoff(n int) time.Duration {
	d := backoffBase
	for i := 1; i < n; i++ {
		d *= 2
		if d >= backoffCap {
			return backoffCap
		}
	}
	return d
}

// pace espera hasta el instante marcado por un Retry-After anterior. Una espera mayor que el techo
// no se hace: se devuelve RateLimitError sin enviar nada.
func (c *Client) pace(ctx context.Context, req request, attempts int) error {
	c.state.mu.Lock()
	notBefore := c.state.notBefore
	c.state.mu.Unlock()
	wait := notBefore.Sub(c.clock.Now())
	if wait <= 0 {
		return nil
	}
	if wait > maxRetryAfter {
		return &RateLimitError{Operation: req.operation, Attempts: attempts, RetryAfter: wait}
	}
	if err := c.clock.Sleep(ctx, wait); err != nil {
		return fmt.Errorf("%s: %w", req.operation, err)
	}
	return nil
}

// attempt envía un intento con su propio timeout, que cubre Do y la lectura del cuerpo.
func (c *Client) attempt(ctx context.Context, req request, target, auth string, red redactor) attemptResult {
	attemptCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var body io.Reader
	if len(req.body) > 0 {
		body = bytes.NewReader(req.body)
	}
	httpReq, err := http.NewRequestWithContext(attemptCtx, req.method, target, body)
	if err != nil {
		return attemptResult{err: errors.New("no se pudo construir la petición")}
	}
	httpReq.Header.Set("Authorization", auth)
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", userAgent)
	if req.contentType != "" {
		httpReq.Header.Set("Content-Type", req.contentType)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return attemptResult{err: err}
	}
	defer resp.Body.Close()

	res := attemptResult{status: resp.StatusCode, header: resp.Header, contentType: resp.Header.Get("Content-Type")}
	// Un 404 de un lote se lee para nombrar el work item que falta.
	if !readsBody(resp.StatusCode) && !(resp.StatusCode == http.StatusNotFound && len(req.batchIDs) > 0) {
		return res
	}
	limit := int64(maxErrorBodyBytes)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		limit = maxResponseBytes
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return attemptResult{err: err}
	}
	if int64(len(data)) > limit {
		res.truncated = true
		data = data[:limit]
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return res // cuerpo de éxito demasiado grande: respuesta inválida
		}
		// Un PAT cortado por el límite no se podría redactar entero: se descarta el final.
		if cut := len(data) - red.maxLen(); cut > 0 {
			data = data[:cut]
		}
	}
	res.body = data
	return res
}

// readsBody indica si hace falta el cuerpo para clasificar la respuesta: los 203, las redirecciones,
// los rechazos de autenticación y el 404 no lo interpretan.
func readsBody(status int) bool {
	switch {
	case status == http.StatusNonAuthoritativeInfo, status == http.StatusNoContent:
		return false
	case status >= 300 && status < 400:
		return false
	case status == http.StatusUnauthorized, status == http.StatusForbidden, status == http.StatusNotFound:
		return false
	}
	return true
}

// observe lee de una respuesta las cabeceras de tasa y Retry-After. Retry-After de cualquier
// respuesta, incluida 200, marca el instante antes del cual no se envía otra petición.
// X-RateLimit-Resource no se lee.
func (c *Client) observe(header http.Header) (retryAfter time.Duration, ok bool) {
	now := c.clock.Now()
	retryAfter, ok = parseRetryAfter(header.Get("Retry-After"), now)

	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	// Solo se actualiza lo que la respuesta trae: una respuesta sin cabeceras no borra lo leído.
	if v, present := parseHeaderFloat(header, "X-RateLimit-Delay"); present {
		c.state.rate.Delay = time.Duration(v * float64(time.Second))
	}
	if v, present := parseHeaderFloat(header, "X-RateLimit-Remaining"); present {
		c.state.rate.Remaining = int(v)
	}
	if v, present := parseHeaderFloat(header, "X-RateLimit-Limit"); present {
		c.state.rate.Limit = int(v)
	}
	if v, present := parseHeaderFloat(header, "X-RateLimit-Reset"); present && v > 0 {
		c.state.rate.Reset = time.Unix(int64(v), 0)
	}
	if ok && retryAfter > 0 {
		if until := now.Add(retryAfter); until.After(c.state.notBefore) {
			c.state.notBefore = until
		}
	}
	return retryAfter, ok
}

// parseHeaderFloat lee una cabecera numérica decimal. present es false si falta o es ilegible.
func parseHeaderFloat(header http.Header, name string) (value float64, present bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(header.Get(name)), 64)
	if err != nil || f < 0 {
		return 0, false
	}
	return f, true
}

// parseRetryAfter acepta segundos enteros o una fecha HTTP (relativa a now).
func parseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	// ParseInt devuelve el extremo del rango con ErrRange: un número fuera de
	// int64 se trata como su extremo en lugar de como un valor ilegible.
	if secs, err := strconv.ParseInt(value, 10, 64); err == nil || errors.Is(err, strconv.ErrRange) {
		if secs < 0 {
			return 0, false
		}
		// Satura en lugar de desbordar: un valor desorbitado nunca puede volverse
		// negativo ni pequeño y saltarse el techo de maxRetryAfter.
		if secs > int64(math.MaxInt64/int64(time.Second)) {
			return time.Duration(math.MaxInt64), true
		}
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(value); err == nil {
		d := t.Sub(now)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

// exhausted construye el error de un 429 o 503 que no se reintenta (presupuesto agotado o espera
// excesiva).
func exhausted(req request, status, attempts int, retryAfter time.Duration, red redactor, res attemptResult) error {
	if status == http.StatusTooManyRequests {
		return &RateLimitError{Operation: req.operation, Attempts: attempts, RetryAfter: retryAfter}
	}
	return &APIError{Status: status, Attempts: attempts, Operation: req.operation, Detail: red.detailOf(parseErrorBody(res.body).Message)}
}

// classify convierte una respuesta que no se reintenta en un valor o en un error tipado (D-05).
func (c *Client) classify(req request, res attemptResult, attempts int, red redactor, out any) error {
	status := res.status
	switch {
	case status == http.StatusNonAuthoritativeInfo, status >= 300 && status < 400,
		status == http.StatusUnauthorized, status == http.StatusForbidden:
		return &AuthError{Status: status, Scope: req.scope, Operation: req.operation}

	case status == http.StatusNoContent:
		return nil

	case status >= 200 && status < 300:
		return decodeSuccess(req, res, out)

	case status == http.StatusNotFound:
		id := req.workItemID
		if id == 0 && len(req.batchIDs) > 0 {
			id = missingIDFromMessage(red.redact(parseErrorBody(res.body).Message), req.batchIDs)
		}
		return &NotFoundError{Operation: req.operation, WorkItemID: id}
	}

	body := parseErrorBody(res.body)
	if req.revPatch && isRevTestRejection(status, body) {
		return &RevConflictError{WorkItemID: req.workItemID, Rev: req.rev, AfterRetry: attempts > 1}
	}
	return &APIError{Status: status, Attempts: attempts, Operation: req.operation, Detail: red.detailOf(body.Message)}
}

// decodeSuccess decodifica una respuesta 2xx. Criterio de la página de inicio de sesión: un 200 con
// Content-Type text/html (o xhtml) es la página de login y se trata como fallo de autenticación sin
// interpretar el cuerpo; cualquier otro cuerpo que no sea JSON válido es ErrInvalidResponse, nunca un
// valor vacío.
func decodeSuccess(req request, res attemptResult, out any) error {
	if mediaType, _, err := mime.ParseMediaType(res.contentType); err == nil &&
		(mediaType == "text/html" || mediaType == "application/xhtml+xml") {
		return &AuthError{Status: res.status, Scope: req.scope, Operation: req.operation}
	}
	if res.truncated {
		return invalidResponse(req.operation, "el cuerpo supera el tamaño máximo admitido")
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(res.body, out); err != nil {
		return invalidResponse(req.operation, "el cuerpo no es JSON válido")
	}
	return nil
}
