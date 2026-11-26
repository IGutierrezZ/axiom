package sddstatus

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// La verificación conjunta nunca bloquea (C-02-b, REQ-28.1, REQ-28.42, REQ-43.32): una ronda
// «Verificación conjunta» en `verify-report.md`, limpia o no, no cambia `nextRecommended`, las
// dependencias ni las razones de bloqueo de `axiom sdd status`; `status` no la lee ni la digiere.
func TestJointVerifyNonblocking(t *testing.T) {
	t.Parallel()
	const spec = "### Requirement: Auth\n#### Scenario: Expected behavior\n"
	base := testVerifyEnvelope("pass", 0, 0, "13/13", "46/46", 0, 0)
	rounds := map[string]string{
		"sin ronda":                 "",
		"ronda limpia":              "\n## Verificación conjunta (2026-11-24)\n\nAlcance: axiom.branch-scope/v1 · solo commits\n\n### api\n- resultado: ok\n- tip: 0123456\n\n**Resultado conjunto:** limpio\n",
		"ronda no limpia con hueco": "\n## Verificación conjunta (2026-11-24)\n\nAlcance: axiom.branch-scope/v1 · solo commits\n\n### web\n- resultado: unverifiable (branch_missing)\n\n**Resultado conjunto:** no limpio — web: branch_missing\n",
		"dos rondas":                "\n## Verificación conjunta (2026-11-23)\n\n**Resultado conjunto:** no limpio — web: branch_missing\n\n## Verificación conjunta (2026-11-24)\n\n**Resultado conjunto:** limpio\n",
	}

	resolve := func(t *testing.T, round string) Status {
		t.Helper()
		root := t.TempDir()
		changeRoot := seedReadyChange(t, root, "thin", "- [x] 1.1 Done\n")
		write(t, filepath.Join(changeRoot, "specs", "auth", "spec.md"), spec)
		write(t, filepath.Join(changeRoot, "verify-report.md"), base+round)
		status, err := Resolve(ResolveOptions{CWD: root, ChangeName: "thin"})
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		return status
	}

	reference := resolve(t, rounds["sin ronda"])
	if reference.NextRecommended != "archive" || reference.Dependencies.Archive != DependencyReady {
		t.Fatalf("la referencia debía poder archivarse: next %q, archive %q", reference.NextRecommended, reference.Dependencies.Archive)
	}
	for name, round := range rounds {
		t.Run(name, func(t *testing.T) {
			status := resolve(t, round)
			if status.NextRecommended != reference.NextRecommended {
				t.Errorf("nextRecommended = %q, se esperaba %q", status.NextRecommended, reference.NextRecommended)
			}
			if !reflect.DeepEqual(status.Dependencies, reference.Dependencies) {
				t.Errorf("dependencias = %+v, se esperaban %+v", status.Dependencies, reference.Dependencies)
			}
			if !reflect.DeepEqual(status.BlockedReasons, reference.BlockedReasons) {
				t.Errorf("razones de bloqueo = %v, se esperaban %v", status.BlockedReasons, reference.BlockedReasons)
			}
			raw, err := json.Marshal(status)
			if err != nil {
				t.Fatal(err)
			}
			if encoded := strings.ToLower(string(raw)); strings.Contains(encoded, "verificación conjunta") || strings.Contains(encoded, "branch-scope") {
				t.Errorf("status no proyecta la verificación conjunta: %s", encoded)
			}
		})
	}
}
