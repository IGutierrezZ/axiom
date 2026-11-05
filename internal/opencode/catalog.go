package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	catalogTimeout = 15 * time.Second
	// maxCatalogOutput es el techo de bytes que se aceptan de stdout al
	// descubrir el catálogo. Un catálogo real con muchos proveedores supera con
	// holgura 1 MiB; 16 MiB sigue acotando la memoria ante una salida desbocada.
	maxCatalogOutput = 16 << 20
	// catalogWaitDelay acota cuánto espera Wait a que se cierren las tuberías
	// cuando un descendiente huérfano las mantiene abiertas.
	catalogWaitDelay = 2 * time.Second
	// catalogKillGrace es el margen que se da a un hijo cancelado antes de
	// cerrar a la fuerza su tubería de stdout.
	catalogKillGrace = 250 * time.Millisecond
)

// Command describes the bounded OpenCode command used for runtime discovery.
type Command struct {
	Path        string
	Args        []string
	Dir         string
	OutputLimit int
	// Env son variables añadidas al entorno del hijo (formato KEY=VALUE).
	Env []string
}

type CommandOutput struct{ Stdout, Stderr []byte }
type CommandRunner func(context.Context, Command) (CommandOutput, error)

// CatalogRunner ejecuta el comando de descubrimiento y devuelve un lector sobre
// su salida estándar, que se consume en streaming. Si el lector implementa
// io.Closer se cierra siempre; si implementa catalogExit se consulta cómo
// terminó el hijo. Un lector sobre un fichero temporal cumple el mismo contrato
// sin tocar el analizador.
type CatalogRunner func(context.Context, Command) (io.Reader, error)

// catalogExit permite al descubrimiento honrar el estado de salida del hijo.
type catalogExit interface {
	// Drained indica que stdout llegó a su fin (EOF) sin cortarse antes.
	Drained() bool
	// WaitError espera al hijo y devuelve su error de salida, si lo hubo.
	WaitError() error
}

type CatalogErrorKind string

const (
	CatalogErrorMissingBinary     CatalogErrorKind = "missing_binary"
	CatalogErrorTimeout           CatalogErrorKind = "timeout"
	CatalogErrorCommandFailed     CatalogErrorKind = "command_failed"
	CatalogErrorOutputTooLarge    CatalogErrorKind = "output_too_large"
	CatalogErrorMalformed         CatalogErrorKind = "malformed_output"
	CatalogErrorUnsupportedSchema CatalogErrorKind = "unsupported_schema"
)

// CatalogError intentionally exposes a category, never command output.
type CatalogError struct{ Kind CatalogErrorKind }

func (e *CatalogError) Error() string { return string(e.Kind) }

// DiscoverCatalog reads OpenCode's effective provider catalog for projectDir.
// La versión mayor de OpenCode decide el origen: V2 consulta la API de modelos
// y captura su salida en un fichero temporal; V1 lee `models --verbose` por
// tubería. Ver discoverCatalog para la política ante una versión no detectable.
func DiscoverCatalog(ctx context.Context, projectDir string) (map[string]Provider, error) {
	return discoverCatalog(ctx, projectDir, DetectRuntimeMajor, streamCatalogCommand, fileCatalogCommand)
}

// discoverCatalog elige el transporte según la versión mayor detectada. Si la
// detección falla (binario ausente, versión ilegible o prerrelease) se usa V1 de
// forma conservadora: es el comportamiento previo y, si el binario falta, el
// propio `models --verbose` devuelve el error tipado honesto (missing_binary)
// en lugar de enmascararlo con un fallo de detección. Una V2 mal reconocida
// como V1 acaba en un CatalogError visible, nunca en un catálogo falso. La
// detección va fuera del plazo del catálogo y tiene el suyo propio.
func discoverCatalog(ctx context.Context, projectDir string, detect func(context.Context) (RuntimeMajor, error), v1, v2 CatalogRunner) (map[string]Provider, error) {
	major, err := detect(ctx)
	if err != nil {
		major = RuntimeV1
	}
	if major == RuntimeV2 {
		return discoverCatalogForMajor(ctx, projectDir, RuntimeV2, v2)
	}
	return discoverCatalogForMajor(ctx, projectDir, RuntimeV1, v1)
}

// DiscoverCatalogWithRunner analiza el catálogo V1 (`models --verbose`) en
// streaming y clasifica el resultado. Se aplican por este orden: el
// desbordamiento del techo, el vencimiento del plazo, el estado de salida del
// hijo (cuando su salida llegó a su fin) y, por último, el error de análisis.
// Así un hijo que muere con código distinto de cero no se disfraza de JSON
// inválido.
func DiscoverCatalogWithRunner(ctx context.Context, projectDir string, runner CatalogRunner) (map[string]Provider, error) {
	return discoverCatalogForMajor(ctx, projectDir, RuntimeV1, runner)
}

func discoverCatalogForMajor(ctx context.Context, projectDir string, major RuntimeMajor, runner CatalogRunner) (map[string]Provider, error) {
	ctx, cancel := context.WithTimeout(ctx, catalogTimeout)
	defer cancel()
	reader, err := runner(ctx, catalogCommand(major, projectDir))
	if err != nil {
		return nil, catalogCommandError(ctx, err)
	}
	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}
	limited := &ceilingReader{r: reader, limit: maxCatalogOutput, cancel: cancel}
	var providers map[string]Provider
	var parseErr error
	if major == RuntimeV2 {
		providers, parseErr = parseV2ModelAPI(limited)
	} else {
		providers, parseErr = parseVerboseCatalog(limited)
	}
	if limited.exceeded {
		return nil, &CatalogError{Kind: CatalogErrorOutputTooLarge}
	}
	if ctx.Err() != nil {
		return nil, &CatalogError{Kind: CatalogErrorTimeout}
	}
	if exit, ok := reader.(catalogExit); ok && (parseErr == nil || exit.Drained()) {
		if waitErr := exit.WaitError(); waitErr != nil {
			return nil, catalogCommandError(ctx, waitErr)
		}
	}
	if parseErr != nil {
		return nil, parseErr
	}
	return providers, nil
}

// ceilingReader corta la lectura al superar limit bytes y cancela el contexto
// para que el hijo deje de producir salida.
type ceilingReader struct {
	r        io.Reader
	limit    int64
	read     int64
	cancel   context.CancelFunc
	exceeded bool
}

func (c *ceilingReader) Read(p []byte) (int, error) {
	if c.exceeded {
		return 0, &CatalogError{Kind: CatalogErrorOutputTooLarge}
	}
	n, err := c.r.Read(p)
	c.read += int64(n)
	if c.read > c.limit {
		c.exceeded = true
		c.cancel()
		return 0, &CatalogError{Kind: CatalogErrorOutputTooLarge}
	}
	return n, err
}

// MergeConfiguredCatalog augments the runtime catalog with file-backed config
// data without replacing runtime-discovered providers or models.
func MergeConfiguredCatalog(runtimeCatalog, configuredCatalog map[string]Provider) map[string]Provider {
	merged := cloneCatalog(runtimeCatalog)
	for providerID, configuredProvider := range configuredCatalog {
		runtimeProvider, exists := merged[providerID]
		if !exists {
			merged[providerID] = cloneProvider(configuredProvider)
			continue
		}
		if runtimeProvider.URL == "" {
			runtimeProvider.URL = configuredProvider.URL
		}
		if runtimeProvider.Models == nil {
			runtimeProvider.Models = map[string]Model{}
		}
		for modelID, configuredModel := range configuredProvider.Models {
			if _, exists := runtimeProvider.Models[modelID]; !exists {
				runtimeProvider.Models[modelID] = configuredModel
			}
		}
		merged[providerID] = runtimeProvider
	}
	return merged
}

func cloneCatalog(catalog map[string]Provider) map[string]Provider {
	clone := make(map[string]Provider, len(catalog))
	for id, provider := range catalog {
		clone[id] = cloneProvider(provider)
	}
	return clone
}

func cloneProvider(provider Provider) Provider {
	provider.Models = cloneModels(provider.Models)
	return provider
}

func cloneModels(models map[string]Model) map[string]Model {
	clone := make(map[string]Model, len(models))
	for id, model := range models {
		clone[id] = model
	}
	return clone
}

func catalogCommandError(ctx context.Context, err error) error {
	var catalogErr *CatalogError
	if errors.As(err, &catalogErr) {
		return catalogErr
	}
	if ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) {
		return &CatalogError{Kind: CatalogErrorTimeout}
	}
	var execErr *exec.Error
	if errors.As(err, &execErr) && (errors.Is(execErr.Err, os.ErrNotExist) || errors.Is(execErr.Err, exec.ErrNotFound)) {
		return &CatalogError{Kind: CatalogErrorMissingBinary}
	}
	return &CatalogError{Kind: CatalogErrorCommandFailed}
}

// parseVerboseCatalog analiza la salida de `opencode models --verbose` en
// streaming, un registro cada vez: una línea de cabecera `proveedor/modelo`
// seguida de un objeto JSON. Las líneas de registro de plugins y hooks se
// toleran como ruido, pero una salida que solo contiene ruido falla cerrada.
func parseVerboseCatalog(r io.Reader) (map[string]Provider, error) {
	providers := map[string]Provider{}
	sawNoise := false
	stream := &recordStream{r: r}
	for {
		line, err := stream.readLine()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, streamReadError(err)
		}
		header := strings.TrimSpace(line)
		if !isCatalogHeader(header) {
			// Plugins y hooks (p. ej. el plugin gestionado de skill-registry
			// fuera de la raíz de un proyecto) pueden escribir en stdout antes o
			// entre los registros. Se saltan esas líneas, recordando que las
			// hubo para que una salida solo de ruido siga fallando cerrada.
			sawNoise = sawNoise || header != ""
			continue
		}
		next, err := stream.peekNonSpace()
		if errors.Is(err, io.EOF) {
			sawNoise = true
			break
		}
		if err != nil {
			return nil, streamReadError(err)
		}
		if next != '{' {
			// Línea con forma de cabecera (ruta, URL) sin JSON detrás.
			sawNoise = true
			continue
		}
		var raw struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			Family       string `json:"family"`
			Capabilities *struct {
				ToolCall  *bool `json:"toolcall"`
				Reasoning *bool `json:"reasoning"`
			} `json:"capabilities"`
			Limit    ModelLimit                 `json:"limit"`
			Cost     ModelCost                  `json:"cost"`
			Variants map[string]json.RawMessage `json:"variants"`
		}
		if err := stream.decode(&raw); err != nil {
			var catalogErr *CatalogError
			if errors.As(err, &catalogErr) {
				return nil, catalogErr
			}
			var typeErr *json.UnmarshalTypeError
			if errors.As(err, &typeErr) {
				return nil, &CatalogError{Kind: CatalogErrorUnsupportedSchema}
			}
			return nil, &CatalogError{Kind: CatalogErrorMalformed}
		}
		if raw.ID == "" || raw.Capabilities == nil || raw.Capabilities.ToolCall == nil || !strings.HasSuffix(header, "/"+raw.ID) {
			return nil, &CatalogError{Kind: CatalogErrorUnsupportedSchema}
		}
		providerID := strings.TrimSuffix(header, "/"+raw.ID)
		if providerID == "" {
			return nil, &CatalogError{Kind: CatalogErrorUnsupportedSchema}
		}
		provider := providers[providerID]
		if provider.ID == "" {
			provider = Provider{ID: providerID, Name: providerID, Models: map[string]Model{}}
		}
		variants := make([]string, 0, len(raw.Variants))
		for key := range raw.Variants {
			variants = append(variants, key)
		}
		sortVariants(variants)
		reasoning := raw.Capabilities.Reasoning != nil && *raw.Capabilities.Reasoning
		provider.Models[raw.ID] = Model{ID: raw.ID, Name: raw.Name, Family: raw.Family, ToolCall: *raw.Capabilities.ToolCall, Reasoning: reasoning, Limit: raw.Limit, Cost: raw.Cost, Variants: variants}
		providers[providerID] = provider
	}
	if len(providers) == 0 && sawNoise {
		return nil, &CatalogError{Kind: CatalogErrorUnsupportedSchema}
	}
	return providers, nil
}

// streamReadError traduce un fallo de lectura del flujo: el techo de salida
// conserva su categoría y cualquier otro corte se trata como salida inválida.
func streamReadError(err error) error {
	var catalogErr *CatalogError
	if errors.As(err, &catalogErr) {
		return catalogErr
	}
	return &CatalogError{Kind: CatalogErrorMalformed}
}

// recordStream lee el flujo línea a línea y deja que un json.Decoder consuma el
// cuerpo de un registro. Los bytes que el decodificador leyó de más se devuelven
// al búfer para no perder la cabecera del registro siguiente.
type recordStream struct {
	r   io.Reader
	buf []byte
	eof bool
}

// fill añade un bloque del flujo al búfer. Devuelve io.EOF solo cuando el flujo
// se agotó y no queda nada pendiente.
func (s *recordStream) fill() error {
	if s.eof {
		return io.EOF
	}
	var chunk [8192]byte
	n, err := s.r.Read(chunk[:])
	s.buf = append(s.buf, chunk[:n]...)
	if err == nil {
		return nil
	}
	if errors.Is(err, io.EOF) {
		s.eof = true
		if len(s.buf) == 0 {
			return io.EOF
		}
		return nil
	}
	return err
}

// readLine devuelve la siguiente línea; admite una última línea sin salto.
func (s *recordStream) readLine() (string, error) {
	for {
		if idx := bytes.IndexByte(s.buf, '\n'); idx >= 0 {
			line := string(s.buf[:idx])
			s.buf = s.buf[idx+1:]
			return line, nil
		}
		if err := s.fill(); err != nil {
			if errors.Is(err, io.EOF) && len(s.buf) > 0 {
				line := string(s.buf)
				s.buf = nil
				return line, nil
			}
			return "", err
		}
	}
}

// peekNonSpace descarta los blancos y devuelve el siguiente byte útil sin
// consumirlo.
func (s *recordStream) peekNonSpace() (byte, error) {
	for {
		for i, b := range s.buf {
			if b != ' ' && b != '\t' && b != '\r' && b != '\n' {
				s.buf = s.buf[i:]
				return b, nil
			}
		}
		s.buf = nil
		if err := s.fill(); err != nil {
			return 0, err
		}
	}
}

// Read sirve primero el búfer y después el flujo, para que el decodificador
// consuma el registro sin ver la lectura adelantada.
func (s *recordStream) Read(p []byte) (int, error) {
	if len(s.buf) > 0 {
		n := copy(p, s.buf)
		s.buf = s.buf[n:]
		return n, nil
	}
	if s.eof {
		return 0, io.EOF
	}
	n, err := s.r.Read(p)
	if errors.Is(err, io.EOF) {
		s.eof = true
	}
	return n, err
}

// decode lee un objeto JSON y recupera lo que el decodificador leyó de más.
func (s *recordStream) decode(v any) error {
	dec := json.NewDecoder(s)
	if err := dec.Decode(v); err != nil {
		return err
	}
	rest, err := io.ReadAll(dec.Buffered())
	if err != nil {
		return err
	}
	s.buf = append(rest, s.buf...)
	return nil
}

// isCatalogHeader reports whether line has the `provider/model` header shape:
// non-empty, no whitespace, at least one slash, and not the start of a JSON
// block. Anything else on stdout is log noise, never a catalog record.
func isCatalogHeader(line string) bool {
	return line != "" && !strings.ContainsAny(line, " \t") && strings.Contains(line, "/") && line[0] != '{'
}

// effortRank orders the known reasoning-effort variant names semantically.
var effortRank = map[string]int{"low": 0, "medium": 1, "high": 2}

// sortVariants emits low/medium/high semantic order when every variant key is
// a known effort name, falling back to lexical order for anything else.
func sortVariants(variants []string) {
	for _, variant := range variants {
		if _, known := effortRank[variant]; !known {
			sort.Strings(variants)
			return
		}
	}
	sort.Slice(variants, func(i, j int) bool { return effortRank[variants[i]] < effortRank[variants[j]] })
}

// streamCatalogCommand lanza el comando y devuelve un lector sobre su stdout.
// Stderr se drena en la gorutina que crea exec hacia io.Discard: CatalogError
// nunca expone la salida del comando, así que el tope de stderr es cero bytes
// retenidos y un hijo charlatán no puede llenar su tubería ni bloquear el
// descubrimiento. El lector devuelto debe cerrarse: Close cancela el proceso y
// lo recoge.
func streamCatalogCommand(ctx context.Context, command Command) (io.Reader, error) {
	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, command.Path, command.Args...)
	cmd.Dir = command.Dir
	if len(command.Env) > 0 {
		cmd.Env = append(os.Environ(), command.Env...)
	}
	afterStart, release := configureProcessGroup(cmd)
	if cmd.WaitDelay == 0 {
		cmd.WaitDelay = catalogWaitDelay
	}
	cmd.Stderr = io.Discard
	stdout, err := cmd.StdoutPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		cancel()
		if release != nil {
			release()
		}
		return nil, err
	}
	if afterStart != nil {
		afterStart()
	}
	// Si el contexto vence y un descendiente huérfano conserva la tubería, se
	// cierra a la fuerza tras un margen para que la lectura no quede colgada.
	go func() {
		<-ctx.Done()
		time.Sleep(catalogKillGrace)
		_ = stdout.Close()
	}()
	return &processStream{cmd: cmd, stdout: stdout, cancel: cancel, release: release}, nil
}

// processStream expone stdout de un proceso en marcha como io.Reader y recuerda
// cómo terminó el hijo.
type processStream struct {
	cmd     *exec.Cmd
	stdout  io.ReadCloser
	cancel  context.CancelFunc
	release func()
	drained bool

	once    sync.Once
	waitErr error
}

func (p *processStream) Read(b []byte) (int, error) {
	n, err := p.stdout.Read(b)
	if errors.Is(err, io.EOF) {
		p.drained = true
	}
	return n, err
}

func (p *processStream) Drained() bool { return p.drained }

func (p *processStream) wait() {
	p.once.Do(func() {
		p.waitErr = p.cmd.Wait()
		if p.release != nil {
			p.release()
		}
	})
}

// WaitError espera a que el hijo termine y devuelve su error de salida.
func (p *processStream) WaitError() error {
	p.wait()
	return p.waitErr
}

// Close cancela el proceso si sigue vivo y lo recoge.
func (p *processStream) Close() error {
	p.cancel()
	p.wait()
	return nil
}

func runCatalogCommand(ctx context.Context, command Command) (CommandOutput, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, command.Path, command.Args...)
	cmd.Dir = command.Dir
	afterStart, release := configureProcessGroup(cmd)
	if release != nil {
		defer release()
	}
	stdout, stderr := &limitedBuffer{limit: command.OutputLimit, cancel: cancel}, &limitedBuffer{limit: command.OutputLimit, cancel: cancel}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Start(); err != nil {
		return CommandOutput{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, err
	}
	if afterStart != nil {
		afterStart()
	}
	err := cmd.Wait()
	if stdout.overflow || stderr.overflow {
		return CommandOutput{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, &CatalogError{Kind: CatalogErrorOutputTooLarge}
	}
	return CommandOutput{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, err
}

type limitedBuffer struct {
	buffer   bytes.Buffer
	limit    int
	overflow bool
	cancel   context.CancelFunc
}

func (b *limitedBuffer) Write(data []byte) (int, error) {
	remaining := b.limit - b.buffer.Len()
	if len(data) > remaining {
		if remaining > 0 {
			_, _ = b.buffer.Write(data[:remaining])
		}
		b.overflow = true
		b.cancel()
		return len(data), nil
	}
	return b.buffer.Write(data)
}

func (b *limitedBuffer) Bytes() []byte { return b.buffer.Bytes() }
