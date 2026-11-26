package cli

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/identitystore"
)

// adoCanary es el PAT ficticio y reconocible de RF-ADO-15. Nunca es una credencial real.
const adoCanary = "ado-canario-PAT-9f8e7d6c5b4a"

// canaryForms son las formas en que el PAT puede delatarse: el valor, su base64 de ":"+PAT (la
// credencial de la cabecera Authorization: Basic) y la cabecera completa.
func canaryForms() []string {
	encoded := base64.StdEncoding.EncodeToString([]byte(":" + adoCanary))
	return []string{adoCanary, encoded, "Basic " + encoded}
}

// canaryServe monta las rutas del proceso con una guarda de credencial (solo acepta el canario, en la
// forma exacta de la cabecera) y, si fail no es nil, sustituye la respuesta de cada ruta por la que
// devuelve fail, que recibe la cabecera Authorization para poder hacerla eco.
func canaryServe(t *testing.T, e *adoEnv, fail func(authorization string) adotest.Response) {
	t.Helper()
	want := canaryForms()[2]
	for _, route := range adoProcessRoutes(adoConnectionOK()) {
		route := route
		e.srv.Handle("GET", route.path, func(r *adotest.Request) adotest.Response {
			got := r.Header.Get("Authorization")
			if got != want {
				return adotest.Response{Status: http.StatusUnauthorized, Body: "credencial no válida"}
			}
			if fail != nil {
				return fail(got)
			}
			return route.response
		})
	}
}

// canaryOutputs recoge todo lo que un flujo muestra: stdout, stderr y el texto de los errores que
// devuelven las funciones del verbo.
type canaryOutputs struct{ items []string }

func (c *canaryOutputs) add(label, text string) {
	c.items = append(c.items, label+": "+text)
}

// runFlows ejecuta connect (con y sin --user, legible y JSON) y discover (legible y JSON) y recoge sus
// salidas y los errores de las funciones de los verbos.
func (c *canaryOutputs) runFlows(t *testing.T, e *adoEnv) {
	t.Helper()
	flows := [][]string{
		{"connect", "--plugin", "azure-devops"},
		{"connect", "--plugin", "azure-devops", "--json"},
		{"connect", "--plugin", "azure-devops", "--user", "abc-123"},
		{"connect", "--plugin", "azure-devops", "--user", "abc-123", "--json"},
		{"discover", "--plugin", "azure-devops"},
		{"discover", "--plugin", "azure-devops", "--json"},
	}
	for _, args := range flows {
		_, out, errOut := e.run(t, args...)
		label := strings.Join(args, " ")
		c.add(label+" stdout", out)
		c.add(label+" stderr", errOut)
	}
	if _, err := connectIntegration(t.Context(), e.root, "azure-devops", ""); err != nil {
		c.add("connectIntegration error", err.Error())
	}
	if _, err := connectIntegration(t.Context(), e.root, "azure-devops", "abc-123"); err != nil {
		c.add("connectIntegration --user error", err.Error())
	}
	if _, err := discoverIntegration(t.Context(), e.root, "azure-devops"); err != nil {
		c.add("discoverIntegration error", err.Error())
	}
}

// findCanaryInFiles busca el canario en todo fichero bajo las raíces. Devuelve los ficheros que lo
// contienen, cuántos recorrió y sus nombres base.
func findCanaryInFiles(t *testing.T, roots ...string) (hits []string, scanned int, names map[string]bool) {
	t.Helper()
	names = map[string]bool{}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			scanned++
			names[filepath.Base(path)] = true
			for _, form := range canaryForms() {
				if strings.Contains(string(data), form) {
					hits = append(hits, path)
					break
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return hits, scanned, names
}

func assertNoCanary(t *testing.T, label string, items []string) {
	t.Helper()
	for _, item := range items {
		for _, form := range canaryForms() {
			if strings.Contains(item, form) {
				t.Errorf("%s: una salida contiene el PAT o su forma codificada (%q): %s", label, form[:min(len(form), 12)]+"…", item)
			}
		}
	}
}

// RF-ADO-15: el PAT canario y su forma base64 no aparecen en stdout, stderr, errores, fichero de
// identidad, axiom.yaml ni salida de discover, ni en ningún fichero bajo el workspace y el directorio
// de estado, en el camino feliz ni en los de error 401, 403, 429 (presupuesto agotado) y 500, ni
// cuando el servidor devuelve la cabecera Authorization en el cuerpo del error.
func TestADOCanaryNeverAppearsInOutputsOrFiles(t *testing.T) {
	echo := func(status int) func(string) adotest.Response {
		return func(authorization string) adotest.Response {
			return adotest.Response{Status: status, JSON: map[string]any{"message": fmt.Sprintf("rechazado; Authorization=%s; pat=%s", authorization, adoCanary)}}
		}
	}
	modes := []struct {
		name string
		fail func(string) adotest.Response
	}{
		{"camino feliz", nil},
		{"401", func(string) adotest.Response {
			return adotest.Response{Status: http.StatusUnauthorized, Body: "no autorizado " + adoCanary}
		}},
		{"403", func(string) adotest.Response {
			return adotest.Response{Status: http.StatusForbidden, Body: "prohibido " + adoCanary}
		}},
		{"429 con presupuesto agotado", func(string) adotest.Response {
			return adotest.Response{Status: http.StatusTooManyRequests, Header: http.Header{"Retry-After": {"2"}}, Body: "límite " + adoCanary}
		}},
		{"500", func(string) adotest.Response {
			return adotest.Response{Status: http.StatusInternalServerError, Body: "fallo " + adoCanary}
		}},
		{"500 con eco de la cabecera", echo(http.StatusInternalServerError)},
		{"400 con eco de la cabecera", echo(http.StatusBadRequest)},
		{"503 con presupuesto agotado y eco", func(a string) adotest.Response {
			r := echo(http.StatusServiceUnavailable)(a)
			r.Header = http.Header{"Retry-After": {"1"}}
			return r
		}},
	}
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			e := newADOEnv(t, adoBlock)
			t.Setenv(adoPATEnv, adoCanary)
			canaryServe(t, e, mode.fail)

			var outputs canaryOutputs
			outputs.runFlows(t, e)

			assertNoCanary(t, mode.name, outputs.items)
			hits, scanned, names := findCanaryInFiles(t, e.root, e.stateDir)
			for _, hit := range hits {
				t.Errorf("el fichero %s contiene el PAT o su forma codificada", hit)
			}
			if !names["axiom.yaml"] {
				t.Errorf("el recorrido debía incluir axiom.yaml (ficheros: %v)", names)
			}
			if mode.fail == nil {
				if !names[identitystore.FileName] {
					t.Error("el camino feliz debe guardar el fichero de identidad y el recorrido debe incluirlo")
				}
				if scanned < 2 {
					t.Errorf("ficheros recorridos: %d", scanned)
				}
			}

			// El canario sí viajó: el servidor recibió la credencial exacta en la cabecera.
			credentialSeen := false
			for _, r := range e.srv.Requests() {
				credentialSeen = credentialSeen || r.Header.Get("Authorization") == canaryForms()[2]
			}
			if !credentialSeen {
				t.Error("el servidor debía recibir la cabecera Authorization con el canario: la prueba sería vacua")
			}

			if mode.fail != nil {
				joined := strings.Join(outputs.items, "\n")
				if !strings.Contains(joined, "Error:") {
					t.Errorf("el modo %s debía producir errores:\n%s", mode.name, joined)
				}
			}
		})
	}
}

// El eco del servidor llega redactado y no vacío: el error cita lo que el servidor dijo sin el PAT.
func TestADOCanaryServerEchoIsRedactedNotDropped(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	t.Setenv(adoPATEnv, adoCanary)
	canaryServe(t, e, func(authorization string) adotest.Response {
		return adotest.Response{Status: http.StatusInternalServerError, JSON: map[string]any{"message": "eco: " + authorization + " " + adoCanary}}
	})

	code, _, errOut := e.run(t, "discover", "--plugin", "azure-devops")

	if code != 1 || !strings.Contains(errOut, "[oculto]") || !strings.Contains(errOut, "eco:") {
		t.Fatalf("el eco debe llegar redactado: %d %q", code, errOut)
	}
	assertNoCanary(t, "eco", []string{errOut})
}

// La prueba no es vacua: el recorrido detecta un canario sembrado en un fichero.
func TestADOCanaryScannerDetectsAPlantedValue(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.json"), []byte(`{"pat":"`+canaryForms()[1]+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "limpio.json"), []byte(`{"pat":"nada"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	hits, scanned, _ := findCanaryInFiles(t, dir)

	if scanned != 2 || len(hits) != 1 || filepath.Base(hits[0]) != "x.json" {
		t.Fatalf("el escaneo debía detectar solo el fichero sembrado: %v (recorridos %d)", hits, scanned)
	}
}
