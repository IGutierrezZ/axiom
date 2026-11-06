package dashboard

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// El Doctor pertenece al proyecto abierto: al cambiar de proyecto el informe
// anterior desaparece y una respuesta de otro proyecto no se publica nunca.
func TestDoctorIsScopedToTheOpenProjectRuntime(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node no está disponible: se omite la prueba de ejecución del Doctor por proyecto")
	}
	dir := t.TempDir()
	for _, name := range []string{"api", "icons", "state", "overview", "ecosystem"} {
		src := readAsset(t, "js/core/"+name+".js")
		src = regexp.MustCompile(`from '\./([a-z]+)\.js'`).ReplaceAllString(src, "from './$1.mjs'")
		if err := os.WriteFile(filepath.Join(dir, name+".mjs"), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	script := `import { state, set, on } from './state.mjs';
import { refreshOverview, resetProjectData, isReportForProject } from './overview.mjs';
import { doctorSections } from './ecosystem.mjs';
console.debug = () => {};

const winPath = name => ['C:', 'repos', name].join(String.fromCharCode(92));
const A = winPath('A');
const B = 'C:/repos/b/';
set('projects', { active_workspace: 'A', projects: [{ id: 'A', path: A }, { id: 'B', path: winPath('B') }] });

// El «servidor» examina el proyecto activo en el momento de recibir la petición y
// responde 40 ms después, como una petición lenta que sigue en vuelo.
let serverProject = A;
let doctorCalls = 0;
globalThis.fetch = async (path) => {
  doctorCalls++;
  const project = serverProject;
  await new Promise(r => setTimeout(r, 40));
  return { ok: true, status: 200, text: async () => JSON.stringify({ project, healthy: true, checks: [{ name: 'x', category: 'project', status: 'ok' }] }) };
};

const published = [];
on('overview', o => { const d = o.doctor; published.push(d.data ? d.data.project : d.status); });

const out = {};
const first = refreshOverview(['doctor']);          // proyecto A, en vuelo
await new Promise(r => setTimeout(r, 5));
// Cambio a B mientras la petición de A sigue en vuelo.
serverProject = winPath('B');
set('projects', { ...state.projects, active_workspace: 'B' });
resetProjectData();
out.afterReset = { status: state.overview.doctor.status, data: state.overview.doctor.data };
const second = refreshOverview(['doctor']);         // comparte la petición de A: debe repetirse
await Promise.all([first, second]);
out.final = { error: state.overview.doctor.error, status: state.overview.doctor.status, project: state.overview.doctor.data && state.overview.doctor.data.project };
out.publishedProjects = published;
out.calls = doctorCalls;

out.accept = [
  isReportForProject({ project: A }, 'c:/repos/a/'),
  isReportForProject({ project: A }, winPath('B')),
  isReportForProject({}, winPath('B')),
  isReportForProject({ project: A }, ''),
];
out.sections = doctorSections([
  { name: 'g', category: 'global' }, { name: 'p', category: 'project' }, { name: 'u' },
]).map(s => [s.key, s.checks.map(c => c.name)]);
console.log(JSON.stringify(out));
`
	if err := os.WriteFile(filepath.Join(dir, "run.mjs"), []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, "run.mjs")
	cmd.Dir = dir
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node falló: %v\n%s", err, raw)
	}
	var res struct {
		AfterReset struct {
			Status string `json:"status"`
			Data   any    `json:"data"`
		} `json:"afterReset"`
		Final struct {
			Status  string `json:"status"`
			Error   string `json:"error"`
			Project string `json:"project"`
		} `json:"final"`
		PublishedProjects []string `json:"publishedProjects"`
		Calls             int      `json:"calls"`
		Accept            []bool   `json:"accept"`
		Sections          [][]any  `json:"sections"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, raw)
	}
	if res.AfterReset.Status != "loading" || res.AfterReset.Data != nil {
		t.Errorf("tras cambiar de proyecto el informe anterior debe invalidarse: %+v", res.AfterReset)
	}
	if res.Final.Status != "ready" || !strings.HasSuffix(res.Final.Project, `B`) {
		t.Errorf("el informe final debe ser el del proyecto B: %+v", res.Final)
	}
	for _, p := range res.PublishedProjects {
		if strings.HasSuffix(p, `A`) {
			t.Errorf("se publicó el informe del proyecto anterior: %v", res.PublishedProjects)
		}
	}
	if res.Calls != 2 {
		t.Errorf("peticiones al Doctor = %d, se esperaban 2 (la compartida con A y la repetición)", res.Calls)
	}
	if want := []bool{true, false, true, true}; !equalBools(res.Accept, want) {
		t.Errorf("isReportForProject = %v, se esperaba %v", res.Accept, want)
	}
	if len(res.Sections) != 2 {
		t.Fatalf("secciones = %v", res.Sections)
	}
	if got, _ := json.Marshal(res.Sections); string(got) != `[["global",["g","u"]],["project",["p"]]]` {
		t.Errorf("secciones = %s", got)
	}
}

func equalBools(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
