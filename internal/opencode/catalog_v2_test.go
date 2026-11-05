package opencode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

const v2ModelAPISample = `{"data":[
  {"id":"gpt-5.5-fast","modelID":"gpt-5.5","providerID":"openai","name":"GPT 5.5 Fast","family":"gpt-5",
   "capabilities":{"tools":true,"reasoning":true},"enabled":true,
   "variants":[{"id":"high"},{"id":"low"},{"id":"medium"}],
   "limit":{"context":128000,"output":8192},"cost":[{"input":1.25,"output":10}]},
  {"id":"text-only","modelID":"text-only","providerID":"openai","name":"Text Only",
   "capabilities":{"tools":false},"enabled":true,"variants":[]},
  {"id":"disabled","modelID":"disabled","providerID":"openai","name":"Disabled",
   "capabilities":{"tools":true},"enabled":false,"variants":[]},
  {"id":"qwen/qwen3","modelID":"qwen/qwen3","providerID":"custom","name":"Qwen 3",
   "capabilities":{"tools":true},"enabled":true,"variants":[]}
]}`

func TestParseV2ModelAPIFiltersDisabledAndUntooledModels(t *testing.T) {
	providers, err := parseV2ModelAPI(strings.NewReader(v2ModelAPISample))
	if err != nil {
		t.Fatalf("parseV2ModelAPI() error = %v", err)
	}
	if _, ok := providers["openai"].Models["disabled"]; ok {
		t.Fatal("el modelo deshabilitado no debe ofrecerse")
	}
	model := providers["openai"].Models["gpt-5.5-fast"]
	if model.Name != "GPT 5.5 Fast" || model.Family != "gpt-5" || !model.ToolCall || !model.Reasoning {
		t.Fatalf("model = %+v", model)
	}
	if got := strings.Join(model.Variants, ","); got != "low,medium,high" {
		t.Fatalf("variants = %q, want orden semántico low,medium,high", got)
	}
	if model.Limit.Context != 128000 || model.Cost.Input != 1.25 {
		t.Fatalf("limit/cost = %+v / %+v", model.Limit, model.Cost)
	}
	if providers["openai"].Models["text-only"].ToolCall {
		t.Fatal("tools=false debe quedar con ToolCall falso")
	}
	// El identificador con barra es válido y conserva el proveedor de la API.
	if providers["custom"].Models["qwen/qwen3"].ID != "qwen/qwen3" {
		t.Fatalf("custom = %+v", providers["custom"])
	}
	// Mismo contrato que V1: solo los modelos con tools llegan al selector.
	if got := FilterModelsForSDD(providers["openai"]); len(got) != 1 || got[0].ID != "gpt-5.5-fast" {
		t.Fatalf("FilterModelsForSDD = %+v, want solo el modelo con tools", got)
	}
}

func TestParseV2ModelAPIReasoningIsExplicit(t *testing.T) {
	for name, capability := range map[string]string{"declarado": `,"reasoning":true`, "ausente": ``} {
		payload := `{"data":[{"id":"m","modelID":"m","providerID":"p","enabled":true,"capabilities":{"tools":true` + capability + `}}]}`
		providers, err := parseV2ModelAPI(strings.NewReader(payload))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := providers["p"].Models["m"].Reasoning; got != (capability != "") {
			t.Fatalf("%s: Reasoning = %t", name, got)
		}
	}
}

func TestParseV2ModelAPIRejectsUnsupportedOrMalformedOutput(t *testing.T) {
	for _, tt := range []struct {
		name    string
		payload string
		kind    CatalogErrorKind
	}{
		{"sin tools", `{"data":[{"id":"m","modelID":"m","providerID":"p","enabled":true}]}`, CatalogErrorUnsupportedSchema},
		{"capabilities vacío", `{"data":[{"id":"m","modelID":"m","providerID":"p","enabled":true,"capabilities":{}}]}`, CatalogErrorUnsupportedSchema},
		{"tools no booleano", `{"data":[{"id":"m","modelID":"m","providerID":"p","enabled":true,"capabilities":{"tools":"true"}}]}`, CatalogErrorUnsupportedSchema},
		{"sin enabled", `{"data":[{"id":"m","modelID":"m","providerID":"p","capabilities":{"tools":true}}]}`, CatalogErrorUnsupportedSchema},
		{"sin proveedor", `{"data":[{"id":"m","modelID":"m","enabled":true,"capabilities":{"tools":true}}]}`, CatalogErrorUnsupportedSchema},
		{"variante sin id", `{"data":[{"id":"m","modelID":"m","providerID":"p","enabled":true,"capabilities":{"tools":true},"variants":[{}]}]}`, CatalogErrorUnsupportedSchema},
		{"sin data", `{}`, CatalogErrorUnsupportedSchema},
		{"basura detrás", `{"data":[]} {"data":[]}`, CatalogErrorUnsupportedSchema},
		{"salida de V1", "openai/gpt-5.5\n{\"id\":\"gpt-5.5\"}\n", CatalogErrorMalformed},
		{"JSON truncado", `{"data":[{"id":`, CatalogErrorMalformed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseV2ModelAPI(strings.NewReader(tt.payload))
			wantCatalogKind(t, err, tt.kind)
		})
	}
}

func TestCatalogCommandForRuntimeMajor(t *testing.T) {
	v1 := catalogCommand(RuntimeV1, `C:\work\project`)
	if strings.Join(v1.Args, " ") != "models --verbose" || v1.Dir != `C:\work\project` || v1.Path != "opencode" {
		t.Fatalf("V1 command = %+v", v1)
	}
	v2 := catalogCommand(RuntimeV2, `C:\work\my project`)
	if got, want := strings.Join(v2.Args, " "), "api get /api/model?location%5Bdirectory%5D=C%3A%5Cwork%5Cmy+project"; got != want {
		t.Fatalf("V2 args = %q, want %q", got, want)
	}
	if v2.Dir != `C:\work\my project` {
		t.Fatalf("V2 Dir = %q", v2.Dir)
	}
}

func TestDiscoverCatalogSelectsTransportByDetectedRuntime(t *testing.T) {
	for _, tt := range []struct {
		name   string
		major  RuntimeMajor
		detErr error
		want   string // transporte invocado
		args   string
	}{
		{"V1", RuntimeV1, nil, "v1", "models --verbose"},
		{"V2", RuntimeV2, nil, "v2", "api get /api/model?location%5Bdirectory%5D=project"},
		{"detección fallida cae a V1", RuntimeUnknown, errors.New("no detectable"), "v1", "models --verbose"},
		{"versión desconocida cae a V1", RuntimeUnknown, nil, "v1", "models --verbose"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var called []string
			var gotArgs string
			v1 := func(_ context.Context, c Command) (io.Reader, error) {
				called, gotArgs = append(called, "v1"), strings.Join(c.Args, " ")
				return strings.NewReader(verboseCatalog), nil
			}
			v2 := func(_ context.Context, c Command) (io.Reader, error) {
				called, gotArgs = append(called, "v2"), strings.Join(c.Args, " ")
				return strings.NewReader(v2ModelAPISample), nil
			}
			detect := func(context.Context) (RuntimeMajor, error) { return tt.major, tt.detErr }
			providers, err := discoverCatalog(context.Background(), "project", detect, v1, v2)
			if err != nil {
				t.Fatalf("discoverCatalog() error = %v", err)
			}
			if strings.Join(called, ",") != tt.want || gotArgs != tt.args {
				t.Fatalf("transporte = %v args = %q, want %s %q", called, gotArgs, tt.want, tt.args)
			}
			if tt.want == "v1" && providers["custom"].Models["qwen/qwen3"].Name != "Qwen 3" {
				t.Fatalf("V1 catalog = %+v", providers)
			}
			if tt.want == "v2" && providers["openai"].Models["gpt-5.5-fast"].ID == "" {
				t.Fatalf("V2 catalog = %+v", providers)
			}
		})
	}
}

func TestDiscoverCatalogV2MapsFailuresToTypedErrors(t *testing.T) {
	run := func(r io.Reader, err error) error {
		_, got := discoverCatalogForMajor(context.Background(), "project", RuntimeV2, func(context.Context, Command) (io.Reader, error) { return r, err })
		return got
	}
	wantCatalogKind(t, run(nil, errors.New("fallo al lanzar")), CatalogErrorCommandFailed)
	wantCatalogKind(t, run(strings.NewReader("Available models:\n- openai/gpt-5.5\n"), nil), CatalogErrorMalformed)
	wantCatalogKind(t, run(strings.NewReader(`{"data":[{"id":"m"}]}`), nil), CatalogErrorUnsupportedSchema)
	wantCatalogKind(t, run(strings.NewReader(`{"data":[`+strings.Repeat(" ", maxCatalogOutput)+`]}`), nil), CatalogErrorOutputTooLarge)
}

// v2HelperStream relanza el binario de test como hijo, capturado en fichero.
func v2HelperStream(ctx context.Context, mode string) (io.Reader, error) {
	return fileCatalogCommand(ctx, Command{
		Path: os.Args[0],
		Args: []string{"-test.run=^TestCatalogV2HelperProcess$"},
		Env:  []string{"AXIOM_CATALOG_V2_HELPER=" + mode},
	})
}

// TestCatalogV2HelperProcess no es una prueba: es el cuerpo del hijo que lanzan
// las pruebas de fichero. Sin la variable de entorno, no hace nada.
func TestCatalogV2HelperProcess(t *testing.T) {
	mode := os.Getenv("AXIOM_CATALOG_V2_HELPER")
	if mode == "" {
		return
	}
	// Prueba de que el transporte es un fichero regular y no una tubería.
	if info, err := os.Stdout.Stat(); err != nil || !info.Mode().IsRegular() {
		os.Exit(4)
	}
	switch mode {
	case "large":
		fmt.Fprint(os.Stdout, v2ModelAPIOfSize(300<<10))
	case "overflow":
		chunk := []byte(strings.Repeat(" ", 1<<20)) // blancos JSON: el análisis llega al techo
		for {
			if _, err := os.Stdout.Write(chunk); err != nil {
				os.Exit(2)
			}
			time.Sleep(5 * time.Millisecond)
		}
	case "fail":
		_, _ = os.Stdout.WriteString(`{"data":[`)
		os.Exit(3)
	case "sleep":
		time.Sleep(time.Minute)
	case "valid-exit-3":
		_, _ = os.Stdout.WriteString(v2ModelAPISample)
		os.Exit(3)
	}
	os.Exit(0)
}

// v2ModelAPIOfSize genera una respuesta V2 válida de al menos minBytes bytes.
func v2ModelAPIOfSize(minBytes int) string {
	var b strings.Builder
	b.WriteString(`{"data":[`)
	for i := 0; b.Len() < minBytes; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"id":"model-%d","modelID":"model-%d","providerID":"bulk","name":"Model %d","capabilities":{"tools":true},"enabled":true,"variants":[{"id":"low"},{"id":"high"}]}`, i, i, i)
	}
	b.WriteString(`]}`)
	return b.String()
}

// captureStream deja ver a la prueba el fichero temporal de un fileStream.
func captureStream(dst **fileStream, mode string) CatalogRunner {
	return func(ctx context.Context, _ Command) (io.Reader, error) {
		r, err := v2HelperStream(ctx, mode)
		if err == nil {
			*dst = r.(*fileStream)
		}
		return r, err
	}
}

func wantRemoved(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("el fichero temporal %s debía borrarse, Stat error = %v", path, err)
	}
}

func TestFileCatalogCommandDeliversOutputAbove256KiBInFull(t *testing.T) {
	var stream *fileStream
	providers, err := discoverCatalogForMajor(context.Background(), "project", RuntimeV2, captureStream(&stream, "large"))
	if err != nil {
		t.Fatalf("discoverCatalogForMajor() error = %v", err)
	}
	info := len(v2ModelAPIOfSize(300 << 10))
	if info <= 262144 {
		t.Fatalf("la muestra (%d bytes) debe superar el truncado de 262144 bytes", info)
	}
	want := strings.Count(v2ModelAPIOfSize(300<<10), `"modelID"`)
	if got := len(providers["bulk"].Models); got != want || want < 1000 {
		t.Fatalf("modelos leídos = %d, want %d (respuesta completa)", got, want)
	}
	wantRemoved(t, stream.path)
}

func TestFileCatalogCommandRemovesCaptureOnChildFailure(t *testing.T) {
	var stream *fileStream
	_, err := discoverCatalogForMajor(context.Background(), "project", RuntimeV2, captureStream(&stream, "fail"))
	wantCatalogKind(t, err, CatalogErrorCommandFailed)
	wantRemoved(t, stream.path)
}

func TestFileCatalogCommandRemovesCaptureOnOverflow(t *testing.T) {
	var stream *fileStream
	started := time.Now()
	_, err := discoverCatalogForMajor(context.Background(), "project", RuntimeV2, captureStream(&stream, "overflow"))
	wantCatalogKind(t, err, CatalogErrorOutputTooLarge)
	if elapsed := time.Since(started); elapsed > catalogTimeout {
		t.Fatalf("el hijo desbocado corrió %v, debía cancelarse antes del plazo", elapsed)
	}
	wantRemoved(t, stream.path)
}

func TestFileStreamCloseCancelsRunningChildAndRemovesCapture(t *testing.T) {
	r, err := v2HelperStream(context.Background(), "sleep")
	if err != nil {
		t.Fatalf("fileCatalogCommand() error = %v", err)
	}
	stream := r.(*fileStream)
	if _, err := os.Stat(stream.path); err != nil {
		t.Fatalf("el fichero de captura debe existir mientras corre el hijo: %v", err)
	}
	started := time.Now()
	_ = stream.Close()
	if elapsed := time.Since(started); elapsed > catalogTimeout {
		t.Fatalf("Close tardó %v en recoger al hijo", elapsed)
	}
	wantRemoved(t, stream.path)
	_ = stream.Close() // idempotente
}

func TestFileCatalogCommandReportsMissingBinary(t *testing.T) {
	_, err := discoverCatalogForMajor(context.Background(), "project", RuntimeV2, func(ctx context.Context, _ Command) (io.Reader, error) {
		return fileCatalogCommand(ctx, Command{Path: "axiom-no-such-opencode-binary"})
	})
	wantCatalogKind(t, err, CatalogErrorMissingBinary)
}

func TestCatalogCommandV2EncodesNonASCIIDirectoryLikeUpstream(t *testing.T) {
	dir := `C:\Users\IñigoGutiérrez Zufia\proyecto`
	got := catalogCommand(RuntimeV2, dir)
	want := "/api/model?location%5Bdirectory%5D=" + url.QueryEscape(dir)
	if len(got.Args) != 3 || got.Args[2] != want {
		t.Fatalf("args = %q, want the url.QueryEscape encoding %q", got.Args, want)
	}
	// Codificación literal: ñ y é en UTF-8 porcentual, espacio como + y \ como %5C.
	if literal := "/api/model?location%5Bdirectory%5D=C%3A%5CUsers%5CI%C3%B1igoGuti%C3%A9rrez+Zufia%5Cproyecto"; got.Args[2] != literal {
		t.Fatalf("arg = %q, want %q", got.Args[2], literal)
	}
	if got.Dir != dir {
		t.Fatalf("Dir = %q, want the raw directory", got.Dir)
	}
}

func TestDiscoverCatalogSharesOneDeadlineBetweenDetectionAndCatalog(t *testing.T) {
	var detectDeadline, runnerDeadline time.Time
	detect := func(ctx context.Context) (RuntimeMajor, error) {
		detectDeadline, _ = ctx.Deadline()
		time.Sleep(300 * time.Millisecond) // la detección consume parte del plazo
		return RuntimeV1, nil
	}
	v1 := func(ctx context.Context, _ Command) (io.Reader, error) {
		runnerDeadline, _ = ctx.Deadline()
		return strings.NewReader(verboseCatalog), nil
	}
	started := time.Now()
	if _, err := discoverCatalog(context.Background(), "project", detect, v1, nil); err != nil {
		t.Fatalf("discoverCatalog() error = %v", err)
	}
	if detectDeadline.IsZero() || runnerDeadline.IsZero() {
		t.Fatal("detección y catálogo deben tener plazo")
	}
	if !runnerDeadline.Equal(detectDeadline) && runnerDeadline.After(detectDeadline) {
		t.Fatalf("el catálogo tiene plazo propio (%v) posterior al de la detección (%v)", runnerDeadline, detectDeadline)
	}
	if limit := started.Add(catalogTimeout + time.Second); detectDeadline.After(limit) {
		t.Fatalf("plazo total %v excede catalogTimeout", detectDeadline.Sub(started))
	}
}
