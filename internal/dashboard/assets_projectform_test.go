package dashboard

import (
	"encoding/json"
	"strings"
	"testing"
)

// G4.1 y G4.3: la validación del formulario de alta vive en un módulo puro de
// js/core/ y se ejecuta con Node, igual que la lógica de worktrees.
func TestProjectFormValidationRuntime(t *testing.T) {
	out := runNodeModules(t, []string{"projectform"},
		"import { validateProjectForm, buildRoles, repoPathProblem, violationsFromError, splitList } from './projectform.mjs';\n"+
			"const card = (o) => ({ key: '', name: '', repos: '', tech: '', nonBlocking: false, ...o });\n"+
			"console.log(JSON.stringify({\n"+
			"  noPath: validateProjectForm({ path: '  ', cards: [] }).violations,\n"+
			"  ok: validateProjectForm({ path: '/p', cards: [card({ key: 'web', name: 'Web UI', repos: 'src/web, .', tech: 'go, react', nonBlocking: true }), card({ name: 'Base de datos' })] }),\n"+
			"  empty: buildRoles([card({}), card({ key: '  ' })]),\n"+
			"  orphan: buildRoles([card({ repos: 'src' })]).violations,\n"+
			"  dup: buildRoles([card({ key: 'web' }), card({ key: 'web' }), card({ key: 'Api' }), card({ key: 'api' }), card({ name: 'Web' })]).violations,\n"+
			"  paths: ['src/web', '.', './a/../b', '/abs', '\\\\\\\\share', 'C:\\\\\\\\x', 'D:rel', '../fuera', 'a/../../fuera', '..\\\\\\\\fuera', ''].map(repoPathProblem),\n"+
			"  badRepos: buildRoles([card({ key: 'core', repos: '/abs, ../x, ok' })]).violations,\n"+
			"  errList: violationsFromError({ message: 'x', data: { violations: ['a', 'b'] } }),\n"+
			"  errPlain: violationsFromError({ message: 'caído' }),\n"+
			"  list: splitList(' a, b ,, c ,'),\n"+
			"}));\n")
	var res struct {
		NoPath []string `json:"noPath"`
		Ok     struct {
			Roles []struct {
				Key          string   `json:"key"`
				Name         string   `json:"name"`
				Repositories []string `json:"repositories"`
				Tech         []string `json:"tech"`
				NonBlocking  bool     `json:"non_blocking"`
			} `json:"roles"`
			Violations []string `json:"violations"`
		} `json:"ok"`
		Empty struct {
			Roles      []any    `json:"roles"`
			Violations []string `json:"violations"`
		} `json:"empty"`
		Orphan    []string `json:"orphan"`
		Dup       []string `json:"dup"`
		Paths     []string `json:"paths"`
		BadRepos  []string `json:"badRepos"`
		ErrList   []string `json:"errList"`
		ErrPlain  []string `json:"errPlain"`
		SplitList []string `json:"list"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, out)
	}

	if len(res.NoPath) != 1 || !strings.Contains(res.NoPath[0], "ruta del proyecto") {
		t.Errorf("sin ruta debe haber un único error de ruta: %q", res.NoPath)
	}

	if len(res.Ok.Violations) != 0 || len(res.Ok.Roles) != 2 {
		t.Fatalf("un formulario válido no debe dar violaciones y sí 2 roles: %+v", res.Ok)
	}
	web, db := res.Ok.Roles[0], res.Ok.Roles[1]
	if web.Key != "web" || web.Name != "Web UI" || !web.NonBlocking || len(web.Repositories) != 2 || len(web.Tech) != 2 || web.Tech[1] != "react" {
		t.Errorf("rol web mal construido: %+v", web)
	}
	if db.Key != "base-de-datos" || db.Name != "Base de datos" || len(db.Repositories) != 1 || db.Repositories[0] != "." || len(db.Tech) != 0 {
		t.Errorf("rol sin clave debe derivarla del nombre y tomar «.» por defecto: %+v", db)
	}

	if len(res.Empty.Roles) != 0 || len(res.Empty.Violations) != 0 {
		t.Errorf("las tarjetas vacías se ignoran sin error: %+v", res.Empty)
	}
	if len(res.Orphan) != 1 || !strings.Contains(res.Orphan[0], "sin identificador ni nombre") {
		t.Errorf("repositorios sin identificador ni nombre son un error: %q", res.Orphan)
	}

	// web repetido, Api/api por mayúsculas y «Web» (derivada de nombre) contra web.
	if len(res.Dup) != 3 ||
		!strings.Contains(res.Dup[0], "«web» está repetido") ||
		!strings.Contains(res.Dup[1], "solo se distinguen por las mayúsculas") ||
		!strings.Contains(res.Dup[2], "«web» está repetido") {
		t.Errorf("claves duplicadas mal informadas: %q", res.Dup)
	}

	wantPaths := []bool{false, false, false, true, true, true, true, true, true, true, false}
	for i, want := range wantPaths {
		if got := res.Paths[i] != ""; got != want {
			t.Errorf("repoPathProblem caso %d: problema=%v, quiero %v (%q)", i, got, want, res.Paths[i])
		}
	}
	if len(res.BadRepos) != 2 {
		t.Errorf("deben informarse la ruta absoluta y la que sale del proyecto, y no «ok»: %q", res.BadRepos)
	}

	if len(res.ErrList) != 2 || res.ErrList[0] != "a" {
		t.Errorf("violationsFromError debe devolver la lista del servidor: %q", res.ErrList)
	}
	if len(res.ErrPlain) != 1 || !strings.Contains(res.ErrPlain[0], "caído") {
		t.Errorf("sin lista debe devolver el mensaje: %q", res.ErrPlain)
	}
	if len(res.SplitList) != 3 {
		t.Errorf("splitList = %q", res.SplitList)
	}
}

// El modal envía `tech` por rol y muestra la lista de violaciones del servidor.
func TestAddProjectModalWiresTechAndViolations(t *testing.T) {
	projects := readAsset(t, "js/projects.js")
	for _, marker := range []string{
		"role-tech-input", "validateProjectForm(", "violationsFromError(", "showViolations(",
	} {
		if !strings.Contains(projects, marker) {
			t.Errorf("projects.js debe contener %q", marker)
		}
	}
	if row := funcBody(t, projects, "addRoleRow"); !strings.Contains(row, "Tecnologías") {
		t.Errorf("cada tarjeta de rol debe tener el campo de tecnologías:\n%s", row)
	}
}

// G4.2: el repositorio de specs es obligatorio y admisible solo en multirepo.
func TestProjectFormMultirepoSpecsRuntime(t *testing.T) {
	out := runNodeModules(t, []string{"projectform"},
		"import { validateProjectForm, specsRepositoryProblem } from './projectform.mjs';\n"+
			"const v = (o) => validateProjectForm({ path: '/p', cards: [], ...o });\n"+
			"console.log(JSON.stringify({\n"+
			"  mono: v({ topology: 'monorepo-embedded', specsRepository: '' }),\n"+
			"  monoIgnoresSpecs: v({ topology: 'monorepo-embedded', specsRepository: '../x' }),\n"+
			"  multiOk: v({ topology: 'multirepo', specsRepository: ' repo-specs ' }),\n"+
			"  multiEmpty: v({ topology: 'multirepo', specsRepository: '  ' }).violations,\n"+
			"  problems: ['.', './', '/abs', 'C:\\\\x', '../fuera', 'a/../../b', 'repo-specs', 'repos/specs'].map(specsRepositoryProblem),\n"+
			"}));\n")
	var res struct {
		Mono struct {
			SpecsRepository string   `json:"specsRepository"`
			Violations      []string `json:"violations"`
		} `json:"mono"`
		MonoIgnoresSpecs struct {
			SpecsRepository string   `json:"specsRepository"`
			Violations      []string `json:"violations"`
		} `json:"monoIgnoresSpecs"`
		MultiOk struct {
			SpecsRepository string   `json:"specsRepository"`
			Violations      []string `json:"violations"`
		} `json:"multiOk"`
		MultiEmpty []string `json:"multiEmpty"`
		Problems   []string `json:"problems"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("salida de node ilegible: %v\n%s", err, out)
	}
	if len(res.Mono.Violations) != 0 || res.Mono.SpecsRepository != "" || len(res.MonoIgnoresSpecs.Violations) != 0 || res.MonoIgnoresSpecs.SpecsRepository != "" {
		t.Errorf("en monorepo-embedded no se pide ni se envía repositorio de specs: %+v %+v", res.Mono, res.MonoIgnoresSpecs)
	}
	if len(res.MultiOk.Violations) != 0 || res.MultiOk.SpecsRepository != "repo-specs" {
		t.Errorf("multirepo con repositorio válido: %+v", res.MultiOk)
	}
	if len(res.MultiEmpty) != 1 || !strings.Contains(res.MultiEmpty[0], "obligatorio") {
		t.Errorf("multirepo sin repositorio debe dar un error de obligatorio: %q", res.MultiEmpty)
	}
	wantBad := []bool{true, true, true, true, true, true, false, false}
	for i, want := range wantBad {
		if got := res.Problems[i] != ""; got != want {
			t.Errorf("specsRepositoryProblem caso %d: problema=%v, quiero %v (%q)", i, got, want, res.Problems[i])
		}
	}
}

// El campo del repositorio de specs está en el modal, oculto salvo en multirepo.
func TestAddProjectModalShowsSpecsRepositoryOnlyForMultirepo(t *testing.T) {
	html, projects := readAsset(t, "index.html"), readAsset(t, "js/projects.js")
	modal := html[strings.Index(html, `id="modal-add-project"`):]
	modal = modal[:strings.Index(modal, "</dialog>")]
	if !strings.Contains(modal, `id="field-project-specs-repository" hidden`) || !strings.Contains(modal, `id="input-project-specs-repository"`) {
		t.Error("el campo del repositorio de specs debe estar en el modal de alta y oculto por defecto")
	}
	for _, marker := range []string{"updateTopologyFields", "'multirepo'", "specs_repository"} {
		if !strings.Contains(projects, marker) {
			t.Errorf("projects.js debe contener %q", marker)
		}
	}
}
