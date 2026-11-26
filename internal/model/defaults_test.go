package model

import (
	"reflect"
	"testing"
)

func TestDefaultComponentsAreTheCoreSet(t *testing.T) {
	want := []ComponentID{
		ComponentEngram,
		ComponentSDD,
		ComponentSkills,
		ComponentContext7,
		ComponentPermission,
		ComponentPersona,
	}
	if got := DefaultComponents(); !reflect.DeepEqual(got, want) {
		t.Fatalf("DefaultComponents() = %v, want %v", got, want)
	}
}

func TestDefaultComponentsDoNotImplyCommunityTools(t *testing.T) {
	selection := Selection{Components: DefaultComponents()}
	if len(selection.CommunityTools) != 0 {
		t.Fatalf("community tools = %v, want none", selection.CommunityTools)
	}
}

func TestDefaultComponentsReturnsAFreshSlice(t *testing.T) {
	first := DefaultComponents()
	first[0] = ComponentContext7
	if got := DefaultComponents()[0]; got != ComponentEngram {
		t.Fatalf("DefaultComponents()[0] = %q after mutating a previous result, want %q", got, ComponentEngram)
	}
}
