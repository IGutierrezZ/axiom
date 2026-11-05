package dashboard

import (
	"os"
	"strings"
	"testing"
)

// El hijo relanzado escribe en ui.log en modo añadir: no pisa lo anterior.
func TestOpenUILogAppends(t *testing.T) {
	f, path, err := OpenUILog()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("primero\n")
	_ = f.Close()
	f, path2, err := OpenUILog()
	if err != nil || path2 != path {
		t.Fatalf("OpenUILog = %v, %q (antes %q)", err, path2, path)
	}
	_, _ = f.WriteString("segundo\n")
	_ = f.Close()
	if path == os.DevNull {
		t.Skip("sin directorio de caché de usuario")
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "primero\nsegundo\n") {
		t.Fatalf("ui.log = %q, quiero ambas líneas en orden", data)
	}
}
