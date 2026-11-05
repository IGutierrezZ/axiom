package dashboard

import (
	"errors"
	"net"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/app"
)

// relaunchSeams sustituye todas las costuras del relanzamiento: ninguna prueba
// lanza un proceso real, resuelve el ejecutable de `go test` ni llama a os.Exit.
type relaunchSeams struct {
	started []startedProcess
	exits   []int
	order   []string // «start» y «exit», en el orden en que ocurrieron
	exited  chan struct{}
	startFn func(exe string, args []string, dir string) error
	// childExit simula la muerte del proceso lanzado; sin escribirle, el hijo «vive».
	childExit chan error
}

type startedProcess struct {
	exe  string
	args []string
	dir  string
}

func stubRelaunch(t *testing.T) *relaunchSeams {
	t.Helper()
	seams := &relaunchSeams{exited: make(chan struct{}, 1)}

	origExe, origLook, origExists := executablePathFn, lookPathFn, fileExistsFn
	origStart, origExit := startProcessFn, exitProcessFn
	origDelay, origShutdown, origLive := relaunchDelay, shutdownTimeout, relaunchLiveCheck
	t.Cleanup(func() {
		relaunchLiveCheck = origLive
		executablePathFn, lookPathFn, fileExistsFn = origExe, origLook, origExists
		startProcessFn, exitProcessFn = origStart, origExit
		relaunchDelay, shutdownTimeout = origDelay, origShutdown
	})

	executablePathFn = func() (string, error) { return "/instalado/axiom", nil }
	lookPathFn = func(string) (string, error) { return "", errors.New("no está en el PATH") }
	fileExistsFn = func(string) bool { return true }
	relaunchDelay = 0
	relaunchLiveCheck = 50 * time.Millisecond
	seams.childExit = make(chan error, 1)
	startProcessFn = func(exe string, args []string, dir string) (<-chan error, error) {
		seams.order = append(seams.order, "start")
		seams.started = append(seams.started, startedProcess{exe, append([]string(nil), args...), dir})
		if seams.startFn != nil {
			if err := seams.startFn(exe, args, dir); err != nil {
				return nil, err
			}
		}
		return seams.childExit, nil
	}
	exitProcessFn = func(code int) {
		seams.order = append(seams.order, "exit")
		seams.exits = append(seams.exits, code)
		seams.exited <- struct{}{}
	}
	return seams
}

func selfUpgradeReport() app.UpgradeRunReport {
	return app.UpgradeRunReport{Status: app.UpgradeStatusSucceeded, SelfToolName: "axiom", RestartRequired: true}
}

func TestResolveRelaunchExecutable(t *testing.T) {
	cases := []struct {
		name      string
		exe       string
		exeErr    error
		exists    bool
		looked    string
		lookErr   error
		want      string
		wantError bool
	}{
		{"usa el ejecutable en curso", "/bin/axiom", nil, true, "/otro/axiom", nil, "/bin/axiom", false},
		{"ya no existe: usa el del PATH", "/bin/axiom", nil, false, "/path/axiom", nil, "/path/axiom", false},
		{"apartado por go install (~): usa el del PATH", "/bin/axiom.exe~", nil, true, "/path/axiom.exe", nil, "/path/axiom.exe", false},
		{"apartado como .old: usa el del PATH", "/bin/axiom.exe.OLD", nil, true, "/path/axiom.exe", nil, "/path/axiom.exe", false},
		{"os.Executable falla: usa el del PATH", "", errors.New("x"), false, "/path/axiom", nil, "/path/axiom", false},
		{"ninguno sirve", "/bin/axiom", nil, false, "", errors.New("no está"), "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stubRelaunch(t)
			executablePathFn = func() (string, error) { return tc.exe, tc.exeErr }
			fileExistsFn = func(string) bool { return tc.exists }
			lookPathFn = func(name string) (string, error) {
				if name != "axiom" {
					t.Errorf("se buscó %q en el PATH, quiero axiom", name)
				}
				return tc.looked, tc.lookErr
			}
			got, err := resolveRelaunchExecutable()
			if (err != nil) != tc.wantError || got != tc.want {
				t.Fatalf("resolveRelaunchExecutable() = %q, %v", got, err)
			}
		})
	}
}

func TestRelaunchArgs(t *testing.T) {
	got := relaunchArgs("/work/space", 8123)
	want := []string{"ui", "--path", "/work/space", "--port", "8123", "--no-browser", "--post-update-sync"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("relaunchArgs = %v, quiero %v", got, want)
	}
}

// Tras autoactualizarse con éxito: el proceso viejo no sincroniza, lanza el nuevo en
// el mismo workspace y puerto, libera el servidor y sale (en ese orden).
func TestUpdateJob_SelfUpgradeRelaunchesWithTheNewBinaryAndExits(t *testing.T) {
	seams := stubRelaunch(t)
	syncCalls := stubUpgradeSequence(t, selfUpgradeReport(), nil, nil, nil)
	root := t.TempDir()
	svc := NewService(root)
	svc.versions, _, _ = newTestVersionChecker("v3.6.0", []VersionInfo{availableVersion})
	srv := NewServer(svc)
	srv.port = 8123

	svc.StartUpdate()
	select {
	case <-seams.exited:
	case <-time.After(5 * time.Second):
		t.Fatalf("el proceso viejo no salió; estado: %+v", svc.UpdateStatus())
	}

	if *syncCalls != 0 {
		t.Fatalf("el proceso viejo no debe sincronizar: %d invocaciones", *syncCalls)
	}
	if len(seams.started) != 1 {
		t.Fatalf("procesos lanzados = %d, quiero 1", len(seams.started))
	}
	got := seams.started[0]
	wantArgs := []string{"ui", "--path", svc.getRootPath(), "--port", "8123", "--no-browser", "--post-update-sync"}
	if got.exe != "/instalado/axiom" || !reflect.DeepEqual(got.args, wantArgs) || got.dir != svc.getRootPath() {
		t.Fatalf("proceso lanzado = %+v, quiero %v", got, wantArgs)
	}
	if !reflect.DeepEqual(seams.order, []string{"start", "exit"}) || !reflect.DeepEqual(seams.exits, []int{0}) {
		t.Fatalf("orden = %v, salidas = %v; quiero lanzar y luego salir con 0", seams.order, seams.exits)
	}

	// Hasta que el proceso sale, la UI ve «restarting» con el comando manual a mano.
	st := svc.UpdateStatus()
	if st.Phase != "restarting" || !st.Running || !strings.HasPrefix(st.ManualHint, "axiom ui --path") || st.FromVersion != "v3.6.0" {
		t.Fatalf("estado = %+v", st)
	}
}

// Si el hijo muere nada más lanzarse (ejecutable dañado o borrado, banderas
// rechazadas), el viejo no cierra su servidor ni sale: sigue sirviendo.
func TestUpdateJob_ChildDyingRightAfterSpawnKeepsTheOldDashboardServing(t *testing.T) {
	seams := stubRelaunch(t)
	seams.childExit <- errors.New("exit status 1")
	stubUpgradeSequence(t, selfUpgradeReport(), nil, nil, nil)
	svc := NewService(t.TempDir())
	svc.versions, _, _ = newTestVersionChecker("v3.6.0", []VersionInfo{availableVersion})
	NewServer(svc)

	svc.StartUpdate()
	st := waitUpdateFinished(t, svc)
	if st.Phase != "failed" || !strings.HasPrefix(st.ManualHint, "axiom ui --path") || !strings.Contains(st.Message, "terminó nada más arrancar") {
		t.Fatalf("estado = %+v, quiero failed con el comando manual", st)
	}
	if len(seams.exits) != 0 {
		t.Fatal("si el hijo murió, el dashboard viejo no debe salir")
	}
}

// Con el hijo vivo, el viejo espera la ventana de comprobación antes de salir.
func TestUpdateJob_ExitsOnlyAfterTheLiveCheckWindow(t *testing.T) {
	seams := stubRelaunch(t)
	relaunchLiveCheck = 200 * time.Millisecond
	stubUpgradeSequence(t, selfUpgradeReport(), nil, nil, nil)
	svc := NewService(t.TempDir())
	svc.versions, _, _ = newTestVersionChecker("v3.6.0", []VersionInfo{availableVersion})
	NewServer(svc)

	start := time.Now()
	svc.StartUpdate()
	select {
	case <-seams.exited:
	case <-time.After(5 * time.Second):
		t.Fatal("el proceso viejo no salió")
	}
	if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
		t.Fatalf("salió a los %v, antes de la ventana de comprobación", elapsed)
	}
}

func TestManualRelaunchHintKeepsWindowsPathsCopyable(t *testing.T) {
	got := manualRelaunchHint(`C:\Users\yo\mi proyecto`)
	if want := `axiom ui --path "C:\Users\yo\mi proyecto"`; got != want {
		t.Fatalf("hint = %s, quiero %s", got, want)
	}
}

func TestUpdateJob_RelaunchFailureKeepsTheOldDashboardServing(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*relaunchSeams)
	}{
		{"no se puede lanzar el proceso", func(s *relaunchSeams) {
			s.startFn = func(string, []string, string) error { return errors.New("acceso denegado") }
		}},
		{"no se encuentra el ejecutable", func(*relaunchSeams) {
			executablePathFn = func() (string, error) { return "", errors.New("sin ruta") }
			fileExistsFn = func(string) bool { return false }
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seams := stubRelaunch(t)
			tc.setup(seams)
			stubUpgradeSequence(t, selfUpgradeReport(), nil, nil, nil)
			svc := NewService(t.TempDir())
			svc.versions, _, _ = newTestVersionChecker("v3.6.0", []VersionInfo{availableVersion})
			NewServer(svc)

			svc.StartUpdate()
			st := waitUpdateFinished(t, svc)
			if st.Phase != "failed" || !strings.HasPrefix(st.ManualHint, "axiom ui --path") || !strings.Contains(st.Message, "Axiom se ha actualizado") {
				t.Fatalf("estado = %+v, quiero failed con el comando manual", st)
			}
			if len(seams.exits) != 0 {
				t.Fatal("si no se pudo relanzar, el dashboard viejo no debe salir")
			}
		})
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// El proceso relanzado debe quedarse con el MISMO puerto aunque el viejo tarde unos
// instantes en liberarlo; no sondea otro.
func TestListenAndServeExact_WaitsForTheSamePort(t *testing.T) {
	port := freePort(t)
	held, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(300 * time.Millisecond)
		held.Close()
	}()

	srv := NewServer(NewService(t.TempDir()))
	got, err := srv.ListenAndServeExact(port, 5*time.Second)
	if err != nil {
		t.Fatalf("ListenAndServeExact: %v", err)
	}
	t.Cleanup(func() { _ = srv.Shutdown(t.Context()) })
	if got != port || srv.Port() != port {
		t.Fatalf("puerto = %d, quiero %d", got, port)
	}
}

func TestListenAndServeExact_FailsWhenThePortStaysBusy(t *testing.T) {
	port := freePort(t)
	held, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()

	srv := NewServer(NewService(t.TempDir()))
	start := time.Now()
	if _, err := srv.ListenAndServeExact(port, 300*time.Millisecond); err == nil {
		t.Fatal("con el puerto ocupado debe fallar en lugar de sondear otro")
	}
	if elapsed := time.Since(start); elapsed < 300*time.Millisecond {
		t.Fatalf("dejó de reintentar antes de tiempo: %v", elapsed)
	}
}
