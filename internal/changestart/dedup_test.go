package changestart

import "testing"

func TestDedupKeyEquivalent(t *testing.T) {
	base := "https://dev.azure.test/org/proj/_apis/wit/attachments/AbC-123"
	same := []string{
		base,
		"HTTPS://DEV.azure.TEST/org/proj/_apis/wit/attachments/AbC-123",
		"https://dev.azure.test:443/org/proj/_apis/wit/attachments/AbC-123",
		"https://dev.azure.test/ORG/Proj/_apis/wit/attachments/abc-123",
		"https://dev.azure.test/org/proj/_apis/wit/attachments/AbC-123?fileName=foto.png",
		"https://dev.azure.test/org/proj/_apis/wit/attachments/AbC-123?FILENAME=otra.png&api-version=7.1",
		"https://dev.azure.test/org/proj/_apis/wit/attachments/AbC-123?download=true&Api-Version=7.1-preview.3",
		"https://dev.azure.test/org/proj/_apis/wit/attachments/AbC-123#fragmento",
		"  https://dev.azure.test/org/proj/_apis/wit/attachments/AbC-123  ",
		"https://dev.azure.test/org/proj/_apis/wit/attachments/%41bC-123",
	}
	want, ok := dedupKey(base)
	if !ok {
		t.Fatalf("dedupKey(%q) debería analizarse", base)
	}
	for _, raw := range same {
		got, ok := dedupKey(raw)
		if !ok || got != want {
			t.Errorf("dedupKey(%q) = %q, %v; quiero %q", raw, got, ok, want)
		}
	}
}

func TestDedupKeyDistinct(t *testing.T) {
	base := "https://dev.azure.test/org/proj/_apis/wit/attachments/abc"
	different := []string{
		"https://dev.azure.test/org/proj/_apis/wit/attachments/abd",
		"https://otro.azure.test/org/proj/_apis/wit/attachments/abc",
		"http://dev.azure.test/org/proj/_apis/wit/attachments/abc",
		"https://dev.azure.test:8443/org/proj/_apis/wit/attachments/abc",
		"http://dev.azure.test:443/org/proj/_apis/wit/attachments/abc",
		"https://dev.azure.test/org/proj/_apis/wit/attachments/abc?version=2",
		"https://dev.azure.test/org/proj/_apis/wit/attachments/abc/",
	}
	want, _ := dedupKey(base)
	for _, raw := range different {
		if got, ok := dedupKey(raw); !ok || got == want {
			t.Errorf("dedupKey(%q) = %q, %v; no debe coincidir con la base", raw, got, ok)
		}
	}
}

func TestDedupKeyQueryOrderIsStable(t *testing.T) {
	a, _ := dedupKey("https://h.test/p?b=2&a=1")
	b, _ := dedupKey("https://h.test/p?a=1&b=2&fileName=x")
	if a != b {
		t.Fatalf("el orden de los parámetros no debe importar: %q != %q", a, b)
	}
}

func TestDedupKeyUnparseable(t *testing.T) {
	for _, raw := range []string{"http://[::1", "https://h.test/%zz", "http://a b/", "\x7f", "ht!tp://x"} {
		if key, ok := dedupKey(raw); ok {
			t.Errorf("dedupKey(%q) = %q; no debería analizarse", raw, key)
		}
	}
}
