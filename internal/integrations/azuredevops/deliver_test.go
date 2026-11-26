package azuredevops_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

// identityFake es una IdentityLookup de una sola entrada.
type identityFake struct {
	identity integrations.TrackerIdentity
	ok       bool
	err      error
}

func (f identityFake) Lookup(plugin, account string) (integrations.TrackerIdentity, bool, error) {
	if plugin != manifest.AzureDevOpsID || account != "contoso" {
		return integrations.TrackerIdentity{}, false, nil
	}
	return f.identity, f.ok, f.err
}

func (e *taskEnv) delivery(ev integrations.Event, cfg map[string]any, links ...integrations.Link) integrations.Delivery {
	config := map[string]any{}
	for k, v := range e.port.Config {
		config[k] = v
	}
	for k, v := range cfg {
		config[k] = v
	}
	return integrations.Delivery{Event: ev, Config: config, Links: links, Secrets: e.port.Secrets}
}

func roleEvent(kind manifest.EventType, actor string) integrations.Event {
	return integrations.Event{Type: kind, Change: "cambio", Role: "fullstack", Actor: actor, OccurredAt: time.Now().UTC(), IdempotencyKey: string(kind) + "/cambio/fullstack"}
}

func taskLink() integrations.Link {
	return integrations.Link{Plugin: manifest.AzureDevOpsID, Kind: integrations.TaskLinkKind, ID: itoa(taskID), Role: "fullstack", Source: integrations.TasksBindSource}
}

func (e *taskEnv) deliver(d integrations.Delivery) (integrations.Outcome, error) {
	return e.p.Deliver(context.Background(), d)
}

func TestDeliverRoleStartMovesTheTaskAndAssignsTheActorWhoDelivers(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("To Do", nil)
	seedMembers(e, adotest.Member{ID: "a1", DisplayName: "Ana Gil"})
	d := e.delivery(roleEvent(manifest.EventRoleStart, "Ana Gil"), nil, taskLink())
	d.Deliverer = "Ana Gil"
	d.Identities = identityFake{identity: *ana(), ok: true}

	out, err := e.deliver(d)

	if err != nil || len(out.Warnings) != 0 || len(out.Links) != 0 || len(out.TimeEntries) != 0 {
		t.Fatalf("resultado = %+v, %v", out, err)
	}
	if got := e.read(t, taskID); got.State != "Active" || got.AssignedToID != "a1" {
		t.Fatalf("Task = %+v", got)
	}
}

func TestDeliverRoleStartUsesTheConfiguredState(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("To Do", nil)
	d := e.delivery(roleEvent(manifest.EventRoleStart, "cli"), map[string]any{"task_state_start": "Tech Review"}, taskLink())

	if _, err := e.deliver(d); err != nil {
		t.Fatal(err)
	}
	if got := e.read(t, taskID).State; got != "Tech Review" {
		t.Fatalf("estado = %q", got)
	}
}

func TestDeliverRoleStartSkipsAssignmentWithAWarningNamingConnect(t *testing.T) {
	cases := map[string]func(*integrations.Delivery){
		"actor cli":        func(d *integrations.Delivery) { d.Event.Actor = "cli" },
		"actor distinto":   func(d *integrations.Delivery) { d.Deliverer = "Berto Ruiz" },
		"sin consulta":     func(d *integrations.Delivery) { d.Identities = nil },
		"sin identidad":    func(d *integrations.Delivery) { d.Identities = identityFake{} },
		"consulta fallida": func(d *integrations.Delivery) { d.Identities = identityFake{err: errors.New("almacén ilegible")} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			e := newTaskEnv(t)
			e.seedLifecycleTask("To Do", nil)
			seedMembers(e, adotest.Member{ID: "a1", DisplayName: "Ana Gil"})
			d := e.delivery(roleEvent(manifest.EventRoleStart, "Ana Gil"), nil, taskLink())
			d.Deliverer = "Ana Gil"
			d.Identities = identityFake{identity: *ana(), ok: true}
			mutate(&d)

			out, err := e.deliver(d)

			if err != nil || len(out.Warnings) != 1 ||
				!strings.HasPrefix(out.Warnings[0], "assignment_skipped: ") ||
				!strings.Contains(out.Warnings[0], "axiom integrations connect --plugin azure-devops") {
				t.Fatalf("resultado = %+v, %v", out, err)
			}
			if got := e.read(t, taskID); got.State != "Active" || got.AssignedToID != "" {
				t.Fatalf("el estado se aplica sin asignar: %+v", got)
			}
		})
	}
}

func TestDeliverCopiesTransitionWarningsWithCodeAndMessage(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("Tech Review", nil)
	d := e.delivery(roleEvent(manifest.EventRoleStart, "cli"), nil, taskLink())

	out, err := e.deliver(d)

	if err != nil {
		t.Fatal(err)
	}
	var kept string
	for _, w := range out.Warnings {
		if strings.HasPrefix(w, "task_state_kept: ") {
			kept = w
		}
	}
	if kept == "" {
		t.Fatalf("avisos = %v", out.Warnings)
	}
}

func TestDeliverRoleApplyApprovedMovesToTechReview(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("Active", nil)
	d := e.delivery(roleEvent(manifest.EventRoleApplyApproved, "Ana Gil"), nil, taskLink())

	out, err := e.deliver(d)

	if err != nil || len(out.Warnings) != 0 {
		t.Fatalf("resultado = %+v, %v", out, err)
	}
	if got := e.read(t, taskID).State; got != "Tech Review" {
		t.Fatalf("estado = %q", got)
	}
}

func TestDeliverUnsupportedEventsArePermanentAndTouchNoNetwork(t *testing.T) {
	for _, kind := range []manifest.EventType{manifest.EventChangeStart, manifest.EventGateSpecApproved, manifest.EventGateDesignApproved} {
		t.Run(string(kind), func(t *testing.T) {
			e := newTaskEnv(t)
			e.seedLifecycleTask("To Do", nil)

			_, err := e.deliver(e.delivery(roleEvent(kind, "cli"), nil, taskLink()))

			if !integrations.IsPermanent(err) || !strings.Contains(err.Error(), "role.start") {
				t.Fatalf("error = %v", err)
			}
			if n := len(e.srv.Requests()); n != 0 {
				t.Fatalf("peticiones = %d, esperaba ninguna", n)
			}
		})
	}
}

func TestDeliverRoleEventWithoutTaskLinkIsPermanentNamingTheWayOut(t *testing.T) {
	for _, kind := range []manifest.EventType{manifest.EventRoleStart, manifest.EventRoleApplyApproved} {
		e := newTaskEnv(t)
		other := taskLink()
		other.Role = "backend"

		_, err := e.deliver(e.delivery(roleEvent(kind, "cli"), nil, other))

		if !integrations.IsPermanent(err) || !errors.Is(err, integrations.ErrTaskNotBound) {
			t.Fatalf("%s: error = %v", kind, err)
		}
		for _, want := range []string{"git pull", "axiom integrations tasks create|bind", "axiom integrations sync --retry-abandoned"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s: el mensaje debe nombrar %q: %v", kind, want, err)
			}
		}
		if len(e.srv.Requests()) != 0 {
			t.Fatal("sin vínculo no hay red")
		}
	}
}

func TestDeliverRoleEventPermanenceFollowsTheFailure(t *testing.T) {
	t.Run("estado inexistente es permanente", func(t *testing.T) {
		e := newTaskEnv(t)
		e.seedLifecycleTask("To Do", nil)
		_, err := e.deliver(e.delivery(roleEvent(manifest.EventRoleStart, "cli"), map[string]any{"task_state_start": "Nada"}, taskLink()))
		if !integrations.IsPermanent(err) || !errors.Is(err, integrations.ErrStateUnknown) {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("regla del proceso es permanente", func(t *testing.T) {
		e := newTaskEnv(t)
		e.seedLifecycleTask("To Do", nil)
		e.srv.RejectStateChange("TF401320: regla")
		_, err := e.deliver(e.delivery(roleEvent(manifest.EventRoleStart, "cli"), nil, taskLink()))
		if !integrations.IsPermanent(err) || !errors.Is(err, integrations.ErrTrackerRejected) {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("Task inexistente es permanente", func(t *testing.T) {
		e := newTaskEnv(t)
		_, err := e.deliver(e.delivery(roleEvent(manifest.EventRoleStart, "cli"), nil, taskLink()))
		if !integrations.IsPermanent(err) {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("conflicto de rev es transitorio", func(t *testing.T) {
		e := newTaskEnv(t)
		e.seedLifecycleTask("To Do", nil)
		e.srv.EditAfterNextRead(taskID, map[string]any{"System.Title": "otra edición"})
		_, err := e.deliver(e.delivery(roleEvent(manifest.EventRoleStart, "cli"), nil, taskLink()))
		if integrations.IsPermanent(err) || !errors.Is(err, integrations.ErrRevConflict) {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("autenticación es transitoria", func(t *testing.T) {
		e := newTaskEnv(t)
		e.srv.Reply("GET", "/contoso/Fabrikam/_apis/wit/workitems/77", adotest.Response{Status: 401, Body: "no"})
		_, err := e.deliver(e.delivery(roleEvent(manifest.EventRoleStart, "cli"), nil, taskLink()))
		if err == nil || integrations.IsPermanent(err) {
			t.Fatalf("error = %v", err)
		}
	})
}

// ---- archive (D19)

const archivedItem = 55

func archiveEvent() integrations.Event {
	return integrations.Event{Type: manifest.EventArchive, Change: "cambio", Actor: "cli", OccurredAt: time.Now().UTC(), IdempotencyKey: "archive/cambio/2026-11-26-cambio"}
}

func workItemLink(id int) integrations.Link {
	return integrations.Link{Plugin: manifest.AzureDevOpsID, Kind: "work_item", ID: itoa(id), Source: "change.start"}
}

func (e *taskEnv) seedArchiveItem(itemType, state string) {
	e.srv.SeedWorkItem(archivedItem, 1, map[string]any{"System.WorkItemType": itemType, "System.State": state})
}

func seedProcessStates(e *taskEnv) {
	e.srv.SeedStates("User Story",
		adotest.State{Name: "New", Category: "Proposed"},
		adotest.State{Name: "Active", Category: "InProgress"},
		adotest.State{Name: "Resolved", Category: "Resolved"},
		adotest.State{Name: "Closed", Category: "Completed"},
		adotest.State{Name: "Removed", Category: "Removed"})
	e.srv.SeedStates("Bug",
		adotest.State{Name: "New", Category: "Proposed"},
		adotest.State{Name: "Active", Category: "InProgress"},
		adotest.State{Name: "Fixed", Category: "Resolved"},
		adotest.State{Name: "Done", Category: "Completed"})
}

func archiveState(t *testing.T, e *taskEnv) string {
	t.Helper()
	wi, _ := e.srv.WorkItem(archivedItem)
	s, _ := wi.Fields["System.State"].(string)
	return s
}

func TestDeliverArchiveWithoutConfiguredStateDoesNotTouchTheNetwork(t *testing.T) {
	e := newTaskEnv(t)

	out, err := e.deliver(e.delivery(archiveEvent(), nil, workItemLink(archivedItem)))

	if err != nil || len(out.Warnings) != 0 || len(e.srv.Requests()) != 0 {
		t.Fatalf("resultado = %+v, %v, peticiones = %d", out, err, len(e.srv.Requests()))
	}
}

func TestDeliverArchiveWithoutWorkItemLinkSucceedsWithAWarningAndNoNetwork(t *testing.T) {
	e := newTaskEnv(t)

	out, err := e.deliver(e.delivery(archiveEvent(), map[string]any{"work_item_state_archive": "Closed"}, taskLink()))

	if err != nil || len(out.Warnings) != 1 || !strings.HasPrefix(out.Warnings[0], "work_item_not_linked: ") || len(e.srv.Requests()) != 0 {
		t.Fatalf("resultado = %+v, %v, peticiones = %d", out, err, len(e.srv.Requests()))
	}
}

func TestDeliverArchiveUsesTheStatesOfTheItemOwnType(t *testing.T) {
	cfg := map[string]any{"work_item_state_archive": "Closed"}

	t.Run("User Story cierra a Closed", func(t *testing.T) {
		e := newTaskEnv(t)
		seedProcessStates(e)
		e.seedArchiveItem("User Story", "Active")
		out, err := e.deliver(e.delivery(archiveEvent(), cfg, workItemLink(archivedItem)))
		if err != nil || len(out.Warnings) != 0 || len(out.Links) != 0 || len(out.TimeEntries) != 0 {
			t.Fatalf("resultado = %+v, %v", out, err)
		}
		if got := archiveState(t, e); got != "Closed" {
			t.Fatalf("estado = %q", got)
		}
	})
	t.Run("Bug no tiene Closed: error permanente con sus estados", func(t *testing.T) {
		e := newTaskEnv(t)
		seedProcessStates(e)
		e.seedArchiveItem("Bug", "Active")
		_, err := e.deliver(e.delivery(archiveEvent(), cfg, workItemLink(archivedItem)))
		var unknown *integrations.StateUnknownError
		if !integrations.IsPermanent(err) || !errors.As(err, &unknown) || strings.Join(unknown.Valid, ",") != "New,Active,Fixed,Done" {
			t.Fatalf("error = %v", err)
		}
		if got := archiveState(t, e); got != "Active" {
			t.Fatalf("sin escritura, estado = %q", got)
		}
	})
	t.Run("Bug cierra a Done si es el destino configurado", func(t *testing.T) {
		e := newTaskEnv(t)
		seedProcessStates(e)
		e.seedArchiveItem("Bug", "Fixed")
		if _, err := e.deliver(e.delivery(archiveEvent(), map[string]any{"work_item_state_archive": "Done"}, workItemLink(archivedItem))); err != nil {
			t.Fatal(err)
		}
		if got := archiveState(t, e); got != "Done" {
			t.Fatalf("estado = %q", got)
		}
	})
}

func TestDeliverArchiveAlreadyInTargetDoesNotWrite(t *testing.T) {
	e := newTaskEnv(t)
	seedProcessStates(e)
	e.seedArchiveItem("User Story", "Closed")

	out, err := e.deliver(e.delivery(archiveEvent(), map[string]any{"work_item_state_archive": "Closed"}, workItemLink(archivedItem)))

	if err != nil || len(out.Warnings) != 0 || e.srv.AppliedPatches(archivedItem) != 0 {
		t.Fatalf("resultado = %+v, %v, PATCH = %d", out, err, e.srv.AppliedPatches(archivedItem))
	}
}

func TestDeliverArchiveClosedItemInAnotherStateIsKeptWithAWarning(t *testing.T) {
	for _, state := range []string{"Removed", "Closed"} {
		e := newTaskEnv(t)
		seedProcessStates(e)
		e.seedArchiveItem("User Story", state)
		target := "Resolved"
		if state == "Closed" {
			target = "Active"
		}

		out, err := e.deliver(e.delivery(archiveEvent(), map[string]any{"work_item_state_archive": target}, workItemLink(archivedItem)))

		if err != nil || len(out.Warnings) != 1 || !strings.HasPrefix(out.Warnings[0], "work_item_state_kept: ") || e.srv.AppliedPatches(archivedItem) != 0 {
			t.Fatalf("%s: resultado = %+v, %v", state, out, err)
		}
	}
}

func TestDeliverArchiveRevConflictIsTransientAndRulesRejectionIsPermanent(t *testing.T) {
	cfg := map[string]any{"work_item_state_archive": "Closed"}

	t.Run("conflicto de rev", func(t *testing.T) {
		e := newTaskEnv(t)
		seedProcessStates(e)
		e.seedArchiveItem("User Story", "Active")
		e.srv.EditAfterNextRead(archivedItem, map[string]any{"System.Title": "otra edición"})
		_, err := e.deliver(e.delivery(archiveEvent(), cfg, workItemLink(archivedItem)))
		if err == nil || integrations.IsPermanent(err) || !errors.Is(err, integrations.ErrRevConflict) {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("400 de reglas", func(t *testing.T) {
		e := newTaskEnv(t)
		seedProcessStates(e)
		e.seedArchiveItem("User Story", "Active")
		e.srv.RejectStateChange("TF401320: regla")
		_, err := e.deliver(e.delivery(archiveEvent(), cfg, workItemLink(archivedItem)))
		if !integrations.IsPermanent(err) || !errors.Is(err, integrations.ErrTrackerRejected) {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("work item inexistente es permanente", func(t *testing.T) {
		e := newTaskEnv(t)
		seedProcessStates(e)
		_, err := e.deliver(e.delivery(archiveEvent(), cfg, workItemLink(archivedItem)))
		if !integrations.IsPermanent(err) {
			t.Fatalf("error = %v", err)
		}
	})
}
