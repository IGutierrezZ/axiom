package opencode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// catalogOfSize genera un catálogo verbose válido de al menos minBytes bytes.
func catalogOfSize(minBytes int) string {
	var b strings.Builder
	for i := 0; b.Len() < minBytes; i++ {
		fmt.Fprintf(&b, "bulk/model-%d\n{\"id\":\"model-%d\",\"name\":\"Model %d\",\"capabilities\":{\"toolcall\":true},\"variants\":{\"low\":{},\"high\":{}}}\n", i, i, i)
	}
	return b.String()
}

func readerRunner(r io.Reader) CatalogRunner {
	return func(context.Context, Command) (io.Reader, error) { return r, nil }
}

func wantCatalogKind(t *testing.T, err error, kind CatalogErrorKind) {
	t.Helper()
	var catalogErr *CatalogError
	if !errors.As(err, &catalogErr) || catalogErr.Kind != kind {
		t.Fatalf("error = %v, want %v", err, kind)
	}
}

func TestDiscoverCatalogAcceptsOutputAboveOneMiBBelowCeiling(t *testing.T) {
	data := catalogOfSize(3 << 20)
	if len(data) <= 1<<20 || len(data) >= maxCatalogOutput {
		t.Fatalf("fixture size = %d, want between 1 MiB and %d", len(data), maxCatalogOutput)
	}
	providers, err := DiscoverCatalogWithRunner(context.Background(), "project", readerRunner(strings.NewReader(data)))
	if err != nil {
		t.Fatalf("DiscoverCatalogWithRunner() error = %v, want output between 1 MiB and 16 MiB accepted", err)
	}
	if len(providers["bulk"].Models) < 1000 {
		t.Fatalf("models = %d, want the whole streamed catalog", len(providers["bulk"].Models))
	}
}

func TestDiscoverCatalogRejectsOutputAboveCeilingWithTypedError(t *testing.T) {
	data := catalogOfSize(maxCatalogOutput + 1)
	_, err := DiscoverCatalogWithRunner(context.Background(), "project", readerRunner(strings.NewReader(data)))
	wantCatalogKind(t, err, CatalogErrorOutputTooLarge)
}

func TestDiscoverCatalogRejectsOverflowCarriedByNoiseAfterValidRecords(t *testing.T) {
	r := io.MultiReader(strings.NewReader(verboseCatalog), strings.NewReader(strings.Repeat("ruido de plugin\n", maxCatalogOutput/16+1)))
	_, err := DiscoverCatalogWithRunner(context.Background(), "project", readerRunner(r))
	wantCatalogKind(t, err, CatalogErrorOutputTooLarge)
}

func TestDiscoverCatalogInvalidJSONInTheMiddleOfAStream(t *testing.T) {
	data := "ok/one\n{\"id\":\"one\",\"capabilities\":{\"toolcall\":true}}\nbad/two\n{\"id\": nope}\n"
	_, err := DiscoverCatalogWithRunner(context.Background(), "project", readerRunner(strings.NewReader(data)))
	wantCatalogKind(t, err, CatalogErrorMalformed)
}

func TestDiscoverCatalogKeepsRecordsPackedInOneChunk(t *testing.T) {
	// Los registros pegados sin separación extra obligan al lector a devolver
	// al búfer lo que el decodificador leyó de más.
	data := "a/one\n{\"id\":\"one\",\"capabilities\":{\"toolcall\":true}}\nb/two\n{\"id\":\"two\",\"capabilities\":{\"toolcall\":false}}\n"
	providers, err := DiscoverCatalogWithRunner(context.Background(), "project", readerRunner(io.MultiReader(strings.NewReader(data))))
	if err != nil || len(providers) != 2 {
		t.Fatalf("providers = %v, err = %v; want both records", providers, err)
	}
}

func TestDiscoverCatalogHonorsChildExitStatusOverParseError(t *testing.T) {
	// Un hijo que muere a mitad de registro con código distinto de cero es un
	// comando fallido, no un JSON inválido.
	runner := func(ctx context.Context, command Command) (io.Reader, error) {
		return helperStream(ctx, "crash-mid-record")
	}
	_, err := DiscoverCatalogWithRunner(context.Background(), "project", runner)
	wantCatalogKind(t, err, CatalogErrorCommandFailed)
}

func TestDiscoverCatalogDoesNotBlockOnAbundantStderr(t *testing.T) {
	runner := func(ctx context.Context, command Command) (io.Reader, error) {
		return helperStream(ctx, "stderr-flood")
	}
	done := make(chan error, 1)
	var providers map[string]Provider
	go func() {
		var err error
		providers, err = DiscoverCatalogWithRunner(context.Background(), "project", runner)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil || len(providers["custom"].Models) != 1 {
			t.Fatalf("providers = %v, err = %v; want the catalog despite 8 MiB of stderr", providers, err)
		}
	case <-time.After(60 * time.Second):
		t.Fatal("discovery blocked while the child filled its stderr pipe")
	}
}

func TestDiscoverCatalogCancelsChildThatOverflowsStdout(t *testing.T) {
	runner := func(ctx context.Context, command Command) (io.Reader, error) {
		return helperStream(ctx, "stdout-forever")
	}
	started := time.Now()
	_, err := DiscoverCatalogWithRunner(context.Background(), "project", runner)
	wantCatalogKind(t, err, CatalogErrorOutputTooLarge)
	if elapsed := time.Since(started); elapsed > catalogTimeout {
		t.Fatalf("overflowing child ran for %v, want cancellation well before the discovery timeout", elapsed)
	}
}

// helperStream relanza el binario de test como hijo con el modo indicado.
func helperStream(ctx context.Context, mode string) (io.Reader, error) {
	return streamCatalogCommand(ctx, Command{
		Path: os.Args[0],
		Args: []string{"-test.run=^TestCatalogHelperProcess$"},
		Env:  []string{"AXIOM_CATALOG_HELPER=" + mode},
	})
}

// TestCatalogHelperProcess no es una prueba: es el cuerpo del hijo que lanzan
// las pruebas anteriores. Sin la variable de entorno, no hace nada.
func TestCatalogHelperProcess(t *testing.T) {
	mode := os.Getenv("AXIOM_CATALOG_HELPER")
	if mode == "" {
		return
	}
	switch mode {
	case "stderr-flood":
		chunk := []byte(strings.Repeat("e", 1024))
		for i := 0; i < 8*1024; i++ {
			_, _ = os.Stderr.Write(chunk)
		}
		_, _ = os.Stdout.WriteString(verboseCatalog[:strings.Index(verboseCatalog, "other/plain")])
	case "stdout-forever":
		chunk := []byte(strings.Repeat("x", 1024) + "\n")
		for {
			if _, err := os.Stdout.Write(chunk); err != nil {
				os.Exit(2)
			}
		}
	case "crash-mid-record":
		_, _ = os.Stdout.WriteString("custom/model\n{\"id\":")
		os.Exit(3)
	}
	os.Exit(0)
}
