package marks

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests"
)

// Códigos de rechazo de validación.
const (
	CodeCaseUnknown        = "marks_case_unknown"
	CodeEnvironmentInvalid = "marks_environment_invalid"
	CodeResultInvalid      = "marks_result_invalid"
	CodeNoteInvalid        = "marks_note_invalid"
	CodeNoteTooLong        = "marks_note_too_long"
	CodeCommitInvalid      = "marks_commit_invalid"
)

const continuationPick = "elige un caso, un entorno y un resultado de la tabla de pruebas"

// Input es lo que el usuario aporta al marcar.
type Input struct{ Case, Environment, Result, Commit, Note string }

// Rejection es un rechazo tipado de la validación, con su continuación.
type Rejection struct{ Code, Message, Continuation string }

// ValidateInput valida una marca contra la batería actual y los entornos declarados. Devuelve
// la entrada sin By, At ni PlanDigest (los pone quien escribe) o el primer rechazo, en el
// orden caso, entorno, resultado, nota y commit.
func ValidateInput(in Input, battery acceptancetests.Battery, environments []string) (Entry, *Rejection) {
	applicable, known := applicableEnvironments(battery, environments, in.Case)
	if !known {
		return Entry{}, &Rejection{CodeCaseUnknown, "el caso " + quote(in.Case) + " no existe en pruebas.md", continuationPick}
	}
	if !applicable[in.Environment] {
		return Entry{}, &Rejection{CodeEnvironmentInvalid,
			"el entorno " + quote(in.Environment) + " no está declarado o no aplica al caso " + in.Case, continuationPick}
	}
	result := Result(in.Result)
	if !result.Valid() {
		return Entry{}, &Rejection{CodeResultInvalid, "el resultado " + quote(in.Result) + " no es ok, ko, blocked ni n/a", continuationPick}
	}
	note := strings.TrimSpace(in.Note)
	if !utf8.ValidString(note) || hasControl(note) {
		return Entry{}, &Rejection{CodeNoteInvalid, "la nota contiene caracteres de control o UTF-8 inválido", "escribe la nota en una sola línea de texto"}
	}
	if utf8.RuneCountInString(note) > MaxNoteRunes {
		return Entry{}, &Rejection{CodeNoteTooLong, "la nota supera el máximo de 500 caracteres", "acorta la nota a 500 caracteres o menos"}
	}
	commit := strings.TrimSpace(in.Commit)
	if !validCommit(commit) {
		return Entry{}, &Rejection{CodeCommitInvalid,
			"el commit debe tener como máximo 100 caracteres de A-Z, a-z, 0-9 y ._/+:#@-", "corrige el commit (SHA, etiqueta o build#123) o déjalo vacío"}
	}
	return Entry{Case: in.Case, Environment: in.Environment, Result: result, Commit: commit, Note: note}, nil
}

// applicableEnvironments devuelve los entornos aplicables del caso (columna Entornos ∩
// declarados; la unión de sus filas si el ID se repite) y si el caso existe.
func applicableEnvironments(battery acceptancetests.Battery, environments []string, id string) (map[string]bool, bool) {
	declared := make(map[string]bool, len(environments))
	for _, env := range environments {
		declared[env] = true
	}
	out := map[string]bool{}
	known := false
	for _, c := range battery.Cases {
		if c.ID != id {
			continue
		}
		known = true
		for _, env := range c.Environments {
			if declared[env] {
				out[env] = true
			}
		}
	}
	return out, known
}

func hasControl(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func validCommit(s string) bool {
	if utf8.RuneCountInString(s) > MaxCommitRunes {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case strings.ContainsRune("._/+:#@-", r):
		default:
			return false
		}
	}
	return true
}

func quote(s string) string {
	if utf8.RuneCountInString(s) > 40 {
		s = string([]rune(s)[:40]) + "…"
	}
	return `"` + strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '?'
		}
		return r
	}, s) + `"`
}
