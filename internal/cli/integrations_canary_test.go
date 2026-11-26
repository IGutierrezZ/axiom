package cli

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

const canaryValue = "s3cr3t-CANARIO-9f8e7d"

// Canario (RF-INT-16): un valor distintivo en el almacén en memoria y en la variable de entorno no
// aparece, tras ningún flujo, ni en axiom.yaml, ni en ningún fichero del espacio (carpeta del cambio,
// ficheros con punto y cerrojo incluidos), ni en stdout, stderr o ninguna salida --json.
func TestIntegrationsCanaryNeverAppearsInFilesOrOutputs(t *testing.T) {
	root, changeRoot, fake := integrationsWorkspace(t, fakeBlock)
	sealForGateTest(t, root, "x")
	fake.LeakTokenInError() // el doble envuelve el valor del secreto en su error
	t.Setenv("AXIOM_INTEGRATION_FAKE_TOKEN", canaryValue)
	if err := secrets.Default().Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(canaryValue)); err != nil {
		t.Fatal(err)
	}
	warnings := captureIntegrationWarnings(t)
	if err := os.WriteFile(filepath.Join(changeRoot, "design.md"), []byte("diseño\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var outputs []string
	collect := func(name, stdout, stderr string) {
		outputs = append(outputs, name+" stdout: "+stdout, name+" stderr: "+stderr)
	}

	// gate record con la entrega fallida.
	var gateOut bytes.Buffer
	if err := RunSDDGate([]string{"record", "--cwd", root, "--change", "x", "--gate", "design", "--decision", "approved", "--reason", "ok", "--actor", "Ana", "--json"}, &gateOut); err != nil {
		t.Fatal(err)
	}
	collect("gate record", gateOut.String(), warnings.String())
	warnings.Reset()

	// sync (sigue fallando), con y sin --json.
	for _, args := range [][]string{{"sync", "--cwd", root}, {"sync", "--cwd", root, "--json"}} {
		_, out, errOut := runIntegrationsCLI(t, "", args...)
		collect(strings.Join(args, " "), out, errOut+warnings.String())
		warnings.Reset()
	}

	// emit fallido, con y sin --json.
	_, out, errOut := runIntegrationsCLI(t, "", "emit", "--cwd", root, "--change", "x", "--event", "gate.design.approved", "--json")
	collect("emit", out, errOut)

	// secret set por stdin (el valor viaja por stdin) y status, con y sin --json.
	_, out, errOut = runIntegrationsCLI(t, canaryValue+"\n", "secret", "set", "--plugin", "fake", "--name", "token")
	collect("secret set", out, errOut)
	for _, args := range [][]string{{"secret", "status", "--plugin", "fake"}, {"secret", "status", "--plugin", "fake", "--json"}} {
		_, out, errOut = runIntegrationsCLI(t, "", args...)
		collect(strings.Join(args, " "), out, errOut)
	}

	for _, output := range outputs {
		if strings.Contains(output, canaryValue) {
			t.Errorf("una salida contiene el valor del secreto: %q", output)
		}
	}
	if !strings.Contains(strings.Join(outputs, "\n"), "[REDACTED]") {
		t.Error("el error del plugin debía llegar saneado con el marcador de redacción")
	}

	// Ningún fichero del espacio (axiom.yaml, carpeta del cambio, ficheros con punto y cerrojo).
	scanned := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		scanned++
		if bytes.Contains(data, []byte(canaryValue)) {
			t.Errorf("el fichero %s contiene el valor del secreto", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !fileExistsCLI(filepath.Join(changeRoot, ".integrations-outbox.yaml")) || scanned < 4 {
		t.Fatalf("el recorrido debía incluir la cola y los demás ficheros (escaneados %d)", scanned)
	}
	if !fileExistsCLI(filepath.Join(changeRoot, ".integrations.lock")) {
		t.Error("el cerrojo (sin git, en la carpeta del cambio) forma parte del recorrido")
	}
}
