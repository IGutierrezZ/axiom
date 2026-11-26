package nametemplate

import (
	"slices"
	"testing"
)

func TestLiteralPrefixes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		raw, prefix string
		want        []string
	}{
		{"feature/{change}", "inc", []string{"feature"}},
		{"{prefix}/{change}", "inc", []string{"inc"}},
		{"{prefix}/{change}", "", nil},
		{"{kind}/{slug}", "inc", []string{"feature", "fix"}},
		{"a/b/TASK-{change}", "inc", []string{"a/b"}},
		{"{change}-x", "inc", nil},
		{"{ref}/{change}", "inc", nil},
		{"feat-{change}", "inc", nil},
		{"{prefix}-{kind}/{change}", "inc", []string{"inc-feature", "inc-fix"}},
	}
	for _, c := range cases {
		parsed, err := Parse(c.raw)
		if err != nil {
			t.Fatalf("%s: %v", c.raw, err)
		}
		if got := LiteralPrefixes(parsed, c.prefix); !slices.Equal(got, c.want) {
			t.Errorf("%s (prefijo %q): %v, se esperaba %v", c.raw, c.prefix, got, c.want)
		}
	}
}
