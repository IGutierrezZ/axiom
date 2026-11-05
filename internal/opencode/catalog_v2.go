package opencode

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"os/exec"
	"sync"
	"time"
)

// catalogSizeCheckInterval acota cada cuánto se mide el fichero de captura
// mientras el hijo escribe en él.
const catalogSizeCheckInterval = 100 * time.Millisecond

// catalogCommand construye el comando de descubrimiento para la versión mayor.
// V2 usa la API interna de modelos acotada al directorio del proyecto; la
// forma exacta (`api get` con `location[directory]` codificado) sigue a
// gentle-ai. V1 y cualquier versión no reconocida usan `models --verbose`.
func catalogCommand(major RuntimeMajor, projectDir string) Command {
	args := []string{"models", "--verbose"}
	if major == RuntimeV2 {
		args = []string{"api", "get", "/api/model?location%5Bdirectory%5D=" + url.QueryEscape(projectDir)}
	}
	return Command{Path: "opencode", Args: args, Dir: projectDir}
}

// parseV2ModelAPI analiza la respuesta de la API de modelos de OpenCode V2. Se
// exige `capabilities.tools` y `enabled` explícitos: un identificador de modelo
// por sí solo no prueba que sirva para las fases con herramientas. Los modelos
// deshabilitados no se ofrecen; los que declaran tools=false se conservan con
// ToolCall falso y los descarta FilterModelsForSDD, igual que en V1.
func parseV2ModelAPI(r io.Reader) (map[string]Provider, error) {
	var response struct {
		Data []struct {
			ID           string `json:"id"`
			ProviderID   string `json:"providerID"`
			ModelID      string `json:"modelID"`
			Name         string `json:"name"`
			Family       string `json:"family"`
			Enabled      *bool  `json:"enabled"`
			Capabilities *struct {
				Tools     *bool `json:"tools"`
				Reasoning *bool `json:"reasoning"`
			} `json:"capabilities"`
			Variants []struct {
				ID string `json:"id"`
			} `json:"variants"`
			Limit ModelLimit  `json:"limit"`
			Cost  []ModelCost `json:"cost"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&response); err != nil {
		return nil, catalogJSONError(err)
	}
	// Tras el objeto no puede venir nada más: lo contrario indica que la salida
	// no es solo la respuesta de la API.
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, &CatalogError{Kind: CatalogErrorUnsupportedSchema}
		}
		return nil, catalogJSONError(err)
	}
	if response.Data == nil {
		return nil, &CatalogError{Kind: CatalogErrorUnsupportedSchema}
	}
	providers := map[string]Provider{}
	for _, raw := range response.Data {
		if raw.ProviderID == "" || raw.ModelID == "" || raw.ID == "" || raw.Capabilities == nil || raw.Capabilities.Tools == nil || raw.Enabled == nil {
			return nil, &CatalogError{Kind: CatalogErrorUnsupportedSchema}
		}
		if !*raw.Enabled {
			continue
		}
		provider := providers[raw.ProviderID]
		if provider.ID == "" {
			provider = Provider{ID: raw.ProviderID, Name: raw.ProviderID, Models: map[string]Model{}}
		}
		variants := make([]string, 0, len(raw.Variants))
		for _, variant := range raw.Variants {
			if variant.ID == "" {
				return nil, &CatalogError{Kind: CatalogErrorUnsupportedSchema}
			}
			variants = append(variants, variant.ID)
		}
		sortVariants(variants)
		model := Model{ID: raw.ID, Name: raw.Name, Family: raw.Family, ToolCall: *raw.Capabilities.Tools, Limit: raw.Limit, Variants: variants}
		if raw.Capabilities.Reasoning != nil {
			model.Reasoning = *raw.Capabilities.Reasoning
		}
		if len(raw.Cost) > 0 {
			model.Cost = raw.Cost[0]
		}
		provider.Models[raw.ID] = model
		providers[raw.ProviderID] = provider
	}
	return providers, nil
}

// catalogJSONError traduce un fallo del decodificador JSON: conserva el
// CatalogError del techo, un tipo inesperado es un esquema no soportado y el
// resto es salida mal formada.
func catalogJSONError(err error) error {
	var catalogErr *CatalogError
	if errors.As(err, &catalogErr) {
		return catalogErr
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return &CatalogError{Kind: CatalogErrorUnsupportedSchema}
	}
	return &CatalogError{Kind: CatalogErrorMalformed}
}

// fileCatalogCommand lanza el comando con stdout redirigido a un fichero
// temporal regular y devuelve un lector sobre ese fichero. OpenCode V2 trunca a
// 262144 bytes la salida de `api get` cuando stdout es una tubería, mientras que
// el mismo comando contra un fichero entrega la respuesta completa (gentle-ai
// #5833), de ahí que V2 no use la tubería de streamCatalogCommand. Stderr se
// descarta: CatalogError nunca expone la salida del comando. Un monitor cancela
// al hijo si el fichero supera el techo. El lector devuelto debe cerrarse: Close
// cancela el proceso, lo recoge y borra el fichero.
func fileCatalogCommand(ctx context.Context, command Command) (io.Reader, error) {
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
	capture, err := os.CreateTemp("", "axiom-opencode-catalog-")
	if err != nil {
		cancel()
		if release != nil {
			release()
		}
		return nil, err
	}
	cmd.Stdout = capture
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		_ = capture.Close()
		_ = os.Remove(capture.Name())
		cancel()
		if release != nil {
			release()
		}
		return nil, err
	}
	if afterStart != nil {
		afterStart()
	}
	// El hijo heredó su propio descriptor; el lector reabre la ruta para leer
	// desde el principio con un desplazamiento independiente.
	_ = capture.Close()
	stream := &fileStream{cmd: cmd, path: capture.Name(), cancel: cancel, release: release, done: make(chan struct{})}
	go stream.reap()
	go stream.monitorSize()
	return stream, nil
}

// fileStream sirve la salida de un hijo capturada en un fichero temporal. La
// lectura espera a que el hijo termine y solo entonces abre el fichero, de modo
// que se lee la respuesta completa y estable. Cumple catalogExit.
type fileStream struct {
	cmd     *exec.Cmd
	path    string
	cancel  context.CancelFunc
	release func()

	done    chan struct{} // se cierra al recoger al hijo
	waitErr error         // válido tras observar done

	file    *os.File
	drained bool

	closeOnce sync.Once
}

func (f *fileStream) reap() {
	f.waitErr = f.cmd.Wait()
	if f.release != nil {
		f.release()
	}
	close(f.done)
}

// monitorSize cancela al hijo si el fichero de captura rebasa el techo,
// conservando la cota de escritura que daba la tubería.
func (f *fileStream) monitorSize() {
	ticker := time.NewTicker(catalogSizeCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-f.done:
			return
		case <-ticker.C:
			if info, err := os.Stat(f.path); err == nil && info.Size() > maxCatalogOutput {
				f.cancel()
				return
			}
		}
	}
}

func (f *fileStream) Read(p []byte) (int, error) {
	<-f.done
	if f.file == nil {
		file, err := os.Open(f.path)
		if err != nil {
			return 0, err
		}
		f.file = file
	}
	n, err := f.file.Read(p)
	if errors.Is(err, io.EOF) {
		f.drained = true
	}
	return n, err
}

func (f *fileStream) Drained() bool { return f.drained }

// WaitError espera a que el hijo termine y devuelve su error de salida.
func (f *fileStream) WaitError() error {
	<-f.done
	return f.waitErr
}

// Close cancela el proceso si sigue vivo, lo recoge y borra siempre el fichero.
func (f *fileStream) Close() error {
	f.closeOnce.Do(func() {
		f.cancel()
		<-f.done
		if f.file != nil {
			_ = f.file.Close()
		}
		// El borrado puede fallar en Windows si un antivirus u otro proceso aún
		// tiene abierto el fichero. El paquete no tiene mecanismo de log ni de
		// aviso y un fallo aquí no altera el resultado del descubrimiento (el
		// fichero solo contiene la lista de modelos y vive en el directorio
		// temporal), así que se ignora a propósito.
		_ = os.Remove(f.path)
	})
	return nil
}
