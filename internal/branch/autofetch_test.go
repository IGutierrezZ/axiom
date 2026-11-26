package branch

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// El resultado de partida es apagado, sin red intentada y con `notices` como `[]` en el JSON.
func TestAutofetchResultStartsDisabledWithAnEmptyNoticeList(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(NewAutofetchResult())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"schema":"axiom.autofetch-result/v1"`, `"operation":"autofetch run"`, `"autofetch":"disabled"`, `"fetch":"not_attempted"`, `"notices":[]`} {
		if !strings.Contains(string(payload), want) {
			t.Errorf("JSON %s: falta %s", payload, want)
		}
	}
}

func TestAutofetchResultWriteTextListsFetchAndNoticesWithTheirCommand(t *testing.T) {
	t.Parallel()
	result := NewAutofetchResult()
	result.Autofetch, result.Fetch = AutofetchStateEnabled, AutofetchFetchSkippedInterval
	result.Notices = append(result.Notices, AutofetchNotice{Kind: AutofetchNoticeFetchFailed, Branch: "origin", Message: "no pudo traer origin", Command: "git fetch origin"})
	var out bytes.Buffer

	result.WriteText(&out)

	for _, want := range []string{"Autofetch: encendido", "omitido", "Aviso: no pudo traer origin", "git fetch origin"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("texto %q: falta %q", out.String(), want)
		}
	}
}
