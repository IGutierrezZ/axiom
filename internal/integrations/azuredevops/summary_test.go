package azuredevops_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

const summaryItem = 90

func (e *taskEnv) seedSummaryItem(tags, description string) {
	e.srv.SeedWorkItem(summaryItem, 1, map[string]any{
		"System.WorkItemType": "User Story",
		"System.Tags":         tags,
		"System.Description":  description,
	})
}

func (e *taskEnv) summary(t *testing.T, html, prefix string, max int, tags ...string) (integrations.SummaryResult, error) {
	t.Helper()
	return e.p.UpdateSummary(context.Background(), integrations.SummaryRequest{
		PortRequest: e.port, ItemID: itoa(summaryItem), HTML: html, TagPrefix: prefix, Tags: tags, TagMax: max,
	})
}

func (e *taskEnv) summaryFields(t *testing.T) (description string, tags []string) {
	t.Helper()
	wi, ok := e.srv.WorkItem(summaryItem)
	if !ok {
		t.Fatal("falta el work item")
	}
	raw, _ := wi.Fields["System.Tags"].(string)
	for _, tag := range strings.Split(raw, ";") {
		if tag = strings.TrimSpace(tag); tag != "" {
			tags = append(tags, tag)
		}
	}
	description, _ = wi.Fields["System.Description"].(string)
	return description, tags
}

func TestUpdateSummaryWithoutDifferencesDoesNotWrite(t *testing.T) {
	e := newTaskEnv(t)
	e.seedSummaryItem("req:RF-A-1; req:RF-A-2; equipo", "<h2>x</h2>")

	res, err := e.summary(t, "<h2>x</h2>", "req:", 50, "req:RF-A-2", "req:RF-A-1")

	if err != nil || res.Wrote || res.DescriptionChanged || res.TagsSent {
		t.Fatalf("resultado = %+v, %v", res, err)
	}
	if e.srv.AppliedPatches(summaryItem) != 0 {
		t.Fatal("sin diferencias no hay PATCH")
	}
}

func TestUpdateSummaryWritesOnceKeepsForeignTagsAndReportsTheDiff(t *testing.T) {
	e := newTaskEnv(t)
	e.seedSummaryItem("equipo; axiom:cambio; prov:ADO:1; req:RF-A-9", "viejo")

	res, err := e.summary(t, "<h2>nuevo</h2>", "req:", 50, "req:RF-A-1", "req:RF-A-2")

	if err != nil || !res.Wrote || !res.DescriptionChanged || !res.TagsSent {
		t.Fatalf("resultado = %+v, %v", res, err)
	}
	if e.srv.AppliedPatches(summaryItem) != 1 {
		t.Fatalf("una sola escritura, hubo %d", e.srv.AppliedPatches(summaryItem))
	}
	description, tags := e.summaryFields(t)
	if description != "<h2>nuevo</h2>" {
		t.Fatalf("descripción = %q", description)
	}
	want := "equipo,axiom:cambio,prov:ADO:1,req:RF-A-1,req:RF-A-2"
	if strings.Join(tags, ",") != want {
		t.Fatalf("etiquetas = %v, esperaba %s", tags, want)
	}
	if strings.Join(res.Added, ",") != "req:RF-A-1,req:RF-A-2" || strings.Join(res.Removed, ",") != "req:RF-A-9" {
		t.Fatalf("Added = %v, Removed = %v", res.Added, res.Removed)
	}
}

func TestUpdateSummaryOverTheCapRemovesEveryOwnTagAndKeepsTheDescription(t *testing.T) {
	e := newTaskEnv(t)
	e.seedSummaryItem("equipo; req:RF-A-1; req:prov:ADO:1", "viejo")

	res, err := e.summary(t, "<h2>nuevo</h2>", "req:", 2, "req:RF-A-1", "req:RF-A-2", "req:RF-A-3")

	if err != nil || !res.Wrote || !hasCode(res.Warnings, "req_tags_over_limit") {
		t.Fatalf("resultado = %+v, %v", res, err)
	}
	description, tags := e.summaryFields(t)
	if description != "<h2>nuevo</h2>" || strings.Join(tags, ",") != "equipo" {
		t.Fatalf("descripción = %q, etiquetas = %v", description, tags)
	}
}

func TestUpdateSummaryTotalOverOneHundredIsOverTheLimit(t *testing.T) {
	e := newTaskEnv(t)
	foreign := make([]string, 0, 60)
	for i := 0; i < 60; i++ {
		foreign = append(foreign, fmt.Sprintf("ajena-%d", i))
	}
	e.seedSummaryItem(strings.Join(foreign, "; "), "")
	desired := make([]string, 0, 50)
	for i := 0; i < 50; i++ {
		desired = append(desired, fmt.Sprintf("req:RF-A-%d", i))
	}

	res, err := e.summary(t, "x", "req:", 50, desired...)

	if err != nil || !hasCode(res.Warnings, "req_tags_over_limit") || res.TagsSent {
		t.Fatalf("resultado = %+v, %v", res, err)
	}
	if _, tags := e.summaryFields(t); len(tags) != 60 {
		t.Fatalf("quedan %d etiquetas, esperaba las 60 ajenas", len(tags))
	}
}

func TestUpdateSummaryInvalidTagWarnsAndWritesNoRequirementTag(t *testing.T) {
	e := newTaskEnv(t)
	e.seedSummaryItem("req:RF-A-1", "")

	res, err := e.summary(t, "x", "req:", 50, "req:RF-A-1", "req:con;punto")

	if err != nil || !hasCode(res.Warnings, "req_tag_invalid") {
		t.Fatalf("resultado = %+v, %v", res, err)
	}
	if _, tags := e.summaryFields(t); len(tags) != 0 {
		t.Fatalf("etiquetas = %v, esperaba ninguna", tags)
	}
}

func TestUpdateSummaryUsesTheConfiguredPrefixWithCaseSensitivity(t *testing.T) {
	e := newTaskEnv(t)
	e.seedSummaryItem("Req:ajena; spec/RF-A-0", "")

	res, err := e.summary(t, "x", "spec/", 50, "spec/RF-A-1")

	if err != nil || !res.Wrote {
		t.Fatalf("resultado = %+v, %v", res, err)
	}
	_, tags := e.summaryFields(t)
	if strings.Join(tags, ",") != "Req:ajena,spec/RF-A-1" {
		t.Fatalf("etiquetas = %v: «Req:» no es del prefijo spec/ y se conserva", tags)
	}
}

func TestUpdateSummaryRetriesRevConflictsUpToThreeAttempts(t *testing.T) {
	edit := map[string]any{"System.Title": "editado por otra persona"}

	t.Run("se recupera al tercer intento", func(t *testing.T) {
		e := newTaskEnv(t)
		e.seedSummaryItem("", "")
		e.srv.EditAfterNextRead(summaryItem, edit)
		e.srv.EditAfterNextRead(summaryItem, edit)

		res, err := e.summary(t, "x", "req:", 50, "req:RF-A-1")

		if err != nil || !res.Wrote {
			t.Fatalf("resultado = %+v, %v", res, err)
		}
		if got := e.srv.Count("GET", "/contoso/Fabrikam/_apis/wit/workitems/90"); got != 3 {
			t.Fatalf("lecturas = %d, esperaba 3", got)
		}
	})

	t.Run("agota los tres intentos", func(t *testing.T) {
		e := newTaskEnv(t)
		e.seedSummaryItem("", "")
		for i := 0; i < 3; i++ {
			e.srv.EditAfterNextRead(summaryItem, edit)
		}

		_, err := e.summary(t, "x", "req:", 50, "req:RF-A-1")

		if !errors.Is(err, integrations.ErrRevConflict) {
			t.Fatalf("error = %v", err)
		}
		if e.srv.AppliedPatches(summaryItem) != 0 {
			t.Fatal("ningún PATCH se aplicó")
		}
	})
}

func TestUpdateSummaryMissingItemIsTaskNotFound(t *testing.T) {
	e := newTaskEnv(t)

	_, err := e.summary(t, "x", "req:", 50)

	if !errors.Is(err, integrations.ErrTaskNotFound) {
		t.Fatalf("error = %v", err)
	}
}
