package kickoff

import (
	"testing"
)

// Pruebas de compatibilidad de la capacidad change-actor-identity (RF-ACT-04, RF-ACT-07,
// RF-ACT-08): el dominio no valida el valor del actor ni del sellador, así que los registros
// anteriores y los nuevos con la identidad git conviven.

func TestLegacyActorRecordsStillLoadAndValidate(t *testing.T) {
	changeRoot := t.TempDir()
	for _, actor := range []string{"cli", "maintainer", "release-manager"} {
		if err := AppendGate(changeRoot, GateRecord{Gate: GateSpec, Decision: DecisionRejected, Reason: "anterior", Actor: actor}); err != nil {
			t.Fatalf("AppendGate(actor %q) error = %v", actor, err)
		}
	}
	ledger, err := LoadGates(changeRoot)
	if err != nil {
		t.Fatalf("LoadGates() error = %v", err)
	}
	if len(ledger.Records) != 3 || ledger.Records[0].Actor != "cli" || ledger.Records[1].Actor != "maintainer" || ledger.Records[2].Actor != "release-manager" {
		t.Fatalf("Records = %+v, se esperaban los tres actores anteriores intactos", ledger.Records)
	}

	for _, sealedBy := range []string{"cli", SealedByInferred} {
		root := t.TempDir()
		k := sampleKickoff("inc-99-example")
		k.SealedBy = sealedBy
		if _, wrote, err := Seal(root, k); err != nil || !wrote {
			t.Fatalf("Seal(sealed_by %q) = wrote %v, error %v", sealedBy, wrote, err)
		}
		loaded, err := Load(root)
		if err != nil || loaded == nil {
			t.Fatalf("Load() = %v, error %v", loaded, err)
		}
		if loaded.SealedBy != sealedBy {
			t.Errorf("SealedBy = %q, se esperaba %q", loaded.SealedBy, sealedBy)
		}
		if err := loaded.Validate(); err != nil {
			t.Errorf("Validate(sealed_by %q) error = %v", sealedBy, err)
		}
	}
}

func TestNewActorRecordCoexistsWithLegacy(t *testing.T) {
	changeRoot := t.TempDir()
	if err := AppendGate(changeRoot, GateRecord{Gate: GateSpec, Decision: DecisionRejected, Reason: "anterior", Actor: "cli"}); err != nil {
		t.Fatal(err)
	}
	if err := AppendGate(changeRoot, GateRecord{Gate: GateSpec, Decision: DecisionApproved, Actor: "Ana Pérez <ana@example.com>"}); err != nil {
		t.Fatal(err)
	}
	ledger, err := LoadGates(changeRoot)
	if err != nil {
		t.Fatalf("LoadGates() error = %v", err)
	}
	if len(ledger.Records) != 2 || ledger.Records[0].Actor != "cli" || ledger.Records[1].Actor != "Ana Pérez <ana@example.com>" {
		t.Fatalf("Records = %+v, el registro cli debe quedar intacto junto al nuevo", ledger.Records)
	}
}

func TestIdentityFormatActorValidates(t *testing.T) {
	k := sampleKickoff("inc-99-example")
	k.SealedBy = "Ana Pérez <ana@example.com>"
	if err := k.ValidateForSeal(); err != nil {
		t.Fatalf("ValidateForSeal() con identidad git error = %v", err)
	}
	ledger := GateLedger{Schema: GateLedgerSchemaV1, Records: []GateRecord{{Gate: GateSpec, Decision: DecisionApproved, Actor: "Ana Pérez <ana@example.com>"}}}
	if err := ledger.Validate(); err != nil {
		t.Fatalf("GateLedger.Validate() con identidad git error = %v", err)
	}
}

func TestActorIdentityYAMLRoundTrip(t *testing.T) {
	const actor = "Íñigo: Gutiérrez # Zufia <inigo@example.com>"

	changeRoot := t.TempDir()
	if err := AppendGate(changeRoot, GateRecord{Gate: GateSpec, Decision: DecisionApproved, Actor: actor}); err != nil {
		t.Fatalf("AppendGate() error = %v", err)
	}
	ledger, err := LoadGates(changeRoot)
	if err != nil {
		t.Fatalf("LoadGates() error = %v", err)
	}
	if len(ledger.Records) != 1 || ledger.Records[0].Actor != actor {
		t.Fatalf("Actor = %+v, se esperaba %q sin pérdida", ledger.Records, actor)
	}

	sealRoot := t.TempDir()
	k := sampleKickoff("inc-99-example")
	k.SealedBy = actor
	if _, wrote, err := Seal(sealRoot, k); err != nil || !wrote {
		t.Fatalf("Seal() = wrote %v, error %v", wrote, err)
	}
	loaded, err := Load(sealRoot)
	if err != nil || loaded == nil {
		t.Fatalf("Load() = %v, error %v", loaded, err)
	}
	if loaded.SealedBy != actor {
		t.Fatalf("SealedBy = %q, se esperaba %q sin pérdida", loaded.SealedBy, actor)
	}
}

func TestValidateForSealKeepsInferredExemption(t *testing.T) {
	if err := sealRosterKickoff(ExecutionContinuous, SealedByInferred, "core", "web").ValidateForSeal(); err != nil {
		t.Fatalf("ValidateForSeal() de un retrosellado error = %v, debe seguir exento", err)
	}
	// Una identidad git no es el valor reservado: no activa la exención.
	if err := sealRosterKickoff(ExecutionContinuous, "Ana Pérez <ana@example.com>", "core", "web").ValidateForSeal(); err == nil {
		t.Fatal("ValidateForSeal() = nil, la identidad git no debe activar la exención del retrosellado")
	}
}

func TestToKickoffSealedByIsFallbackActor(t *testing.T) {
	parsed, err := ParseSealArgs([]string{"--cwd", "/repo", "--change", "inc-99-example", "--execution-style", "checkpointed"})
	if err != nil {
		t.Fatalf("ParseSealArgs() error = %v", err)
	}
	if got := parsed.ToKickoff("inc-99-example").SealedBy; got != FallbackActor {
		t.Fatalf("ToKickoff().SealedBy = %q, se esperaba %q", got, FallbackActor)
	}
}
