package kickoff

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestLoadGatesAbsentFileReturnsEmptyLedgerNoError(t *testing.T) {
	changeRoot := t.TempDir()
	ledger, err := LoadGates(changeRoot)
	if err != nil {
		t.Fatalf("LoadGates() error = %v, se esperaba nil", err)
	}
	if len(ledger.Records) != 0 {
		t.Fatalf("Records = %+v, se esperaba vacio sin fichero", ledger.Records)
	}
}

func TestAppendGatePreservesPreviousRecords(t *testing.T) {
	changeRoot := t.TempDir()
	if err := AppendGate(changeRoot, GateRecord{Gate: GateSpec, Decision: DecisionRejected, Reason: "primero", Actor: "maintainer"}); err != nil {
		t.Fatalf("primer AppendGate() error = %v", err)
	}
	if err := AppendGate(changeRoot, GateRecord{Gate: GateSpec, Decision: DecisionApproved, Reason: "segundo tras remediar", Actor: "maintainer"}); err != nil {
		t.Fatalf("segundo AppendGate() error = %v", err)
	}

	ledger, err := LoadGates(changeRoot)
	if err != nil {
		t.Fatalf("LoadGates() error = %v", err)
	}
	if len(ledger.Records) != 2 {
		t.Fatalf("Records = %+v, se esperaban 2 (el anexado no debe perder el primero)", ledger.Records)
	}
	if ledger.Records[0].Reason != "primero" || ledger.Records[1].Reason != "segundo tras remediar" {
		t.Fatalf("orden/registros = %+v, se esperaba preservar el primero y anadir el segundo", ledger.Records)
	}
}

// TestAppendGateConcurrentWritesLoseNoRecords must be run with -race: two
// goroutines append to the SAME ledger concurrently, and AppendGate's
// exclusive lock must serialize them so neither append is lost.
func TestAppendGateConcurrentWritesLoseNoRecords(t *testing.T) {
	changeRoot := t.TempDir()
	const n = 20
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = AppendGate(changeRoot, GateRecord{
				Gate:     GateSpec,
				Decision: DecisionApproved,
				Reason:   fmt.Sprintf("motivo-%d", i),
				Actor:    "maintainer",
			})
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("AppendGate() goroutine %d error = %v", i, err)
		}
	}

	ledger, err := LoadGates(changeRoot)
	if err != nil {
		t.Fatalf("LoadGates() error = %v", err)
	}
	if len(ledger.Records) != n {
		t.Fatalf("len(Records) = %d, se esperaban %d (ninguno perdido bajo concurrencia)", len(ledger.Records), n)
	}
	seenReasons := make(map[string]bool, n)
	for _, r := range ledger.Records {
		seenReasons[r.Reason] = true
	}
	if len(seenReasons) != n {
		t.Fatalf("se esperaban %d motivos distintos, se vieron %d (registro duplicado o perdido)", n, len(seenReasons))
	}
}

func TestAppendGateRejectsInvalidRecord(t *testing.T) {
	changeRoot := t.TempDir()
	err := AppendGate(changeRoot, GateRecord{Gate: GateKey("bogus"), Decision: DecisionApproved, Actor: "maintainer"})
	if err == nil {
		t.Fatal("AppendGate() con una clave de compuerta desconocida debia devolver error")
	}
	ledger, loadErr := LoadGates(changeRoot)
	if loadErr != nil {
		t.Fatalf("LoadGates() error = %v", loadErr)
	}
	if len(ledger.Records) != 0 {
		t.Fatalf("Records = %+v, un registro invalido no debio anexarse", ledger.Records)
	}
}

// legacyGateRecord es el registro de gates.yaml tal y como lo declaraba el
// esquema anterior a la salida 2 (sin handed_off). Se congela aqui a proposito:
// prueba que un binario anterior sigue leyendo un ledger con la clave nueva
// (REQ-26.3) y que los registros sin marca no cambian ni un byte.
type legacyGateRecord struct {
	Gate            GateKey      `yaml:"gate"`
	Decision        GateDecision `yaml:"decision"`
	Reason          string       `yaml:"reason"`
	ArtifactDigest  string       `yaml:"artifact_digest,omitempty"`
	EvidenceKind    EvidenceKind `yaml:"evidence_kind,omitempty"`
	EvidenceRef     string       `yaml:"evidence_ref,omitempty"`
	EvidenceBaseRef string       `yaml:"evidence_base_ref,omitempty"`
	Verified        bool         `yaml:"verified,omitempty"`
	Actor           string       `yaml:"actor"`
	RecordedAt      time.Time    `yaml:"recorded_at"`
}

type legacyGateLedger struct {
	Schema  string             `yaml:"schema"`
	Change  string             `yaml:"change"`
	Records []legacyGateRecord `yaml:"records"`
}

func TestGateRecordWithoutHandedOffSerializesLikeTheLegacySchema(t *testing.T) {
	at := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	current, err := yaml.Marshal(GateLedger{
		Schema: GateLedgerSchemaV1,
		Change: "x",
		Records: []GateRecord{
			{Gate: GateSpec, Decision: DecisionApproved, Reason: "ok", ArtifactDigest: "d1", Actor: "maintainer", RecordedAt: at},
			{Gate: GateDesign, Decision: DecisionRejected, Reason: "falta", ArtifactDigest: "d2", Actor: "maintainer", RecordedAt: at},
		},
	})
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}
	legacy, err := yaml.Marshal(legacyGateLedger{
		Schema: GateLedgerSchemaV1,
		Change: "x",
		Records: []legacyGateRecord{
			{Gate: GateSpec, Decision: DecisionApproved, Reason: "ok", ArtifactDigest: "d1", Actor: "maintainer", RecordedAt: at},
			{Gate: GateDesign, Decision: DecisionRejected, Reason: "falta", ArtifactDigest: "d2", Actor: "maintainer", RecordedAt: at},
		},
	})
	if err != nil {
		t.Fatalf("yaml.Marshal() anterior error = %v", err)
	}
	if string(current) != string(legacy) {
		t.Fatalf("los registros sin marca deben serializar byte a byte como antes\nactual:\n%s\nanterior:\n%s", current, legacy)
	}
	if strings.Contains(string(current), "handed_off") {
		t.Fatalf("handed_off no debe aparecer en un registro sin cesion:\n%s", current)
	}
}

func TestGateRecordHandedOffIsWrittenAndReadBack(t *testing.T) {
	changeRoot := t.TempDir()
	if err := AppendGate(changeRoot, GateRecord{Gate: GateSpec, Decision: DecisionApproved, ArtifactDigest: "d1", HandedOff: true, Actor: "maintainer"}); err != nil {
		t.Fatalf("AppendGate() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(changeRoot, GatesFileName))
	if err != nil {
		t.Fatalf("leer gates.yaml: %v", err)
	}
	if !strings.Contains(string(data), "handed_off: true") {
		t.Fatalf("gates.yaml = %s, se esperaba handed_off: true", data)
	}
	ledger, err := LoadGates(changeRoot)
	if err != nil {
		t.Fatalf("LoadGates() error = %v", err)
	}
	if len(ledger.Records) != 1 || !ledger.Records[0].HandedOff {
		t.Fatalf("Records = %+v, se esperaba un registro con HandedOff", ledger.Records)
	}
}

// TestLegacyBinaryReadsAGatesFileWithHandedOff reproduce la lectura de un
// binario anterior: decodifica con el struct congelado, sin KnownFields, y la
// puerta resulta approved aunque no vea la cesion (REQ-26.3).
func TestLegacyBinaryReadsAGatesFileWithHandedOff(t *testing.T) {
	data := []byte("schema: axiom.sdd-gate-ledger/v1\n" +
		"change: x\n" +
		"records:\n" +
		"  - gate: spec\n" +
		"    decision: approved\n" +
		"    reason: ok\n" +
		"    artifact_digest: d1\n" +
		"    handed_off: true\n" +
		"    actor: maintainer\n" +
		"    recorded_at: 2026-11-03T10:00:00Z\n")
	var legacy legacyGateLedger
	if err := yaml.Unmarshal(data, &legacy); err != nil {
		t.Fatalf("el struct anterior no decodifica la clave nueva: %v", err)
	}
	if len(legacy.Records) != 1 || legacy.Records[0].Decision != DecisionApproved {
		t.Fatalf("legacy = %+v, se esperaba un approved", legacy)
	}

	// El binario nuevo tambien lo lee, con la marca, y la puerta resulta approved.
	current, err := ParseGateLedger(data)
	if err != nil {
		t.Fatalf("ParseGateLedger() error = %v", err)
	}
	if !current.Records[0].HandedOff {
		t.Fatalf("Records = %+v, se esperaba HandedOff", current.Records)
	}
	states, err := EvaluateGates(Inputs{
		Execution: ExecutionCheckpointed,
		Artifacts: map[string]string{"spec": "d1"},
		Ledger:    current,
	})
	if err != nil {
		t.Fatalf("EvaluateGates() error = %v", err)
	}
	if state := findGateState(t, states, GateSpec); state.Status != "approved" || !state.HandedOff {
		t.Fatalf("estado = %+v, se esperaba approved con cesion", state)
	}
}

// TestHandedOffToIsPersistedWithTheMarkAndReadByTheFrozenSchema fija que la fase
// cedida se escribe como handed_off_to junto a handed_off, vuelve en
// GateState.HandedOffTo y que el struct congelado anterior la ignora (REQ-26.3).
func TestHandedOffToIsPersistedWithTheMarkAndReadByTheFrozenSchema(t *testing.T) {
	changeRoot := t.TempDir()
	if err := AppendGate(changeRoot, GateRecord{Gate: GateSpec, Decision: DecisionApproved, ArtifactDigest: "d1", HandedOff: true, HandedOffTo: "design", Actor: "maintainer"}); err != nil {
		t.Fatalf("AppendGate() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(changeRoot, GatesFileName))
	if err != nil {
		t.Fatalf("leer gates.yaml: %v", err)
	}
	if !strings.Contains(string(data), "handed_off_to: design") {
		t.Fatalf("gates.yaml = %s, se esperaba handed_off_to: design", data)
	}

	var legacy legacyGateLedger
	if err := yaml.Unmarshal(data, &legacy); err != nil {
		t.Fatalf("el struct anterior no decodifica la clave nueva: %v", err)
	}
	if len(legacy.Records) != 1 || legacy.Records[0].Decision != DecisionApproved {
		t.Fatalf("legacy = %+v, se esperaba un approved", legacy)
	}

	ledger, err := LoadGates(changeRoot)
	if err != nil {
		t.Fatalf("LoadGates() error = %v", err)
	}
	states, err := EvaluateGates(Inputs{Execution: ExecutionCheckpointed, Artifacts: map[string]string{"spec": "d1"}, Ledger: ledger})
	if err != nil {
		t.Fatalf("EvaluateGates() error = %v", err)
	}
	if state := findGateState(t, states, GateSpec); !state.HandedOff || state.HandedOffTo != "design" {
		t.Fatalf("estado = %+v, se esperaba la cesion persistida design", state)
	}
}

// Las salidas 1 y 3 no escriben ni handed_off ni handed_off_to.
func TestExitsOneAndThreeWriteNeitherHandOffKey(t *testing.T) {
	changeRoot := t.TempDir()
	for _, record := range []GateRecord{
		{Gate: GateSpec, Decision: DecisionRejected, Reason: "falta", ArtifactDigest: "d1", Actor: "maintainer"},
		{Gate: GateDesign, Decision: DecisionApproved, ArtifactDigest: "d2", Actor: "maintainer"},
	} {
		if err := AppendGate(changeRoot, record); err != nil {
			t.Fatalf("AppendGate() error = %v", err)
		}
	}
	data, err := os.ReadFile(filepath.Join(changeRoot, GatesFileName))
	if err != nil {
		t.Fatalf("leer gates.yaml: %v", err)
	}
	if strings.Contains(string(data), "handed_off") {
		t.Fatalf("gates.yaml no debe llevar ninguna clave de cesion:\n%s", data)
	}
}

func TestParseGateLedgerStillRejectsAnUnknownDecision(t *testing.T) {
	data := []byte("schema: axiom.sdd-gate-ledger/v1\n" +
		"change: x\n" +
		"records:\n" +
		"  - gate: spec\n" +
		"    decision: hand_off\n" +
		"    handed_off: true\n" +
		"    actor: maintainer\n")
	if _, err := ParseGateLedger(data); err == nil {
		t.Fatal("ParseGateLedger() acepto una decision distinta de approved|rejected")
	}
}

// TestGateLockRetryBackoffGrowsCapsAndStaysAboveBase pins the three
// properties AppendGate's contention retry depends on. A flat delay makes
// every contender retry on the same tick and keep colliding, which is what
// drained the budget under twenty-way contention on a loaded CI machine.
func TestGateLockRetryBackoffGrowsCapsAndStaysAboveBase(t *testing.T) {
	original := randomInt63n
	t.Cleanup(func() { randomInt63n = original })

	// No jitter: the raw backoff curve is what this case pins.
	randomInt63n = func(int64) int64 { return 0 }

	first := gateLockRetryBackoff(0)
	if first != gateLockRetryDelay {
		t.Fatalf("gateLockRetryBackoff(0) = %v, se esperaba el retardo base %v", first, gateLockRetryDelay)
	}

	previous := first
	for attempt := 1; attempt <= 6; attempt++ {
		got := gateLockRetryBackoff(attempt)
		if got < previous {
			t.Fatalf("gateLockRetryBackoff(%d) = %v, menor que el intento anterior %v: el backoff debe crecer", attempt, got, previous)
		}
		previous = got
	}

	for _, attempt := range []int{20, 100, gateLockAcquireAttempts - 1} {
		if got := gateLockRetryBackoff(attempt); got > gateLockMaxRetryDelay {
			t.Fatalf("gateLockRetryBackoff(%d) = %v, por encima del techo %v", attempt, got, gateLockMaxRetryDelay)
		}
	}

	// Maximum jitter must never drive a retry below the base delay, or a
	// contender would spin hot instead of yielding.
	randomInt63n = func(n int64) int64 {
		if n <= 0 {
			return 0
		}
		return n - 1
	}
	for attempt := 0; attempt <= 12; attempt++ {
		if got := gateLockRetryBackoff(attempt); got < gateLockRetryDelay {
			t.Fatalf("con jitter maximo, gateLockRetryBackoff(%d) = %v, por debajo del retardo base %v", attempt, got, gateLockRetryDelay)
		}
	}
}
