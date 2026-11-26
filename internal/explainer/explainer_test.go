package explainer

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const goodDigest = "sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func goodHTML() string {
	return `<html><head><meta name="axiom-source-digest" content="` + goodDigest + `"></head>` +
		`<body><p>Este resumen no sustituye al documento: lo que se aprueba es el md.</p>` +
		`<section data-axiom-ref="REQ-1">x</section></body></html>`
}

func codes(findings []Finding) []string {
	var out []string
	for _, f := range findings {
		out = append(out, f.Code)
	}
	return out
}

func TestKindsAndFileName(t *testing.T) {
	t.Parallel()
	if got := Kinds(); !reflect.DeepEqual(got, []Kind{KindSpec, KindDesign}) {
		t.Fatalf("Kinds() = %v", got)
	}
	if FileName(KindSpec) != "resumen-spec.html" || FileName(KindDesign) != "resumen-design.html" {
		t.Fatal("nombres de fichero inesperados")
	}
	if FileName("kickoff.yaml") != "" || ValidKind("spec.md") || !ValidKind(KindSpec) {
		t.Fatal("la lista blanca de tipos no se respeta")
	}
	if FormatDigest("ab") != "sha256:ab" {
		t.Fatal("FormatDigest")
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	if f := Validate([]byte(goodHTML())); len(f) != 0 {
		t.Fatalf("HTML válido con hallazgos: %v", f)
	}
	cases := []struct {
		name string
		html string
		want string
	}{
		{"script mayúsculas", strings.Replace(goodHTML(), "<body>", "<body><SCRIPT>1</SCRIPT>", 1), CodeScriptTag},
		{"src https", strings.Replace(goodHTML(), "<body>", `<body><img src="https://x/y.png">`, 1), CodeExternalURL},
		{"href protocolo relativo", strings.Replace(goodHTML(), "<body>", `<body><a href='//x'>a</a>`, 1), CodeExternalURL},
		{"xlink:href", strings.Replace(goodHTML(), "<body>", `<body><svg><use xlink:href="http://x/a.svg#i"/></svg>`, 1), CodeExternalURL},
		{"srcset", strings.Replace(goodHTML(), "<body>", `<body><img srcset="//x/a.png 2x">`, 1), CodeExternalURL},
		{"url css", strings.Replace(goodHTML(), "<body>", `<body><style>a{background:url(//x/a.png)}</style>`, 1), CodeExternalURL},
		{"import css", strings.Replace(goodHTML(), "<body>", `<body><style>@import "a.css";</style>`, 1), CodeExternalURL},
		{"meta ausente", strings.Replace(goodHTML(), `<meta name="axiom-source-digest" content="`+goodDigest+`">`, "", 1), CodeDigestMetaMissing},
		{"meta vacía", strings.Replace(goodHTML(), goodDigest, "", 1), CodeDigestMetaMissing},
		{"meta mal formada", strings.Replace(goodHTML(), goodDigest, "sha256:xyz", 1), CodeDigestMetaMissing},
		{"sin aviso", strings.Replace(goodHTML(), "lo que se aprueba es el md", "otra cosa", 1), CodeApprovalNoticeMissing},
		{"sin refs", strings.Replace(goodHTML(), `data-axiom-ref="REQ-1"`, "", 1), CodeCitedIDsMissing},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := codes(Validate([]byte(tc.html)))
			if !reflect.DeepEqual(got, []string{tc.want}) {
				t.Fatalf("hallazgos = %v, quería [%s]", got, tc.want)
			}
		})
	}
}

func TestValidateAdmitsDataURIsAndAnchors(t *testing.T) {
	t.Parallel()
	html := strings.Replace(goodHTML(), "<body>", `<body><img src="data:image/png;base64,AAAA"><a href="#x">a</a>`, 1)
	if f := Validate([]byte(html)); len(f) != 0 {
		t.Fatalf("data: y anclas deben admitirse: %v", f)
	}
}

func TestValidateApprovalNoticeAcrossLinesAndCase(t *testing.T) {
	t.Parallel()
	html := strings.Replace(goodHTML(), "lo que se aprueba es el md", "LO QUE SE\n  APRUEBA   ES EL MD", 1)
	if f := Validate([]byte(html)); len(f) != 0 {
		t.Fatalf("el aviso debe reconocerse tras colapsar espacios: %v", f)
	}
}

func TestSourceDigest(t *testing.T) {
	t.Parallel()
	upper := strings.ToUpper(goodDigest)
	cases := []struct {
		name   string
		html   string
		want   string
		wantOK bool
	}{
		{"orden normal", `<meta name="axiom-source-digest" content="` + goodDigest + `">`, goodDigest, true},
		{"orden invertido", `<meta content="` + goodDigest + `" name="axiom-source-digest">`, goodDigest, true},
		{"comillas simples", `<meta name='axiom-source-digest' content='` + goodDigest + `'>`, goodDigest, true},
		{"mayúsculas y espacios", `<META NAME="axiom-source-digest" CONTENT="  ` + upper + ` ">`, goodDigest, true},
		{"varias meta", `<meta charset="utf-8"><meta name="viewport" content="x"><meta name="axiom-source-digest" content="` + goodDigest + `">`, goodDigest, true},
		{"primera con el nombre", `<meta name="axiom-source-digest" content="sha256:aa"><meta name="axiom-source-digest" content="sha256:bb">`, "sha256:aa", true},
		{"sin content", `<meta name="axiom-source-digest">`, "", true},
		{"content vacío", `<meta name="axiom-source-digest" content="">`, "", true},
		{"ausente", `<meta name="viewport" content="x">`, "", false},
	}
	for _, tc := range cases {
		got, ok := SourceDigest([]byte(tc.html))
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("%s: SourceDigest = (%q, %v), quería (%q, %v)", tc.name, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestCitedIDs(t *testing.T) {
	t.Parallel()
	html := `<a data-axiom-ref="B"></a><i DATA-AXIOM-REF='A'></i><b data-axiom-ref="B"></b><c data-axiom-ref="prov:KOF:1"></c>`
	if got, want := CitedIDs([]byte(html)), []string{"B", "A", "prov:KOF:1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("CitedIDs = %v, quería %v", got, want)
	}
	if got := CitedIDs([]byte("<p>nada</p>")); len(got) != 0 {
		t.Fatalf("sin refs = %v", got)
	}
}

func TestReadFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	if _, err := ReadFile(filepath.Join(dir, "no-existe.html")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("inexistente: %v", err)
	}

	exact := filepath.Join(dir, "exacto.html")
	if err := os.WriteFile(exact, make([]byte, MaxBytes), 0o644); err != nil {
		t.Fatal(err)
	}
	if data, err := ReadFile(exact); err != nil || len(data) != MaxBytes {
		t.Fatalf("exactamente MaxBytes: len=%d err=%v", len(data), err)
	}

	big := filepath.Join(dir, "grande.html")
	if err := os.WriteFile(big, make([]byte, MaxBytes+1), 0o644); err != nil {
		t.Fatal(err)
	}
	if data, err := ReadFile(big); !errors.Is(err, ErrTooLarge) || data != nil {
		t.Fatalf("MaxBytes+1: len=%d err=%v", len(data), err)
	}

	if _, err := ReadFile(dir); !errors.Is(err, ErrNotRegular) {
		t.Fatalf("directorio: %v", err)
	}

	link := filepath.Join(dir, "enlace.html")
	if err := os.Symlink(exact, link); err != nil {
		t.Skipf("la plataforma no permite enlaces simbólicos: %v", err)
	}
	if _, err := ReadFile(link); !errors.Is(err, ErrNotRegular) {
		t.Fatalf("enlace simbólico: %v", err)
	}
}
