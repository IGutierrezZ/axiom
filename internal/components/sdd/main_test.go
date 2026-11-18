package sdd

import (
	"os"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/rddharness"
)

// TestMain activates the RDD injection point so the tests that exercise the
// dormant review lifecycle rendering keep covering that code. The tests that
// check the suspension (C-01-c) switch it off locally with
// rddharness.SetForTesting(false).
func TestMain(m *testing.M) {
	rddharness.SetForTesting(true)
	os.Exit(m.Run())
}
