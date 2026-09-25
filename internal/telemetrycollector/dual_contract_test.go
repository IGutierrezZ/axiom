package telemetrycollector

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v3/internal/telemetry"
)

func TestDualContract_EventsAndRuntimeHandlers(t *testing.T) {
	s, err := OpenStorage(filepath.Join(t.TempDir(), "dual_events.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	server := &Server{Storage: s, Limiter: NewRateLimiter(100)}
	mux := server.NewMux()

	send := func(path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		r.RemoteAddr = "192.0.2.123:4321"
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}

	t.Run("Event schemas acceptance and rejection", func(t *testing.T) {
		// 1. Canonical Axiom telemetry event
		axiomEvent := strings.Replace(validInstallEvent, "gentle-ai.telemetry-event/v1", "axiom.telemetry-event/v1", 1)
		wAxiom := send("/v1/events", axiomEvent)
		if wAxiom.Code != http.StatusAccepted {
			t.Fatalf("expected 202 Accepted for Axiom event, got %d", wAxiom.Code)
		}

		// 2. Legacy Gentle AI telemetry event
		legacyEvent := validInstallEvent
		wLegacy := send("/v1/events", legacyEvent)
		if wLegacy.Code != http.StatusAccepted {
			t.Fatalf("expected 202 Accepted for legacy event, got %d", wLegacy.Code)
		}

		// 3. Unknown schema event (fail-closed)
		unknownEvent := strings.Replace(validInstallEvent, "gentle-ai.telemetry-event/v1", "unknown.telemetry-event/v1", 1)
		wUnknown := send("/v1/events", unknownEvent)
		if wUnknown.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for unknown schema, got %d", wUnknown.Code)
		}
	})

	t.Run("Runtime event schemas acceptance and ack negotiation", func(t *testing.T) {
		rawFixture := string(runtimeFixture())

		// 1. Canonical Axiom runtime event
		axiomFixture := strings.Replace(rawFixture, "gentle-ai.telemetry-runtime-event/v1", telemetry.AxiomRuntimeEventSchema, 1)
		axiomFixture = strings.Replace(axiomFixture, "0123456789abcdef0123456789abcdef", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 1)
		wAxiom := send("/v1/runtime-events", axiomFixture)
		if wAxiom.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for Axiom runtime event, got %d", wAxiom.Code)
		}
		var ackAxiom map[string]string
		if err := json.Unmarshal(wAxiom.Body.Bytes(), &ackAxiom); err != nil {
			t.Fatal(err)
		}
		if ackAxiom["schema"] != telemetry.AxiomRuntimeDeliverySchema {
			t.Fatalf("expected ack schema %q, got %q", telemetry.AxiomRuntimeDeliverySchema, ackAxiom["schema"])
		}
		if ackAxiom["decision"] != "stored" {
			t.Fatalf("expected decision 'stored', got %q", ackAxiom["decision"])
		}

		// 2. Legacy Gentle AI runtime event
		legacyFixture := strings.Replace(rawFixture, "0123456789abcdef0123456789abcdef", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 1)
		wLegacy := send("/v1/runtime-events", legacyFixture)
		if wLegacy.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for legacy runtime event, got %d", wLegacy.Code)
		}
		var ackLegacy map[string]string
		if err := json.Unmarshal(wLegacy.Body.Bytes(), &ackLegacy); err != nil {
			t.Fatal(err)
		}
		if ackLegacy["schema"] != telemetry.LegacyRuntimeDeliverySchema {
			t.Fatalf("expected ack schema %q, got %q", telemetry.LegacyRuntimeDeliverySchema, ackLegacy["schema"])
		}
		if ackLegacy["decision"] != "stored" {
			t.Fatalf("expected decision 'stored', got %q", ackLegacy["decision"])
		}

		// 3. Unknown runtime schema (fail-closed)
		unknownFixture := strings.Replace(rawFixture, "gentle-ai.telemetry-runtime-event/v1", "unknown.telemetry-runtime-event/v1", 1)
		wUnknown := send("/v1/runtime-events", unknownFixture)
		if wUnknown.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for unknown runtime schema, got %d", wUnknown.Code)
		}
	})
}
