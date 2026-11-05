package cli

import (
	"slices"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/pipeline"
	"github.com/IGutierrezZ/axiom/v3/internal/planner"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

func stepIndex(plan pipeline.StagePlan, id string) int {
	for i, step := range plan.Apply {
		if step.ID() == id {
			return i
		}
	}
	return -1
}

// Persisted state records components in selection order (persona last). Sync
// must apply them in the order install does, with persona before SDD/Engram.
func TestInstallOrderedComponentsMatchesInstallOrder(t *testing.T) {
	selection := model.Selection{
		Agents:     []model.AgentID{model.AgentClaudeCode},
		Components: []model.ComponentID{model.ComponentEngram, model.ComponentSDD, model.ComponentContext7, model.ComponentPersona},
	}
	resolved, err := planner.NewResolver(planner.MVPGraph()).Resolve(selection)
	if err != nil {
		t.Fatal(err)
	}
	got := installOrderedComponents(selection.Components)
	if !slices.Equal(got, resolved.OrderedComponents) {
		t.Fatalf("installOrderedComponents() = %v, want install order %v", got, resolved.OrderedComponents)
	}
	if slices.Index(got, model.ComponentPersona) > slices.Index(got, model.ComponentEngram) {
		t.Fatalf("persona must precede engram, got %v", got)
	}
}

func TestInstallOrderedComponentsKeepsUnknownComponentsAtEnd(t *testing.T) {
	unknown := model.ComponentID("not-in-graph")
	got := installOrderedComponents([]model.ComponentID{unknown, model.ComponentSDD, model.ComponentPersona})
	if len(got) != 3 || got[len(got)-1] != unknown {
		t.Fatalf("installOrderedComponents() = %v, want unknown component last", got)
	}
}

func TestSyncStagePlanAppliesComponentsInInstallOrder(t *testing.T) {
	home := t.TempDir()
	selection := model.Selection{
		Agents:     []model.AgentID{model.AgentClaudeCode},
		Components: []model.ComponentID{model.ComponentEngram, model.ComponentSDD, model.ComponentPersona},
	}
	rt, err := newSyncRuntime(home, selection)
	if err != nil {
		t.Fatal(err)
	}
	plan := rt.stagePlan()
	persona := stepIndex(plan, "sync:component:"+string(model.ComponentPersona))
	engram := stepIndex(plan, "sync:component:"+string(model.ComponentEngram))
	sdd := stepIndex(plan, "sync:component:"+string(model.ComponentSDD))
	if persona < 0 || engram < 0 || sdd < 0 || persona > engram || persona > sdd {
		t.Fatalf("sync order persona=%d engram=%d sdd=%d, want persona first", persona, engram, sdd)
	}
}

func TestSyncAndInstallRunCodeGraphAfterPersona(t *testing.T) {
	home := t.TempDir()
	selection := model.Selection{
		Agents:         []model.AgentID{model.AgentClaudeCode},
		Components:     []model.ComponentID{model.ComponentEngram, model.ComponentPersona},
		CommunityTools: []model.CommunityToolID{model.CommunityToolCodeGraph},
	}

	syncRT, err := newSyncRuntime(home, selection)
	if err != nil {
		t.Fatal(err)
	}
	syncPlan := syncRT.stagePlan()
	if p, c := stepIndex(syncPlan, "sync:component:"+string(model.ComponentPersona)), stepIndex(syncPlan, "sync:community-tool:codegraph-guidance"); p < 0 || c < 0 || c < p {
		t.Fatalf("sync persona=%d codegraph=%d, want codegraph after persona", p, c)
	}

	resolved, err := planner.NewResolver(planner.MVPGraph()).Resolve(selection)
	if err != nil {
		t.Fatal(err)
	}
	installRT, err := newInstallRuntime(home, ScopeGlobal, ChannelStable, selection, resolved, system.PlatformProfile{PackageManager: "brew"})
	if err != nil {
		t.Fatal(err)
	}
	installPlan := installRT.stagePlan()
	if p, c := stepIndex(installPlan, "component:"+string(model.ComponentPersona)), stepIndex(installPlan, "community-tool:"+string(model.CommunityToolCodeGraph)); p < 0 || c < 0 || c < p {
		t.Fatalf("install persona=%d codegraph=%d, want codegraph after persona", p, c)
	}
}
