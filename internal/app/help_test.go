package app

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpContainsAllCommands(t *testing.T) {
	var buf bytes.Buffer
	printHelp(&buf, "v1.0.0-test")
	output := buf.String()

	commands := []string{"install", "uninstall", "sync", "sdd-status", "sdd-continue", "sdd-attempt", "update", "upgrade", "restore", "version"}
	for _, cmd := range commands {
		if !strings.Contains(output, cmd) {
			t.Errorf("help output missing command %q", cmd)
		}
	}
}

// TestHelpAdvertisesNoRetiredWorkCommands keeps help and dispatch aligned: the
// retired remote control-plane commands no longer dispatch, so help must not
// advertise any of them.
func TestHelpAdvertisesNoRetiredWorkCommands(t *testing.T) {
	var buf bytes.Buffer
	printHelp(&buf, "v1.0.0-test")
	output := buf.String()

	for _, cmd := range []string{
		"work-capabilities", "work-start", "work-route", "work-advance",
		"work-verification-decide", "work-reconcile", "work-status", "work-transition",
		"WORK CONTRACT COMPATIBILITY",
	} {
		if strings.Contains(output, cmd) {
			t.Errorf("help output still advertises retired command surface %q", cmd)
		}
	}
}

func TestHelpContainsVersion(t *testing.T) {
	var buf bytes.Buffer
	printHelp(&buf, "v1.2.3")
	if !strings.Contains(buf.String(), "v1.2.3") {
		t.Error("help output should contain the version string")
	}
}

func TestHelpCommandsHeadingIsAligned(t *testing.T) {
	var buf bytes.Buffer
	printHelp(&buf, "v1.2.3")
	if !strings.Contains(buf.String(), "\nCOMMANDS\n  install") {
		t.Fatalf("help output has inconsistent command indentation:\n%s", buf.String())
	}
}

func TestHelpDoesNotAdvertiseRetiredVerifyValidator(t *testing.T) {
	var output bytes.Buffer
	printHelp(&output, "test")
	if strings.Contains(output.String(), "sdd-verify-validate") {
		t.Fatal("help advertises retired report admission")
	}
}

func TestHelpAdvertisesNoRetiredReviewCommands(t *testing.T) {
	var buf bytes.Buffer
	printHelp(&buf, "v1.0.0-test")
	output := buf.String()
	for _, retired := range []string{"review start", "review mode", "review-start", "review-validate", "COMPATIBILITY COMMANDS"} {
		if strings.Contains(output, retired) {
			t.Errorf("help output still advertises retired review surface %q", retired)
		}
	}
}
