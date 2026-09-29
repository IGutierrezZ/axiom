package knowledge

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatchesKeyword_WordBoundaries(t *testing.T) {
	tests := []struct {
		text     string
		kw       string
		expected bool
	}{
		// Falsos positivos por subcadenas que deben ser RECHAZADOS
		{"type CalendarService struct {}", "dar", false},
		{"var standardConfig = true", "dar", false},
		{"func SecondaryLookup() {}", "dar", false},
		{"radarSignal := 42", "dar", false},
		{"alendar item", "alen", false},

		// Coincidencias legítimas con límites de palabra / identificadores
		{"dar", "dar", true},
		{"// Función para dar de alta entidades", "dar", true},
		{"func darDeAlta() error", "dar", true},
		{"func ProcessRefund(id string)", "refund", true},
		{"type RefundProcessor struct", "refund", true},
		{"process_refund_v2", "refund", true},
		{"calculate_tarifa_plana_envio", "tarifa_plana_envio", true},
		{"tarifa_plana_envio", "tarifa", true},
		{"tarifa_plana_envio", "plana", true},
		{"tarifa_plana_envio", "envio", true},
		{"HTTPServer", "server", true},
	}

	for _, tt := range tests {
		got := matchesKeyword(tt.text, tt.kw)
		if got != tt.expected {
			t.Errorf("matchesKeyword(%q, %q) = %v; esperado %v", tt.text, tt.kw, got, tt.expected)
		}
	}
}

func TestExtractKeywords_FiltersStopWords(t *testing.T) {
	input := "¿Dime cuáles son los campos para dar de alta en calendar?"
	keywords := extractKeywords(input)

	// "dime", "cuáles", "son", "los", "para", "dar", "de", "en" deben ser filtrados
	for _, kw := range keywords {
		if kw == "dar" || kw == "dime" || kw == "cuáles" || kw == "son" || kw == "los" || kw == "para" || kw == "de" || kw == "en" {
			t.Errorf("palabra vacía %q no fue filtrada de las keywords: %v", kw, keywords)
		}
	}

	hasCampos := false
	hasCalendar := false
	hasAlta := false
	for _, kw := range keywords {
		if kw == "campos" {
			hasCampos = true
		}
		if kw == "calendar" {
			hasCalendar = true
		}
		if kw == "alta" {
			hasAlta = true
		}
	}

	if !hasCampos || !hasCalendar || !hasAlta {
		t.Errorf("se esperaban keywords 'campos', 'alta' y 'calendar', obtenidas: %v", keywords)
	}
}

func TestQuery_NoFalsePositivesInCalendar(t *testing.T) {
	tempWs := t.TempDir()

	// Módulo alfabéticamente anterior pero irrelevante (Balaruc / Calendar)
	balarucDir := filepath.Join(tempWs, "internal", "balaruc")
	_ = os.MkdirAll(balarucDir, 0755)
	calendarCode := `package balaruc

// CalendarService maneja eventos del calendario
type CalendarService struct {
	StandardView bool
	SecondaryColor string
}
`
	_ = os.WriteFile(filepath.Join(balarucDir, "calendar.go"), []byte(calendarCode), 0644)

	// Módulo relevante: Identity / Users
	identityDir := filepath.Join(tempWs, "internal", "identity")
	_ = os.MkdirAll(identityDir, 0755)
	userCode := `package identity

// AltaUsuario define los campos para dar de alta un nuevo usuario en el sistema
type AltaUsuario struct {
	Username string
	Email    string
	Role     string
}
`
	_ = os.WriteFile(filepath.Join(identityDir, "users.go"), []byte(userCode), 0644)

	// Consulta en lenguaje natural
	res, err := RunQuery(context.Background(), QueryOptions{
		WorkspaceRoot: tempWs,
		Question:      "¿Dime cuáles son los campos para dar de alta en usuarios?",
	})
	if err != nil {
		t.Fatalf("RunQuery falló: %v", err)
	}

	if len(res.Evidences) == 0 {
		t.Fatalf("se esperaban evidencias de código")
	}

	// La evidencia principal DEBE provenir de identity/users.go, NUNCA de balaruc/calendar.go
	firstEv := res.Evidences[0]
	if !strings.Contains(firstEv.File, "identity") {
		t.Errorf("la primera evidencia debería pertenecer a identity, obtenida: %s (Context: %s)", firstEv.File, firstEv.Context)
	}

	for _, ev := range res.Evidences {
		if strings.Contains(ev.File, "balaruc") {
			t.Errorf("evidencia espuria hallada en balaruc/calendar.go debido a falso positivo: %v", ev)
		}
	}
}

func TestQuery_ReadOnlyByDefault(t *testing.T) {
	tempWs := t.TempDir()
	specsRoot := filepath.Join(tempWs, "openspec")

	billingDir := filepath.Join(tempWs, "internal", "billing")
	_ = os.MkdirAll(billingDir, 0755)
	_ = os.WriteFile(filepath.Join(billingDir, "invoice.go"), []byte("package billing\n// regla_facturacion_anual activa\n"), 0644)

	// Por defecto Enrich es false (CQS: solo lectura)
	res, err := RunQuery(context.Background(), QueryOptions{
		WorkspaceRoot: tempWs,
		SpecsRoot:     specsRoot,
		Question:      "¿Cuál es la regla_facturacion_anual?",
		Enrich:        false,
	})
	if err != nil {
		t.Fatalf("RunQuery falló: %v", err)
	}

	if res.SpecUpdated {
		t.Errorf("SpecUpdated debe ser false por defecto en operaciones de consulta")
	}

	// Comprobar que NO se creó ningún fichero de especificación en disco
	specPath := filepath.Join(specsRoot, "specs", "billing", "spec.md")
	if _, err := os.Stat(specPath); !os.IsNotExist(err) {
		t.Errorf("la especificación viva %s NO debió crearse en una consulta de solo lectura", specPath)
	}
}

func TestQuery_ExplicitEnrich(t *testing.T) {
	tempWs := t.TempDir()
	specsRoot := filepath.Join(tempWs, "openspec")

	billingDir := filepath.Join(tempWs, "internal", "billing")
	_ = os.MkdirAll(billingDir, 0755)
	_ = os.WriteFile(filepath.Join(billingDir, "invoice.go"), []byte("package billing\n// regla_facturacion_anual activa\n"), 0644)

	// Consulta con Enrich explícito en true
	res, err := RunQuery(context.Background(), QueryOptions{
		WorkspaceRoot: tempWs,
		SpecsRoot:     specsRoot,
		Question:      "¿Cuál es la regla_facturacion_anual?",
		Enrich:        true,
	})
	if err != nil {
		t.Fatalf("RunQuery falló: %v", err)
	}

	if !res.SpecUpdated {
		t.Errorf("se esperaba SpecUpdated=true cuando Enrich=true")
	}

	specPath := filepath.Join(specsRoot, "specs", "billing", "spec.md")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Errorf("la especificación viva %s debió crearse al activar Enrich", specPath)
	}
}

func TestQuery_ConjunctiveScoringRank(t *testing.T) {
	tempWs := t.TempDir()

	// Módulo 1: contiene sólo una keyword ("usuario")
	mod1Dir := filepath.Join(tempWs, "internal", "auth")
	_ = os.MkdirAll(mod1Dir, 0755)
	_ = os.WriteFile(filepath.Join(mod1Dir, "token.go"), []byte("package auth\n// token de usuario\n"), 0644)

	// Módulo 2: contiene múltiples keywords conjuntas ("alta", "usuario", "campos")
	mod2Dir := filepath.Join(tempWs, "internal", "registration")
	_ = os.MkdirAll(mod2Dir, 0755)
	_ = os.WriteFile(filepath.Join(mod2Dir, "register.go"), []byte("package registration\n// campos requeridos para dar alta de usuario\n"), 0644)

	res, err := RunQuery(context.Background(), QueryOptions{
		WorkspaceRoot: tempWs,
		Question:      "campos alta usuario",
	})
	if err != nil {
		t.Fatalf("RunQuery falló: %v", err)
	}

	if len(res.Evidences) == 0 {
		t.Fatalf("se esperaban evidencias")
	}

	// La evidencia con mayor score debe ser la de registration (múltiples keywords conjuntivas)
	if !strings.Contains(res.Evidences[0].File, "registration") {
		t.Errorf("el primer resultado debe ser el de mayor ranking (registration), obtenido: %s", res.Evidences[0].File)
	}
}
