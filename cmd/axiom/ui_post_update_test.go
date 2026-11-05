package main

import (
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/dashboard"
)

func TestParseUIFlags(t *testing.T) {
	got, err := parseUIFlags([]string{"--path", "/work", "--port", "8123", "--no-browser", "--post-update-sync"})
	if err != nil {
		t.Fatalf("parseUIFlags: %v", err)
	}
	want := uiOptions{port: 8123, noBrowser: true, path: "/work", postUpdateSync: true}
	if got != want {
		t.Fatalf("opciones = %+v, quiero %+v", got, want)
	}

	defaults, err := parseUIFlags(nil)
	if err != nil {
		t.Fatalf("parseUIFlags(nil): %v", err)
	}
	if defaults.port != 8080 || defaults.noBrowser || defaults.path != "." || defaults.postUpdateSync {
		t.Fatalf("valores por defecto = %+v", defaults)
	}

	if _, err := parseUIFlags([]string{"--no-existe"}); err == nil {
		t.Fatal("una bandera desconocida debe dar error")
	}
}

func busyPort(t *testing.T) (int, net.Listener) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l.Addr().(*net.TCPAddr).Port, l
}

// En el arranque normal, un puerto ocupado se sortea con el siguiente libre.
func TestStartDashboardServerProbesAnotherPortNormally(t *testing.T) {
	port, _ := busyPort(t)
	srv := dashboard.NewServer(dashboard.NewService(t.TempDir()))
	got, err := startDashboardServer(srv, uiOptions{port: port})
	if err != nil {
		t.Fatalf("startDashboardServer: %v", err)
	}
	t.Cleanup(func() { _ = srv.Shutdown(t.Context()) })
	if got == port {
		t.Fatalf("con el puerto %d ocupado debió elegir otro", port)
	}
}

// Con --post-update-sync el proceso espera a que el viejo libere el puerto y se
// queda con ese mismo puerto, sin sondear otro.
func TestStartDashboardServerPostUpdateKeepsTheSamePort(t *testing.T) {
	port, held := busyPort(t)
	go func() {
		time.Sleep(300 * time.Millisecond)
		_ = held.Close()
	}()

	srv := dashboard.NewServer(dashboard.NewService(t.TempDir()))
	got, err := startDashboardServer(srv, uiOptions{port: port, postUpdateSync: true})
	if err != nil {
		t.Fatalf("startDashboardServer tras actualizar (puerto %s): %v", strconv.Itoa(port), err)
	}
	t.Cleanup(func() { _ = srv.Shutdown(t.Context()) })
	if got != port {
		t.Fatalf("puerto = %d, quiero el mismo %d", got, port)
	}
}
