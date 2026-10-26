package gitcli

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

// Pruebas de la capacidad de `git merge-tree --write-tree` y de la capa del árbol equivalente (Q1,
// design D-09, REQ-25.16): la versión se lee una vez por proceso, la capa se habilita desde la 2.38
// y, si la versión no se puede leer, se intenta la orden y la salida 129 es «no admitida».

func TestCapabilityParsesTheVersionOfGit(t *testing.T) {
	tests := []struct {
		output      string
		want        gitVersion
		wantAtLeast bool
	}{
		{"git version 2.41.0\n", gitVersion{major: 2, minor: 41, known: true}, true},
		{"git version 2.55.0.windows.5", gitVersion{major: 2, minor: 55, known: true}, true},
		{"git version 2.38.0", gitVersion{major: 2, minor: 38, known: true}, true},
		{"git version 2.37.9", gitVersion{major: 2, minor: 37, known: true}, false},
		{"git version 1.99.0", gitVersion{major: 1, minor: 99, known: true}, false},
		{"git version 3.0.1", gitVersion{major: 3, minor: 0, known: true}, true},
		{"git version Apple Git-xyz", gitVersion{}, false},
		{"", gitVersion{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.output, func(t *testing.T) {
			got := parseGitVersion(tc.output)

			if got != tc.want {
				t.Errorf("parseGitVersion(%q) = %+v, se esperaba %+v", tc.output, got, tc.want)
			}
			if got.atLeast(mergeTreeMinimumMajor, mergeTreeMinimumMinor) != tc.wantAtLeast {
				t.Errorf("atLeast(2.38) de %+v = %v, se esperaba %v: una versión desconocida nunca lo cumple", got, !tc.wantAtLeast, tc.wantAtLeast)
			}
		})
	}
}

// La versión se lee con `git version` una sola vez por proceso, aunque se pregunte muchas veces.
func TestCapabilityReadsTheVersionOnlyOncePerProcess(t *testing.T) {
	repo := newRepository(t)
	var cache processVersion
	seen := observe(t)

	first := cache.read(context.Background(), repo)
	second := cache.read(context.Background(), repo)

	if first != second || !first.known {
		t.Fatalf("versiones %+v y %+v: git está instalado y debía leerse", first, second)
	}
	var launched int
	for _, invocation := range *seen {
		if len(invocation) == 2 && invocation[1] == "version" {
			launched++
		}
	}
	if launched != 1 {
		t.Errorf("se lanzó `git version` %d veces, se esperaba una: %v", launched, *seen)
	}
}

// La variable de prueba manda sobre la versión instalada: falso omite la capa sin lanzar nada y
// cierto la intenta aunque la versión no se lea.
func TestCapabilityOverrideDecidesWithoutLaunchingGit(t *testing.T) {
	repo := newRepository(t)
	for _, supported := range []bool{false, true} {
		MergeTreeSupportedOverride = &supported
		t.Cleanup(func() { MergeTreeSupportedOverride = nil })

		if got := mergeTreeKnownUnsupported(context.Background(), repo); got == supported {
			t.Errorf("override %v: mergeTreeKnownUnsupported = %v", supported, got)
		}
	}
	unsupported := false
	MergeTreeSupportedOverride = &unsupported
	seen := observe(t)

	equivalent, supported, err := New().MergeTreeEquivalent(context.Background(), repo, "refs/heads/main", "refs/heads/main")

	if equivalent || supported || err != nil {
		t.Errorf("MergeTreeEquivalent con la capa no admitida = (%v, %v, %v), se esperaba (false, false, nil)", equivalent, supported, err)
	}
	if len(*seen) != 0 {
		t.Errorf("sin soporte no se lanza nada: %v", *seen)
	}
}

// requireGit238 omite la prueba con un git anterior a la 2.38.
func requireGit238(t *testing.T) {
	t.Helper()
	requireGit(t)
	output, err := exec.Command("git", "version").Output()
	if err != nil {
		t.Skipf("no se pudo leer la versión de git: %v", err)
	}
	version := parseGitVersion(string(output))
	if !version.atLeast(mergeTreeMinimumMajor, mergeTreeMinimumMinor) {
		t.Skipf("la prueba real de merge-tree exige git 2.38 o superior, hay %s", strings.TrimSpace(string(output)))
	}
}

// MergeTreeEquivalent confirma un squash aunque el destino haya avanzado, no confirma una rama con
// cambios propios ni una fusión con conflictos (salida 1: admitida y no equivalente) y solo lanza
// `merge-tree` y `rev-parse`, sin ninguna operación remota.
func TestMergeTreeEquivalentDetectsASquashButNotUnmergedOrConflictingWork(t *testing.T) {
	requireGit238(t)
	ctx := context.Background()
	repo := newRepository(t)
	writeFile(t, repo+"/base.txt", "base\n")
	runGit(t, repo, "add", "base.txt")
	commit(t, repo, "base")
	runGit(t, repo, "switch", "-q", "-c", "feat/squash")
	writeFile(t, repo+"/cambio.txt", "cambio\n")
	runGit(t, repo, "add", "cambio.txt")
	commit(t, repo, "cambio")
	runGit(t, repo, "switch", "-q", "-c", "feat/pendiente", "main")
	writeFile(t, repo+"/pendiente.txt", "pendiente\n")
	runGit(t, repo, "add", "pendiente.txt")
	commit(t, repo, "pendiente")
	runGit(t, repo, "switch", "-q", "-c", "feat/conflicto", "main")
	writeFile(t, repo+"/base.txt", "versión de la rama\n")
	runGit(t, repo, "add", "base.txt")
	commit(t, repo, "conflicto")
	runGit(t, repo, "switch", "-q", "main")
	runGit(t, repo, "merge", "--squash", "feat/squash")
	commit(t, repo, "squash")
	writeFile(t, repo+"/base.txt", "versión de main\n")
	runGit(t, repo, "add", "base.txt")
	commit(t, repo, "main avanza sobre el mismo fichero")
	seen := observe(t)
	git := New()

	for name, tc := range map[string]struct {
		branch         string
		wantEquivalent bool
	}{
		"un squash con main avanzado":  {"refs/heads/feat/squash", true},
		"una rama con cambios propios": {"refs/heads/feat/pendiente", false},
		"una fusión con conflictos":    {"refs/heads/feat/conflicto", false},
		"el propio destino":            {"refs/heads/main", true},
	} {
		t.Run(name, func(t *testing.T) {
			equivalent, supported, err := git.MergeTreeEquivalent(ctx, repo, "refs/heads/main", tc.branch)

			if err != nil || !supported || equivalent != tc.wantEquivalent {
				t.Errorf("MergeTreeEquivalent(main, %s) = (%v, %v, %v), se esperaba (%v, true, nil)", tc.branch, equivalent, supported, err, tc.wantEquivalent)
			}
		})
	}
	for _, invocation := range *seen {
		switch invocation[1] {
		case "merge-tree", "rev-parse", "version":
		default:
			t.Errorf("solo se lanzan merge-tree, rev-parse y version: %v", invocation)
		}
		if remoteOperationFor(invocation[1:]) {
			t.Errorf("ninguna operación remota: %v", invocation)
		}
	}
}

// Los argumentos con forma de opción se rechazan antes de lanzar nada.
func TestMergeTreeEquivalentRejectsArgumentsThatLookLikeOptions(t *testing.T) {
	repo := newRepository(t)
	seen := observe(t)

	for _, option := range []string{"", "--all", "-x"} {
		if _, _, err := New().MergeTreeEquivalent(context.Background(), repo, option, "refs/heads/main"); err == nil {
			t.Errorf("%q no es una referencia válida y debía rechazarse", option)
		}
		if _, _, err := New().MergeTreeEquivalent(context.Background(), repo, "refs/heads/main", option); err == nil {
			t.Errorf("%q no es una referencia válida y debía rechazarse", option)
		}
	}
	if len(*seen) != 0 {
		t.Errorf("las referencias inválidas no lanzan git: %v", *seen)
	}
}

// Q1: si la versión no se puede leer se intenta la orden y la salida 129 es «no admitida», no un
// fallo; la 1 son conflictos (admitida y no equivalente) y cualquier otra, un fallo de git.
func TestCapabilityTreatsExitCode129AsUnsupportedAndOtherCodesAsFailures(t *testing.T) {
	for code, want := range map[int]mergeTreeResult{0: mergeTreeClean, 1: mergeTreeConflicts, 129: mergeTreeUnsupported, 128: mergeTreeFailed, 2: mergeTreeFailed} {
		if got := classifyMergeTree(code); got != want {
			t.Errorf("classifyMergeTree(%d) = %v, se esperaba %v", code, got, want)
		}
	}
}
