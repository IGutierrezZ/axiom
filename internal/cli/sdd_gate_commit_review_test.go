package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// Pruebas de `--commit-review` en `axiom sdd gate record` (RF-GAT-19, RF-GAT-20, RF-GAT-24,
// RF-GAT-25). La separación de la bandera es una prueba de tabla pura; el resto va sobre el punto de
// entrada en proceso `RunSDDGate`, con git real y sin compilar ningún binario.

func TestSplitCommitReviewFlag(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantMode kickoff.CommitReviewMode
		wantRest []string
		wantErr  string
	}{
		{name: "bandera ausente", args: []string{"--gate", "tasks", "--decision", "approved"}, wantRest: []string{"--gate", "tasks", "--decision", "approved"}},
		{name: "sin argumentos", args: nil, wantRest: []string{}},
		{name: "al principio", args: []string{"--commit-review", "stop", "--gate", "tasks"}, wantMode: "stop", wantRest: []string{"--gate", "tasks"}},
		{name: "en medio", args: []string{"--gate", "tasks", "--commit-review", "notify", "--decision", "approved"}, wantMode: "notify", wantRest: []string{"--gate", "tasks", "--decision", "approved"}},
		{name: "al final", args: []string{"--gate", "tasks", "--commit-review", "stop-before-commit"}, wantMode: "stop-before-commit", wantRest: []string{"--gate", "tasks"}},
		{name: "con --hand-off y --json", args: []string{"--hand-off", "--commit-review", "stop", "--json"}, wantMode: "stop", wantRest: []string{"--hand-off", "--json"}},
		{name: "repetida", args: []string{"--commit-review", "stop", "--commit-review", "notify"}, wantErr: "más de una vez"},
		{name: "sin valor al final", args: []string{"--gate", "tasks", "--commit-review"}, wantErr: "requiere un valor"},
		{name: "sin valor antes de otra bandera", args: []string{"--commit-review", "--gate", "tasks"}, wantErr: "requiere un valor"},
		{name: "valor inválido", args: []string{"--commit-review", "always"}, wantErr: "notify, stop, stop-before-commit"},
		{name: "pause", args: []string{"--commit-review", "pause"}, wantErr: "desconocido"},
		{name: "mayúscula", args: []string{"--commit-review", "Stop"}, wantErr: "desconocido"},
		{name: "mayúsculas", args: []string{"--commit-review", "NOTIFY"}, wantErr: "desconocido"},
		{name: "cadena vacía", args: []string{"--commit-review", ""}, wantErr: "desconocido"},
		{name: "con espacios", args: []string{"--commit-review", " stop"}, wantErr: "desconocido"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := append([]string(nil), tc.args...)
			mode, rest, err := splitCommitReviewFlag(tc.args)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("splitCommitReviewFlag() error = %v, se esperaba que contuviera %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("splitCommitReviewFlag() error = %v", err)
			}
			if mode != tc.wantMode {
				t.Errorf("modo = %q, se esperaba %q", mode, tc.wantMode)
			}
			if !reflect.DeepEqual(rest, tc.wantRest) {
				t.Errorf("resto = %v, se esperaba %v", rest, tc.wantRest)
			}
			if !reflect.DeepEqual(tc.args, original) {
				t.Errorf("la separación no debe modificar la entrada: %v -> %v", original, tc.args)
			}
		})
	}
}

// RF-GAT-19 y RF-GAT-20: cada token se acepta y se conserva tal cual en gates.yaml.
func TestGateRecordCommitReviewWritesEveryToken(t *testing.T) {
	for _, mode := range kickoff.CommitReviewModes() {
		t.Run(string(mode), func(t *testing.T) {
			ws := handOffProject(t, guardSealArgs)

			out, err := gateRecord(t, ws, "--gate", "tasks", "--decision", "approved", "--commit-review", string(mode))
			if err != nil {
				t.Fatalf("gate record --commit-review %s: %v", mode, err)
			}

			ledger := handOffLedger(t, ws)
			if len(ledger.Records) != 1 {
				t.Fatalf("registros = %+v, se esperaba uno", ledger.Records)
			}
			record := ledger.Records[0]
			if record.Gate != kickoff.GateTasks || record.Decision != kickoff.DecisionApproved || record.CommitReview != mode || record.HandedOff {
				t.Errorf("registro = %+v", record)
			}
			data, readErr := os.ReadFile(filepath.Join(guardChangeRoot(ws), "gates.yaml"))
			if readErr != nil || !strings.Contains(string(data), "commit_review: "+string(mode)) {
				t.Errorf("gates.yaml debe llevar commit_review: %s (%v):\n%s", mode, readErr, data)
			}
			if want := "revisión por commit: " + string(mode); !strings.Contains(out, want) {
				t.Errorf("la cabecera debe mencionar el modo (%q):\n%s", want, out)
			}
		})
	}
}

// RF-GAT-20: el modo viaja con la cesión en un único registro.
func TestGateRecordCommitReviewTravelsWithTheHandOffInOneRecord(t *testing.T) {
	ws := handOffProject(t, guardSealArgs)

	out, err := gateRecord(t, ws, "--gate", "tasks", "--decision", "approved", "--hand-off", "--commit-review", "stop-before-commit")
	if err != nil {
		t.Fatalf("gate record --hand-off --commit-review: %v", err)
	}

	ledger := handOffLedger(t, ws)
	if len(ledger.Records) != 1 {
		t.Fatalf("registros = %+v, se esperaba uno solo", ledger.Records)
	}
	record := ledger.Records[0]
	if !record.HandedOff || record.HandedOffTo != "apply" || record.CommitReview != kickoff.CommitReviewStopBeforeCommit {
		t.Errorf("registro = %+v, se esperaba la cesión y el modo juntos", record)
	}
	for _, want := range []string{"fase siguiente cedida: apply", "revisión por commit: stop-before-commit", "axiom sdd publish"} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida debe contener %q:\n%s", want, out)
		}
	}
}

// RF-GAT-20: la bandera en cualquier posición equivale a la bandera al final.
func TestGateRecordCommitReviewFlagPositionDoesNotMatter(t *testing.T) {
	ws := handOffProject(t, guardSealArgs)

	if _, err := gateRecord(t, ws, "--commit-review", "stop", "--gate", "tasks", "--decision", "approved"); err != nil {
		t.Fatalf("bandera al principio: %v", err)
	}

	if ledger := handOffLedger(t, ws); len(ledger.Records) != 1 || ledger.Records[0].CommitReview != kickoff.CommitReviewStop {
		t.Fatalf("registros = %+v", ledger.Records)
	}
}

// RF-GAT-20: sin la bandera no se escribe el campo, ni siquiera con la cesión.
func TestGateRecordWithoutCommitReviewNeverWritesTheField(t *testing.T) {
	ws := handOffProject(t, guardSealArgs)

	if _, err := gateRecord(t, ws, "--gate", "spec", "--decision", "approved"); err != nil {
		t.Fatalf("spec: %v", err)
	}
	if _, err := gateRecord(t, ws, "--gate", "tasks", "--decision", "approved", "--hand-off"); err != nil {
		t.Fatalf("tasks con cesión: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(guardChangeRoot(ws), "gates.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "commit_review") {
		t.Errorf("sin --commit-review no se escribe el campo:\n%s", data)
	}
}

// RF-GAT-20 y RF-GAT-24: cada rechazo ocurre sin escribir nada (gates.yaml, digest y kickoff.yaml
// idénticos byte a byte) y nombra la orden que sigue valiendo.
func TestGateRecordCommitReviewRefusals(t *testing.T) {
	t.Run("otra puerta solo admite tasks", func(t *testing.T) {
		ws := handOffProject(t, guardSealArgs)
		wantRefusedWithoutWriting(t, ws, []string{"--gate", "design", "--decision", "approved", "--commit-review", "notify"},
			"solo se admite en la compuerta tasks", "--gate design --decision approved")
	})

	for _, gate := range []string{"spec", "role-apply:core", "integration"} {
		t.Run("la compuerta "+gate+" rechaza la bandera", func(t *testing.T) {
			ws := handOffProject(t, guardSealArgs)
			wantRefusedWithoutWriting(t, ws, []string{"--gate", gate, "--decision", "approved", "--commit-review", "stop"},
				"solo se admite en la compuerta tasks")
		})
	}

	t.Run("un rechazo no admite el modo", func(t *testing.T) {
		ws := handOffProject(t, guardSealArgs)
		wantRefusedWithoutWriting(t, ws, []string{"--gate", "tasks", "--decision", "rejected", "--reason", "x", "--commit-review", "notify"},
			"solo acompaña a `--decision approved`", "--decision rejected")
	})

	t.Run("la evidencia no admite el modo", func(t *testing.T) {
		ws := handOffProject(t, guardSealArgs)
		wantRefusedWithoutWriting(t, ws, []string{"--gate", "tasks", "--decision", "approved", "--evidence-kind", "attestation", "--evidence", "x", "--commit-review", "stop"},
			"no admite `--evidence-kind`")
	})

	t.Run("un cambio continuo no tiene paradas", func(t *testing.T) {
		ws := handOffProject(t, continuousArgs)
		wantRefusedWithoutWriting(t, ws, []string{"--gate", "tasks", "--decision", "approved", "--commit-review", "stop"},
			"no tiene paradas", "checkpointed", "--decision approved")
		if _, err := os.Stat(filepath.Join(guardChangeRoot(ws), "gates.yaml")); err == nil {
			t.Error("un cambio continuo no debe tener gates.yaml tras el rechazo")
		}
	})

	t.Run("un cambio sin sellar no tiene paradas", func(t *testing.T) {
		ws := handOffProject(t, guardSealArgs)
		if err := os.Remove(filepath.Join(guardChangeRoot(ws), "kickoff.yaml")); err != nil {
			t.Fatal(err)
		}
		integrateCommit(t, ws.Master, "sin sellar")
		wantRefusedWithoutWriting(t, ws, []string{"--gate", "tasks", "--decision", "approved", "--commit-review", "stop"},
			"no tiene paradas", "no está sellado")
	})

	t.Run("una puerta tasks ya aprobada sin modo no admite otro registro con modo", func(t *testing.T) {
		ws := handOffProject(t, guardSealArgs)
		if _, err := gateRecord(t, ws, "--gate", "tasks", "--decision", "approved"); err != nil {
			t.Fatalf("aprobación previa: %v", err)
		}
		wantRefusedWithoutWriting(t, ws, []string{"--gate", "tasks", "--decision", "approved", "--commit-review", "stop"},
			"ya está resuelta", "axiom sdd status")
		if ledger := handOffLedger(t, ws); len(ledger.Records) != 1 || ledger.Records[0].CommitReview != "" {
			t.Errorf("registros = %+v, el modo efectivo debe seguir siendo notify", ledger.Records)
		}
	})

	t.Run("una puerta tasks ya aprobada con modo no admite cambiarlo", func(t *testing.T) {
		ws := handOffProject(t, guardSealArgs)
		if _, err := gateRecord(t, ws, "--gate", "tasks", "--decision", "approved", "--commit-review", "stop"); err != nil {
			t.Fatalf("aprobación previa: %v", err)
		}
		wantRefusedWithoutWriting(t, ws, []string{"--gate", "tasks", "--decision", "approved", "--hand-off", "--commit-review", "notify"},
			"ya está resuelta")
		if ledger := handOffLedger(t, ws); len(ledger.Records) != 1 || ledger.Records[0].CommitReview != kickoff.CommitReviewStop {
			t.Errorf("registros = %+v, el modo debe seguir siendo stop", ledger.Records)
		}
	})

	t.Run("valores y formas inválidas de la bandera no escriben", func(t *testing.T) {
		ws := handOffProject(t, guardSealArgs)
		for _, args := range [][]string{
			{"--commit-review", "Stop"},
			{"--commit-review", "pause"},
			{"--commit-review", ""},
			{"--commit-review"},
			{"--commit-review", "stop", "--commit-review", "stop"},
		} {
			full := append([]string{"--gate", "tasks", "--decision", "approved"}, args...)
			before := consentCLISnapshot(t, guardChangeRoot(ws))
			if out, err := gateRecord(t, ws, full...); err == nil {
				t.Errorf("%v debía rechazarse; salida:\n%s", args, out)
			}
			wantNoWrite(t, ws, before)
		}
	})
}

// RF-GAT-20: el kickoff y el digest de la puerta no cambian con el modo.
func TestGateRecordCommitReviewKeepsTheKickoffAndTheDigest(t *testing.T) {
	record := func(t *testing.T, extra ...string) (kickoff.GateRecord, string) {
		t.Helper()
		ws := handOffProject(t, guardSealArgs)
		kickoffPath := filepath.Join(guardChangeRoot(ws), "kickoff.yaml")
		before, err := os.ReadFile(kickoffPath)
		if err != nil {
			t.Fatal(err)
		}
		args := append([]string{"--gate", "tasks", "--decision", "approved"}, extra...)
		if _, err := gateRecord(t, ws, args...); err != nil {
			t.Fatalf("gate record: %v", err)
		}
		after, err := os.ReadFile(kickoffPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, after) {
			t.Errorf("kickoff.yaml no debe cambiar:\nantes:\n%s\ndespués:\n%s", before, after)
		}
		ledger := handOffLedger(t, ws)
		if len(ledger.Records) != 1 {
			t.Fatalf("registros = %+v", ledger.Records)
		}
		return ledger.Records[0], string(after)
	}

	withMode, _ := record(t, "--commit-review", "stop")
	withoutMode, _ := record(t)

	if withMode.ArtifactDigest == "" || withMode.ArtifactDigest != withoutMode.ArtifactDigest {
		t.Errorf("el digest debe ser idéntico con y sin modo: %q frente a %q", withMode.ArtifactDigest, withoutMode.ArtifactDigest)
	}
}

// RF-GAT-20: el `--json` lleva el modo cuando se persiste y no lo lleva en caso contrario.
func TestGateRecordCommitReviewJSON(t *testing.T) {
	ws := handOffProject(t, guardSealArgs)

	t.Run("sin bandera no hay clave", func(t *testing.T) {
		out, err := gateRecord(t, ws, "--gate", "design", "--decision", "approved", "--json")
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]any
		if err := json.Unmarshal([]byte(out), &document); err != nil {
			t.Fatalf("la salida no es JSON: %v\n%s", err, out)
		}
		if _, present := document["commitReview"]; present {
			t.Errorf("sin --commit-review no hay commitReview: %v", document)
		}
	})
	t.Run("con bandera", func(t *testing.T) {
		out, err := gateRecord(t, ws, "--gate", "tasks", "--decision", "approved", "--commit-review", "stop", "--json")
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]any
		if err := json.Unmarshal([]byte(out), &document); err != nil {
			t.Fatalf("la salida no es JSON: %v\n%s", err, out)
		}
		if document["commitReview"] != "stop" || document["gate"] != "tasks" || document["decision"] != "approved" {
			t.Errorf("documento = %v", document)
		}
		if _, present := document["handedOff"]; present {
			t.Errorf("sin cesión no hay handedOff: %v", document)
		}
	})
}

// RF-GAT-25: la ayuda describe la bandera, su vocabulario y su restricción.
func TestGateRecordHelpDocumentsCommitReview(t *testing.T) {
	var out bytes.Buffer
	if err := RunSDDGate([]string{"record", "--help"}, &out); err != nil {
		t.Fatalf("RunSDDGate --help: %v", err)
	}

	help := out.String()
	for _, want := range []string{
		"--commit-review notify|stop|stop-before-commit",
		"--gate tasks --decision approved",
		"checkpointed",
		"sin la bandera equivale a notify",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("la ayuda debe contener %q:\n%s", want, help)
		}
	}
}
