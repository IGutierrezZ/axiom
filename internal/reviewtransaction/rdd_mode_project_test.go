package reviewtransaction

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// El modo RDD de proyecto (D2) decide por encima del global y por debajo del off del clon:
// el valor del proyecto lo escribe el usuario y no se versiona, así que puede forzar `on`.
func TestResolveRDDModeProjectPreferenceTakesPrecedenceOverGlobal(t *testing.T) {
	for _, test := range []struct {
		name       string
		project    string
		global     string
		cloneOff   bool
		effective  RDDMode
		source     RDDModeSource
		projectOut RDDMode
	}{
		{name: "project on over global off", project: "on", global: "off", effective: RDDModeOn, source: RDDModeSourceProject, projectOut: RDDModeOn},
		{name: "project on over global unset", project: "on", effective: RDDModeOn, source: RDDModeSourceProject, projectOut: RDDModeOn},
		{name: "project off over global on", project: "off", global: "on", effective: RDDModeOff, source: RDDModeSourceProject, projectOut: RDDModeOff},
		{name: "project unset falls to global on", global: "on", effective: RDDModeOn, source: RDDModeSourceGlobal},
		{name: "project unset falls to global off", global: "off", effective: RDDModeOff, source: RDDModeSourceGlobal},
		{name: "nothing set defaults to off", effective: RDDModeOff, source: RDDModeSourceDefault},
		{name: "clone off wins over project on", project: "on", global: "on", cloneOff: true, effective: RDDModeOff, source: RDDModeSourceCloneLocal, projectOut: RDDModeOn},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := initSnapshotRepo(t)
			inputs := RDDGlobalMode{Value: test.global, Project: test.project}
			if test.cloneOff {
				if _, err := SetCloneLocalRDDMode(context.Background(), repo, RDDModeOff, "", inputs); err != nil {
					t.Fatalf("SetCloneLocalRDDMode(off) error = %v", err)
				}
			}
			status, err := ResolveRDDMode(context.Background(), repo, inputs)
			if err != nil {
				t.Fatalf("ResolveRDDMode error = %v", err)
			}
			if status.Effective != test.effective || status.Source != test.source || status.Project != test.projectOut {
				t.Fatalf("status = %#v, want effective=%s source=%s project=%q", status, test.effective, test.source, test.projectOut)
			}
		})
	}
}

func TestResolveRDDModeFailsClosedOnAnUnknownProjectValue(t *testing.T) {
	repo := initSnapshotRepo(t)
	status, err := ResolveRDDMode(context.Background(), repo, RDDGlobalMode{Value: "on", Project: "sometimes"})
	if !errors.Is(err, ErrRDDModeUnknown) {
		t.Fatalf("error = %v, want ErrRDDModeUnknown", err)
	}
	if status.Enabled() || status.Source != RDDModeSourceProject {
		t.Fatalf("an unknown project value must project a disabled status from the project source: %#v", status)
	}
}

func TestSetCloneLocalRDDModeEnableRefusesWhileTheProjectPreferenceIsOff(t *testing.T) {
	repo := initSnapshotRepo(t)
	_, err := SetCloneLocalRDDMode(context.Background(), repo, RDDModeUnset, "", RDDGlobalMode{Value: "on", Project: "off"})
	var disabled *RDDDisabledError
	if !errors.As(err, &disabled) || disabled.Source != RDDModeSourceProject {
		t.Fatalf("error = %v, want RDDDisabledError from the project source", err)
	}
	if !strings.Contains(err.Error(), "axiom review mode enable") {
		t.Fatalf("the refusal must name the command that turns reviews on: %v", err)
	}
}
