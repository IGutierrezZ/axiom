package integrations

import (
	"errors"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

const digestA = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
const digestB = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"

func TestFingerprint(t *testing.T) {
	tests := []struct {
		name    string
		digest  string
		ordinal int
		want    string
	}{
		{"con digest usa los 16 primeros hex", digestA, 3, "d0123456789abcdef"},
		{"otro digest da otra huella", digestB, 3, "dfedcba9876543210"},
		{"digest vacío usa el ordinal", "", 1, "o1"},
		{"digest vacío con ordinal 4", "", 4, "o4"},
		{"digest corto se usa entero", "abc", 1, "dabc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Fingerprint(tt.digest, tt.ordinal); got != tt.want {
				t.Fatalf("Fingerprint(%q, %d) = %q, quería %q", tt.digest, tt.ordinal, got, tt.want)
			}
		})
	}
}

func TestIdempotencyKeys(t *testing.T) {
	t.Run("gate.spec.approved con digest", func(t *testing.T) {
		got := GateApprovedKey(manifest.EventGateSpecApproved, "x", Fingerprint(digestA, 1))
		if want := "gate.spec.approved/x/d0123456789abcdef"; got != want {
			t.Fatalf("clave = %q, quería %q", got, want)
		}
	})
	t.Run("gate.design.approved con respaldo ordinal", func(t *testing.T) {
		got := GateApprovedKey(manifest.EventGateDesignApproved, "x", Fingerprint("", 2))
		if want := "gate.design.approved/x/o2"; got != want {
			t.Fatalf("clave = %q, quería %q", got, want)
		}
	})
	t.Run("role.apply.approved", func(t *testing.T) {
		got := RoleApplyApprovedKey("x", "backend", Fingerprint(digestA, 1))
		if want := "role.apply.approved/x/backend/d0123456789abcdef"; got != want {
			t.Fatalf("clave = %q, quería %q", got, want)
		}
	})
	t.Run("role.start exacto", func(t *testing.T) {
		if got := RoleStartKey("x", "backend"); got != "role.start/x/backend" {
			t.Fatalf("clave = %q", got)
		}
	})
	t.Run("archive exacto", func(t *testing.T) {
		if got := ArchiveKey("x", "2026-11-19-x"); got != "archive/x/2026-11-19-x" {
			t.Fatalf("clave = %q", got)
		}
	})
	t.Run("change.start solo catálogo", func(t *testing.T) {
		if got := ChangeStartKey("x"); got != "change.start/x" {
			t.Fatalf("clave = %q", got)
		}
	})
	t.Run("la misma aprobación da la misma clave y otro digest da otra", func(t *testing.T) {
		a1 := GateApprovedKey(manifest.EventGateSpecApproved, "x", Fingerprint(digestA, 1))
		a2 := GateApprovedKey(manifest.EventGateSpecApproved, "x", Fingerprint(digestA, 7))
		b := GateApprovedKey(manifest.EventGateSpecApproved, "x", Fingerprint(digestB, 1))
		if a1 != a2 {
			t.Fatalf("el ordinal no cuenta si hay digest: %q != %q", a1, a2)
		}
		if a1 == b {
			t.Fatalf("un digest distinto debe dar otra clave: %q", b)
		}
	})
}

func TestNewEventValidatesCatalog(t *testing.T) {
	ev, err := NewEvent("gate.spec.approved", "x")
	if err != nil {
		t.Fatalf("NewEvent válido: %v", err)
	}
	if ev.Type != manifest.EventGateSpecApproved || ev.Change != "x" {
		t.Fatalf("evento = %#v", ev)
	}

	_, err = NewEvent("gate.tasks.approved", "x")
	if err == nil {
		t.Fatal("un evento fuera del catálogo no se puede construir")
	}
	if !errors.Is(err, ErrUnknownEvent) {
		t.Fatalf("el error debe ser ErrUnknownEvent: %v", err)
	}
	var typed *UnknownEventError
	if !errors.As(err, &typed) || typed.Name != "gate.tasks.approved" {
		t.Fatalf("error tipado con el nombre rechazado, obtuvo %#v", err)
	}
	for _, admitted := range manifest.Events() {
		if !strings.Contains(err.Error(), string(admitted)) {
			t.Errorf("el error debe nombrar el evento admitido %q: %s", admitted, err)
		}
	}
	if !strings.Contains(err.Error(), "gate.tasks.approved") {
		t.Errorf("el error debe nombrar el evento rechazado: %s", err)
	}

	if _, err := NewEvent("archive", ""); err == nil {
		t.Fatal("el cambio no puede estar vacío")
	}
}

func TestPolicyPerEvent(t *testing.T) {
	tests := []struct {
		event                                  EventType
		emittable, needsRole, needsGate, store bool
	}{
		{manifest.EventChangeStart, false, false, false, true},
		{manifest.EventGateSpecApproved, true, false, true, true},
		{manifest.EventGateDesignApproved, true, false, true, true},
		{manifest.EventRoleStart, true, true, false, true},
		{manifest.EventRoleApplyApproved, true, true, true, true},
		{manifest.EventArchive, true, false, false, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.event), func(t *testing.T) {
			p := PolicyFor(tt.event)
			if p.Emittable != tt.emittable || p.NeedsRole != tt.needsRole || p.NeedsApprovedGate != tt.needsGate || p.PersistOutcome != tt.store {
				t.Fatalf("PolicyFor(%s) = %+v", tt.event, p)
			}
		})
	}
	if got := PolicyFor("desconocido"); got != (EventPolicy{}) {
		t.Fatalf("un evento desconocido no tiene política: %+v", got)
	}
}
