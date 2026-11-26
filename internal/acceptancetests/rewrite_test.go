package acceptancetests

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

func idMap(pairs ...string) reqid.IDMap {
	var as []reqid.Assignment
	for i := 0; i+1 < len(pairs); i += 2 {
		as = append(as, reqid.Assignment{Provisional: pairs[i], Definitive: pairs[i+1]})
	}
	return reqid.NewIDMap(as)
}

func TestRewriteCubreByteForByte(t *testing.T) {
	prov := reqid.ProvisionalText("1", "1")
	other := reqid.ProvisionalText("1", "2")
	text := "# Pruebas " + prov + "\r\n\r\n" + header +
		row("Cubre", prov+"#Escenario "+prov+"; "+other+"; REQ-9.9#x") +
		"| P-02 | " + prov + " | fullstack | manual | alta | dev | pre " + prov + " | pasos " + prov + " | ok |\r\n" +
		"\nfin " + prov
	m := idMap(prov, "REQ-5.1", other, "REQ-5.2")
	got := RewriteCubre(text, m)
	want := strings.NewReplacer(
		"; "+other+";", "; REQ-5.2;",
		"| "+prov+" | fullstack", "| REQ-5.1 | fullstack",
		"| "+prov+"#Escenario", "| REQ-5.1#Escenario",
	).Replace(text)
	// Los demás usos de prov (título, nombre de escenario, pasos, fin) deben quedar intactos.
	if got != want {
		t.Fatalf("reescritura:\n%q\nesperado:\n%q", got, want)
	}
	if strings.Count(got, prov) != strings.Count(text, prov)-2 {
		t.Fatalf("se sustituyeron usos fuera de la parte ID de Cubre")
	}
}

func TestRewriteCubreIdempotentAndNoOps(t *testing.T) {
	prov := reqid.ProvisionalText("1", "1")
	text := header + row("Cubre", prov+"#A")
	m := idMap(prov, "REQ-5.1")
	once := RewriteCubre(text, m)
	if once == text || !strings.Contains(once, "REQ-5.1#A") {
		t.Fatalf("no reescribió: %q", once)
	}
	if twice := RewriteCubre(once, m); twice != once {
		t.Fatal("no es idempotente")
	}
	if RewriteCubre(text, reqid.NewIDMap(nil)) != text {
		t.Fatal("sin mapa debe devolver el texto igual")
	}
	if noTable := "solo " + prov + "\n"; RewriteCubre(noTable, m) != noTable {
		t.Fatal("sin tabla debe devolver el texto igual")
	}
	bad := strings.Replace(header, "Cubre", "Otra", 1) + row("Cubre", prov)
	if RewriteCubre(bad, m) != bad {
		t.Fatal("con cabecera inválida debe devolver el texto igual")
	}
}

func TestRewriteCubreLeavesDefinitiveIDsAndWrongWidthRows(t *testing.T) {
	prov := reqid.ProvisionalText("1", "1")
	short := "| P-09 | " + prov + " |\n"
	text := header + row("Cubre", "REQ-1.1#A") + short
	if got := RewriteCubre(text, idMap(prov, "REQ-5.1")); got != text {
		t.Fatalf("cambió algo que no debía: %q", got)
	}
}
