package branch

import "testing"

func TestResolveChangeBranchComposesPrefixAndChange(t *testing.T) {
	if got := ResolveChangeBranch("mi-cambio", "inc"); got != "inc/mi-cambio" {
		t.Fatalf("ResolveChangeBranch sin fuentes = %q, se esperaba inc/mi-cambio", got)
	}
}

func TestResolveChangeBranchSourceOrder(t *testing.T) {
	cases := []struct {
		name    string
		sources []ChangeBranchSource
		want    string
	}{
		{"la primera con valor gana", []ChangeBranchSource{NamedBranch("--branch", "a/uno"), NamedBranch("snapshot", "b/dos")}, "a/uno"},
		{"salta las vacías", []ChangeBranchSource{NamedBranch("--branch", ""), NamedBranch("snapshot", "b/dos"), NamedBranch("registro", "c/tres")}, "b/dos"},
		{"el orden lo fija quien llama", []ChangeBranchSource{NamedBranch("registro", "c/tres"), NamedBranch("snapshot", "b/dos")}, "c/tres"},
		{"todas vacías componen el prefijo", []ChangeBranchSource{NamedBranch("--branch", ""), NamedBranch("snapshot", "")}, "inc/x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveChangeBranch("x", "inc", tc.sources...); got != tc.want {
				t.Fatalf("= %q, se esperaba %q", got, tc.want)
			}
		})
	}
}

func TestResolveChangeBranchWithoutPrefixIsEmpty(t *testing.T) {
	if got := ResolveChangeBranch("x", ""); got != "" {
		t.Fatalf("sin fuentes ni prefijo se esperaba vacío, salió %q", got)
	}
	if got := ResolveChangeBranch("x", "", NamedBranch("registro", "r/x")); got != "r/x" {
		t.Fatalf("una fuente con valor debe ganar, salió %q", got)
	}
}

func TestArchiveBranchFor(t *testing.T) {
	if got := ArchiveBranchFor("inc/mi-cambio"); got != "inc/mi-cambio-archive" {
		t.Fatalf("ArchiveBranchFor = %q", got)
	}
}
