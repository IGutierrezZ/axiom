package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// Pruebas de la guarda D con git real, entorno aislado y sin red (REQ-25.21, G-11, S-18,
// design D-18): bloquea las escrituras del agente en los documentos de Axiom cuando el veredicto
// es `wrong_branch` en los dos contextos y no hace nada en cualquier otro caso. Decide solo con
// lecturas de ficheros: ninguna de estas pruebas observa un proceso de git durante la guarda.

func guardOf(t *testing.T, payload GuardPayload) GuardDecision {
	t.Helper()
	return Default().Guard(context.Background(), payload)
}

func preWrite(paths ...string) GuardPayload {
	return GuardPayload{Event: GuardEventPre, Tool: "Write", Paths: paths}
}

func documentOf(repo string) string { return filepath.Join(changeRootOf(repo), "proposal.md") }

func wantBlocked(t *testing.T, decision GuardDecision) {
	t.Helper()
	if decision.Action != GuardActionBlock || decision.Message == "" {
		t.Fatalf("decisión %+v, se esperaba un bloqueo con mensaje", decision)
	}
}

func wantNothing(t *testing.T, decision GuardDecision) {
	t.Helper()
	if decision != guardNone {
		t.Fatalf("decisión %+v, se esperaba no hacer nada", decision)
	}
}

// REQ-25.21, escenario «Bloquea un documento de Axiom en rama equivocada» y supuesto S-18: el
// bloqueo rige aunque no haya un cambio registrado ni kickoff (los nombres salen de la política),
// con el motivo, la rama actual y la esperada, y la continuación exacta de `ensure`. No lanza un
// solo proceso.
func TestGuardBlocksAnAxiomDocumentOnTheWrongBranchEvenWithoutARegisteredChange(t *testing.T) {
	repo := embeddedCoreProject(t)
	seen := observe(t)

	decision := guardOf(t, preWrite(documentOf(repo)))

	wantBlocked(t, decision)
	for _, want := range []string{documentOf(repo), "«main»", "«feat/x»", "axiom branch ensure --cwd ", `" --change x`, "repite la escritura"} {
		if !strings.Contains(decision.Message, want) {
			t.Errorf("el mensaje %q debe contener %q", decision.Message, want)
		}
	}
	if cwd := continuationCWD(t, decision.Message); !pathidentity.SameDirectory(cwd, repo) {
		t.Errorf("--cwd = %q, se esperaba la carpeta maestra %q", cwd, repo)
	}
	if len(*seen) != 0 {
		t.Errorf("la guarda lanzó procesos de git: %v", *seen)
	}
}

// REQ-25.21 (escenario «Permite un documento de Axiom en la rama correcta») y G-11: la rama del
// cambio y la de archivado son correctas (unión de los dos contextos); cualquier otra, no.
func TestGuardAllowsTheChangeBranchAndTheArchiveBranchButNotAnyOther(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	document := documentOf(repo)

	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	wantNothing(t, guardOf(t, preWrite(document)))
	runGit(t, repo, "switch", "-q", "-c", "feat/x-archive")
	wantNothing(t, guardOf(t, preWrite(document)))
	runGit(t, repo, "switch", "-q", "-c", "otra")
	wantBlocked(t, guardOf(t, preWrite(document)))
}

// S-18 y G-11: el mensaje del bloqueo ofrece las dos continuaciones, la del cambio y la de
// archivado, porque la guarda no sabe en qué fase está el agente.
func TestGuardBlockMessageOffersTheArchiveContinuation(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)

	decision := guardOf(t, preWrite(documentOf(repo)))

	wantBlocked(t, decision)
	if !strings.Contains(decision.Message, "si estás archivando el cambio, con `axiom branch ensure --cwd ") ||
		!strings.Contains(decision.Message, "--change x --archive`") {
		t.Errorf("el mensaje %q debe ofrecer la continuación de archivado", decision.Message)
	}
}

// Un HEAD desacoplado no es la rama de ningún cambio: bloquea y lo dice.
func TestGuardBlocksADetachedHead(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	runGit(t, repo, "switch", "-q", "--detach", "HEAD")

	decision := guardOf(t, preWrite(documentOf(repo)))

	wantBlocked(t, decision)
	if !strings.Contains(decision.Message, "desacoplado") {
		t.Errorf("el mensaje %q debe decir que el HEAD está desacoplado", decision.Message)
	}
}

// REQ-25.21 escenario 1 (con instantánea) y REQ-25.36: los nombres de la instantánea mandan sobre
// la política, así que con el prefijo `inc` la rama `feat/x` es la equivocada y `inc/x` la correcta.
func TestGuardUsesTheSnapshotNamesOfTheKickoff(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, "git:\n  integration_branch: main\n  branch_prefix: inc\n"))
	sealWithSnapshot(t, repo, "x", snapshotOf(t, repo, "x", "core"), "core")
	commitAll(t, repo, "kickoff con instantánea")
	document := documentOf(repo)

	decision := guardOf(t, preWrite(document))
	wantBlocked(t, decision)
	if !strings.Contains(decision.Message, "«inc/x»") {
		t.Errorf("el mensaje %q debe nombrar la rama de la instantánea «inc/x»", decision.Message)
	}

	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	wantBlocked(t, guardOf(t, preWrite(document)))
	runGit(t, repo, "switch", "-q", "-c", "inc/x")
	wantNothing(t, guardOf(t, preWrite(document)))
}

// REQ-25.37 (R15): un kickoff sin instantánea deja el cambio exento; los sumideros acumulativos
// por nombre también.
func TestGuardExemptsAChangeWithoutSnapshotAndTheCumulativeSinks(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	sealChange(t, repo, "x", "core")

	wantNothing(t, guardOf(t, preWrite(documentOf(repo))))
	for _, change := range []string{"e2e-cumulative", "cumulative-core"} {
		wantNothing(t, guardOf(t, preWrite(filepath.Join(repo, "openspec", "changes", change, "spec.md"))))
	}
}

// REQ-25.21: el documento ODD `odd/tasks/<f>.md` está cubierto, con los nombres de la política, y
// su continuación repite el carril; no ofrece la de archivado, que el carril ODD no tiene.
func TestGuardCoversTheODDDocument(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	document := filepath.Join(repo, "odd", "tasks", "x.md")

	decision := guardOf(t, preWrite(document))
	wantBlocked(t, decision)
	if !strings.Contains(decision.Message, `" --change x --lane odd`) {
		t.Errorf("el mensaje %q debe incluir la continuación con --lane odd", decision.Message)
	}
	if strings.Contains(decision.Message, "archivando") {
		t.Errorf("el carril ODD no tiene archive: el mensaje %q no debe ofrecerlo", decision.Message)
	}

	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	wantNothing(t, guardOf(t, preWrite(document)))
	// `odd/tasks/README.sh` no es un documento de Axiom: el bloqueo no lo cubre.
	runGit(t, repo, "switch", "-q", "main")
	wantNothing(t, guardOf(t, preWrite(filepath.Join(repo, "odd", "tasks", "README.sh"))))
}

// REQ-25.21, escenario «Inerte fuera de un proyecto Axiom»: con `OriginStart` no hace nada y no
// escribe ningún fichero, ni siquiera dentro de un repositorio git.
func TestGuardIsInertOutsideAnAxiomProject(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	commit(t, dir, "inicial")
	before := checkoutState(t, dir)
	entries := listDirectory(t, dir)

	wantNothing(t, guardOf(t, preWrite(filepath.Join(dir, "openspec", "changes", "x", "proposal.md"))))
	wantNothing(t, guardOf(t, preWrite(filepath.Join(dir, "odd", "tasks", "x.md"))))

	if after := checkoutState(t, dir); after != before {
		t.Errorf("la guarda no debe tocar el repositorio:\nantes:\n%s\ndespués:\n%s", before, after)
	}
	if after := listDirectory(t, dir); !slices.Equal(after, entries) {
		t.Errorf("la guarda no debe escribir ficheros: antes %v, después %v", entries, after)
	}
}

func listDirectory(t *testing.T, dir string) []string {
	t.Helper()
	var names []string
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}
		names = append(names, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return names
}

// REQ-25.21, escenario «Fail-open ante un error interno»: configuración, kickoff o registro
// ilegibles, y un HEAD que no se puede leer, no bloquean.
func TestGuardFailsOpenOnInternalErrors(t *testing.T) {
	t.Parallel()
	cases := map[string]func(t *testing.T) (repo string){
		"configuración ilegible": func(t *testing.T) string { return newProject(t, "workspace: [sin cerrar\n") },
		"kickoff ilegible": func(t *testing.T) string {
			repo := embeddedCoreProject(t)
			writeFile(t, filepath.Join(changeRootOf(repo), "kickoff.yaml"), "schema: [sin cerrar\n")
			return repo
		},
		"registro ilegible": func(t *testing.T) string {
			repo := embeddedCoreProject(t)
			writeFile(t, recordFile(commonDir(repo), "sdd", "x"), "{ no es json\n")
			return repo
		},
		"HEAD ilegible": func(t *testing.T) string {
			repo := embeddedCoreProject(t)
			writeFile(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/.invalid\n")
			return repo
		},
		"HEAD ausente": func(t *testing.T) string {
			repo := embeddedCoreProject(t)
			if err := os.Remove(filepath.Join(repo, ".git", "HEAD")); err != nil {
				t.Fatal(err)
			}
			return repo
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			repo := setup(t)
			wantNothing(t, guardOf(t, preWrite(documentOf(repo))))
		})
	}
}

// Un contexto ya cancelado (el presupuesto agotado) y un pánico recuperado tampoco bloquean.
func TestGuardDoesNothingWithACancelledContextOrARecoveredPanic(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	payload := preWrite(documentOf(repo))
	wantBlocked(t, guardOf(t, payload))

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	wantNothing(t, Default().Guard(cancelled, payload))

	// Un motor sin montar entra en pánico al consultar el registro: la guarda lo recupera.
	var unmounted *Engine
	wantNothing(t, unmounted.Guard(context.Background(), payload))
}

// Cada evento tiene una única responsabilidad (design D-18, paso 5): `pre` solo bloquea y solo
// documentos de Axiom; `post` no bloquea nada (el aviso del resto de ficheros llega con S11-c2).
func TestGuardOnlyBlocksOnPreAndOnlyAxiomDocuments(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	document := documentOf(repo)

	post := preWrite(document)
	post.Event = GuardEventPost
	wantNothing(t, guardOf(t, post))
	unknown := preWrite(document)
	unknown.Event = ""
	wantNothing(t, guardOf(t, unknown))
	wantNothing(t, guardOf(t, preWrite(filepath.Join(repo, "internal", "foo", "foo.go"))))
	wantNothing(t, guardOf(t, preWrite()))
}

// Un parche de `apply_patch` puede traer varios ficheros: basta con que uno sea un documento de
// Axiom en la rama equivocada.
func TestGuardBlocksWhenAnyOfSeveralPathsIsAnAxiomDocument(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)

	wantBlocked(t, guardOf(t, preWrite(filepath.Join(repo, "internal", "a.go"), documentOf(repo))))

	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	wantNothing(t, guardOf(t, preWrite(filepath.Join(repo, "internal", "a.go"), documentOf(repo))))
}

// REQ-25.21 + Locate: la clasificación léxica solo reconoce los segmentos; que el documento sea del
// proyecto lo confirma `Locate`. Una carpeta `openspec/changes/x/` de un fixture dentro del
// proyecto no es un documento de Axiom de este proyecto.
func TestGuardConfirmsTheContainmentUnderThePlanningHome(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	fixture := filepath.Join(repo, "internal", "testdata", "openspec", "changes", "x", "proposal.md")

	wantNothing(t, guardOf(t, preWrite(fixture)))
	wantBlocked(t, guardOf(t, preWrite(documentOf(repo))))
}

// REQ-25.5 y G-11 en un multirepo: el repositorio de specs se trabaja in situ en su integración,
// de modo que su documento de Axiom bloquea en cualquier otra rama, con la continuación apuntando
// a la carpeta maestra; el repositorio de código no interviene.
func TestGuardBlocksTheSpecsRepositoryOfAMultirepoOutsideItsIntegrationBranch(t *testing.T) {
	t.Parallel()
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	document := filepath.Join(specs, "openspec", "changes", "x", "proposal.md")

	wantNothing(t, guardOf(t, preWrite(document)))

	runGit(t, specs, "switch", "-q", "-c", "feat/x")
	decision := guardOf(t, preWrite(document))
	wantBlocked(t, decision)
	if !strings.Contains(decision.Message, "«main»") || !pathidentity.SameDirectory(continuationCWD(t, decision.Message), master) {
		t.Errorf("el mensaje %q debe exigir la integración «main» y apuntar a la carpeta maestra %q", decision.Message, master)
	}
}

// Cargas de los tres runtimes (design D-18, paso 1): Claude Code, Codex y OpenCode.
func TestParseGuardPayloadOfTheThreeRuntimes(t *testing.T) {
	cwd := t.TempDir()
	at := func(parts ...string) string { return filepath.Join(append([]string{cwd}, parts...)...) }
	marshal := func(value any) string {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	cases := []struct {
		name  string
		agent string
		input string
		want  GuardPayload
	}{
		{
			name:  "Claude Code: file_path relativo, contra cwd",
			agent: GuardAgentClaudeCode,
			input: marshal(map[string]any{"cwd": cwd, "hook_event_name": "PreToolUse", "tool_name": "Write", "tool_input": map[string]any{"file_path": "openspec/changes/x/proposal.md"}}),
			want:  GuardPayload{CWD: cwd, Event: GuardEventPre, Tool: "Write", Paths: []string{at("openspec", "changes", "x", "proposal.md")}},
		},
		{
			name:  "Claude Code: file_path absoluto en PostToolUse",
			agent: GuardAgentClaudeCode,
			input: marshal(map[string]any{"cwd": cwd, "hook_event_name": "PostToolUse", "tool_name": "Edit", "tool_input": map[string]any{"file_path": at("internal", "a.go")}}),
			want:  GuardPayload{CWD: cwd, Event: GuardEventPost, Tool: "Edit", Paths: []string{at("internal", "a.go")}},
		},
		{
			name:  "Claude Code: notebook_path de NotebookEdit",
			agent: GuardAgentClaudeCode,
			input: marshal(map[string]any{"cwd": cwd, "hook_event_name": "PreToolUse", "tool_name": "NotebookEdit", "tool_input": map[string]any{"notebook_path": "n.ipynb"}}),
			want:  GuardPayload{CWD: cwd, Event: GuardEventPre, Tool: "NotebookEdit", Paths: []string{at("n.ipynb")}},
		},
		{
			name:  "Claude Code: otro evento no es de la guarda",
			agent: GuardAgentClaudeCode,
			input: marshal(map[string]any{"cwd": cwd, "hook_event_name": "SessionStart", "tool_name": "Write", "tool_input": map[string]any{"file_path": "a.go"}}),
			want:  GuardPayload{CWD: cwd, Tool: "Write", Paths: []string{at("a.go")}},
		},
		{
			name:  "Claude Code: las rutas se limpian",
			agent: GuardAgentClaudeCode,
			input: marshal(map[string]any{"cwd": cwd, "hook_event_name": "PreToolUse", "tool_name": "Write", "tool_input": map[string]any{"file_path": "openspec/changes/x/../../internal/a.go"}}),
			want:  GuardPayload{CWD: cwd, Event: GuardEventPre, Tool: "Write", Paths: []string{at("openspec", "internal", "a.go")}},
		},
		{
			name:  "Codex: cabeceras del parche de apply_patch, en orden y sin repetidos",
			agent: GuardAgentCodex,
			input: marshal(map[string]any{"cwd": cwd, "hook_event_name": "PreToolUse", "tool_name": "apply_patch", "tool_input": map[string]any{
				"command": "*** Begin Patch\n*** Add File: openspec/changes/x/a.md\n+hola\n*** Update File: internal/b.go\n@@\n-a\n+b\n*** Move to: internal/c.go\n*** Delete File: odd/tasks/y.md\n*** Update File: internal/b.go\n*** End Patch\n",
			}}),
			want: GuardPayload{CWD: cwd, Event: GuardEventPre, Tool: "apply_patch", Paths: []string{
				at("openspec", "changes", "x", "a.md"), at("internal", "b.go"), at("internal", "c.go"), at("odd", "tasks", "y.md"),
			}},
		},
		{
			name:  "Codex: el parche como lista de cadenas y con saltos CRLF",
			agent: GuardAgentCodex,
			input: marshal(map[string]any{"cwd": cwd, "hook_event_name": "PostToolUse", "tool_name": "apply_patch", "tool_input": map[string]any{
				"command": []string{"apply_patch", "*** Begin Patch\r\n*** Update File: internal/b.go\r\n*** End Patch\r\n"},
			}}),
			want: GuardPayload{CWD: cwd, Event: GuardEventPost, Tool: "apply_patch", Paths: []string{at("internal", "b.go")}},
		},
		{
			name:  "Codex: el parche como cadena directa de tool_input",
			agent: GuardAgentCodex,
			input: marshal(map[string]any{"cwd": cwd, "hook_event_name": "PreToolUse", "tool_name": "apply_patch", "tool_input": "*** Begin Patch\n*** Add File: a.md\n*** End Patch"}),
			want:  GuardPayload{CWD: cwd, Event: GuardEventPre, Tool: "apply_patch", Paths: []string{at("a.md")}},
		},
		{
			name:  "OpenCode: {event, tool, filePath, cwd}",
			agent: GuardAgentOpenCode,
			input: marshal(map[string]any{"event": "pre", "tool": "write", "filePath": "openspec/changes/x/proposal.md", "cwd": cwd}),
			want:  GuardPayload{CWD: cwd, Event: GuardEventPre, Tool: "write", Paths: []string{at("openspec", "changes", "x", "proposal.md")}},
		},
		{
			name:  "OpenCode: evento post y ruta absoluta",
			agent: GuardAgentOpenCode,
			input: marshal(map[string]any{"event": "post", "tool": "edit", "filePath": at("internal", "a.go"), "cwd": cwd}),
			want:  GuardPayload{CWD: cwd, Event: GuardEventPost, Tool: "edit", Paths: []string{at("internal", "a.go")}},
		},
		{
			name:  "OpenCode: un evento desconocido no es de la guarda",
			agent: GuardAgentOpenCode,
			input: marshal(map[string]any{"event": "durante", "tool": "edit", "filePath": "a.go", "cwd": cwd}),
			want:  GuardPayload{CWD: cwd, Tool: "edit", Paths: []string{at("a.go")}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseGuardPayload(tc.agent, strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("ParseGuardPayload: %v", err)
			}
			if got.Event != tc.want.Event || got.Tool != tc.want.Tool || got.CWD != tc.want.CWD || !slices.Equal(got.Paths, tc.want.Paths) {
				t.Errorf("carga = %+v\nse esperaba %+v", got, tc.want)
			}
		})
	}
}

// Una carga que no es del runtime que dice `--agent` no produce rutas: nada que bloquear.
func TestParseGuardPayloadWithoutPathsOrWithTheWrongShapeYieldsNoPaths(t *testing.T) {
	cwd := t.TempDir()
	for name, input := range map[string]struct{ agent, body string }{
		"Claude sin tool_input":               {GuardAgentClaudeCode, `{"cwd":"` + filepath.ToSlash(cwd) + `","hook_event_name":"PreToolUse"}`},
		"Claude con tool_input no objeto":     {GuardAgentClaudeCode, `{"hook_event_name":"PreToolUse","tool_input":"x"}`},
		"Codex sin cabeceras":                 {GuardAgentCodex, `{"hook_event_name":"PreToolUse","tool_input":{"command":"ls -la"}}`},
		"Codex con cabecera sin ruta":         {GuardAgentCodex, `{"hook_event_name":"PreToolUse","tool_input":{"command":"*** Update File:   \n"}}`},
		"OpenCode sin filePath":               {GuardAgentOpenCode, `{"event":"pre","tool":"write"}`},
		"OpenCode con filePath en blanco":     {GuardAgentOpenCode, `{"event":"pre","tool":"write","filePath":"  "}`},
		"Claude con una ruta con byte nulo":   {GuardAgentClaudeCode, `{"hook_event_name":"PreToolUse","tool_input":{"file_path":"a\u0000b"}}`},
		"Codex con cabeceras que no son ruta": {GuardAgentCodex, `{"hook_event_name":"PreToolUse","tool_input":{"command":"echo *** Update File: a.go"}}`},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := ParseGuardPayload(input.agent, strings.NewReader(input.body))
			if err != nil || len(got.Paths) != 0 {
				t.Errorf("carga = %+v (%v), se esperaba sin rutas y sin error", got, err)
			}
		})
	}
}

// Compruebo que el tipo de salida de un parche no depende del orden de las claves del objeto.
func TestParseGuardPayloadOfCodexIsDeterministic(t *testing.T) {
	body := `{"hook_event_name":"PreToolUse","cwd":"` + filepath.ToSlash(t.TempDir()) + `","tool_input":{"b":"*** Add File: b.md","a":"*** Add File: a.md","c":["*** Add File: c.md"]}}`

	first, err := ParseGuardPayload(GuardAgentCodex, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	for range 20 {
		again, err := ParseGuardPayload(GuardAgentCodex, strings.NewReader(body))
		if err != nil || !slices.Equal(again.Paths, first.Paths) {
			t.Fatalf("la carga debe dar siempre las mismas rutas: %v frente a %v (%v)", again.Paths, first.Paths, err)
		}
	}
	var names []string
	for _, path := range first.Paths {
		names = append(names, filepath.Base(path))
	}
	if !slices.Equal(names, []string{"a.md", "b.md", "c.md"}) {
		t.Errorf("rutas = %v, se esperaban las de las claves en orden alfabético", names)
	}
}
