package workspace

import (
	"slices"
	"testing"
)

// RF-TOP-21: cada accesor expone el vocabulario cerrado del esquema y devuelve una
// copia nueva por llamada, de modo que modificarla no altera el validador.
func TestPromotionValuesReturnsAFreshCopy(t *testing.T) {
	got := PromotionValues()
	if want := []string{"branch", "release", "ask"}; !slices.Equal(got, want) {
		t.Fatalf("PromotionValues() = %v, quiero %v", got, want)
	}
	got[0] = "zzz"
	if PromotionValues()[0] != PromotionBranch {
		t.Fatal("PromotionValues debe devolver una copia nueva por llamada")
	}
}

func TestWorkflowLanesReturnsAFreshCopy(t *testing.T) {
	got := WorkflowLanes()
	if want := []WorkflowLane{"odd", "sdd", "ask"}; !slices.Equal(got, want) {
		t.Fatalf("WorkflowLanes() = %v, quiero %v", got, want)
	}
	got[0] = "zzz"
	if WorkflowLanes()[0] != WorkflowLaneODD {
		t.Fatal("WorkflowLanes debe devolver una copia nueva por llamada")
	}
}

func TestExplainerHTMLModesReturnsAFreshCopy(t *testing.T) {
	got := ExplainerHTMLModes()
	if want := []ExplainerHTMLMode{"ask", "always", "never"}; !slices.Equal(got, want) {
		t.Fatalf("ExplainerHTMLModes() = %v, quiero %v", got, want)
	}
	got[0] = "zzz"
	if ExplainerHTMLModes()[0] != ExplainerHTMLAsk {
		t.Fatal("ExplainerHTMLModes debe devolver una copia nueva por llamada")
	}
}
