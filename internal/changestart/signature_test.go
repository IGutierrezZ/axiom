package changestart

import (
	"archive/zip"
	"bytes"
	"testing"
)

var (
	pngBytes  = append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, "datos"...)
	jpegBytes = append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, "datos"...)
	gifBytes  = []byte("GIF89a\x01\x00")
	webpBytes = []byte("RIFF\x10\x00\x00\x00WEBPVP8 ")
	bmpBytes  = append([]byte("BM"), make([]byte, 20)...)
	pdfBytes  = []byte("%PDF-1.7\n%%EOF")
	oleBytes  = append([]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, make([]byte, 16)...)
)

// makeZip construye un ZIP con las entradas dadas (contenido mínimo).
func makeZip(t *testing.T, names ...string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, n := range names {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("<x/>")); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestSignatureTable(t *testing.T) {
	docx := makeZip(t, "[Content_Types].xml", "word/document.xml")
	xlsx := makeZip(t, "[Content_Types].xml", "xl/workbook.xml")
	pptx := makeZip(t, "[Content_Types].xml", "ppt/presentation.xml")
	noTypes := makeZip(t, "word/document.xml")

	tests := []struct {
		name string
		ext  string
		data []byte
		ok   bool
	}{
		{"png", "png", pngBytes, true},
		{"png con firma jpeg", "png", jpegBytes, false},
		{"png vacío", "png", nil, false},
		{"png truncado", "png", pngBytes[:5], false},
		{"jpg", "jpg", jpegBytes, true},
		{"jpeg", "jpeg", jpegBytes, true},
		{"jpg con firma png", "jpg", pngBytes, false},
		{"gif87", "gif", []byte("GIF87a.."), true},
		{"gif89", "gif", gifBytes, true},
		{"gif sin versión", "gif", []byte("GIF90a.."), false},
		{"webp", "webp", webpBytes, true},
		{"webp sin marca", "webp", []byte("RIFF\x10\x00\x00\x00WAVEfmt "), false},
		{"webp corto", "webp", []byte("RIFF\x10\x00\x00\x00WEB"), false},
		{"bmp", "bmp", bmpBytes, true},
		{"bmp corto", "bmp", []byte("BM123"), false},
		{"pdf", "pdf", pdfBytes, true},
		{"pdf html", "pdf", []byte("<html>"), false},
		{"docx", "docx", docx, true},
		{"docx es xlsx", "docx", xlsx, false},
		{"xlsx", "xlsx", xlsx, true},
		{"xlsx es pptx", "xlsx", pptx, false},
		{"pptx", "pptx", pptx, true},
		{"pptx es docx", "pptx", docx, false},
		{"docx sin content types", "docx", noTypes, false},
		{"docx no es zip", "docx", []byte("PK\x03\x04basura"), false},
		{"docx texto", "docx", []byte("hola"), false},
		{"doc ole", "doc", oleBytes, true},
		{"xls ole", "xls", oleBytes, true},
		{"ppt ole", "ppt", oleBytes, true},
		{"doc zip", "doc", docx, false},
		{"txt", "txt", []byte("hola ñandú\n"), true},
		{"txt con bom", "txt", append([]byte{0xEF, 0xBB, 0xBF}, "hola"...), true},
		{"txt vacío", "txt", nil, true},
		{"txt utf16", "txt", []byte{0xFF, 0xFE, 'h', 0, 'i', 0}, false},
		{"txt nul", "txt", []byte("a\x00b"), false},
		{"txt utf8 inválido", "txt", []byte("a\xffb"), false},
		{"txt html", "txt", []byte("<!DOCTYPE html><html><body>x</body></html>"), false},
		{"txt html mayúsculas", "txt", []byte("  <HTML><BODY>x"), false},
		{"txt xml", "txt", []byte("<?xml version=\"1.0\"?><a/>"), false},
		{"log", "log", []byte("linea 1\nlinea 2"), true},
		{"md", "md", []byte("# titulo"), true},
		{"csv", "csv", []byte("a,b\n1,2"), true},
		{"json", "json", []byte(`{"a":1}`), true},
		{"json binario", "json", pngBytes, false},
		{"extensión desconocida", "exe", []byte("MZ"), false},
		{"svg nunca", "svg", []byte("<svg/>"), false},
		{"extensión vacía", "", []byte("hola"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := signatureMatches(tc.ext, tc.data); got != tc.ok {
				t.Fatalf("signatureMatches(%q) = %v, quiero %v", tc.ext, got, tc.ok)
			}
		})
	}
}

func TestOOXMLCheckDoesNotDecompress(t *testing.T) {
	// El directorio central declara un tamaño descomprimido enorme y los datos son basura: solo una
	// implementación que descomprimiera fallaría o reservaría memoria.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, n := range []string{"[Content_Types].xml", "word/document.xml"} {
		w, err := zw.CreateRaw(&zip.FileHeader{
			Name: n, Method: zip.Deflate, CRC32: 1, CompressedSize64: 5, UncompressedSize64: 1 << 40,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("junk!")); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if !signatureMatches("docx", buf.Bytes()) {
		t.Fatal("la comprobación OOXML debe bastar con los nombres del directorio central")
	}
}

func TestImageExtFromSignature(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"png", pngBytes, "png"}, {"jpg", jpegBytes, "jpg"}, {"gif", gifBytes, "gif"},
		{"webp", webpBytes, "webp"}, {"bmp", bmpBytes, "bmp"},
		{"pdf no es imagen", pdfBytes, ""}, {"html", []byte("<html>"), ""}, {"vacío", nil, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := imageExtFromSignature(tc.data); got != tc.want {
				t.Fatalf("imageExtFromSignature = %q, quiero %q", got, tc.want)
			}
		})
	}
}

func TestExtensionAndContentTypeRules(t *testing.T) {
	for _, ext := range []string{"png", "jpg", "jpeg", "gif", "webp", "bmp", "pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "txt", "md", "csv", "json", "log"} {
		if !extensionAllowed(ext) {
			t.Errorf("la extensión %q debe estar permitida", ext)
		}
		if !contentTypeAllowed(ext, "application/octet-stream") {
			t.Errorf("octet-stream debe valer para %q", ext)
		}
	}
	for _, ext := range []string{"svg", "html", "htm", "sh", "ps1", "exe", "js", "", "PNG"} {
		if extensionAllowed(ext) {
			t.Errorf("la extensión %q no debe estar permitida", ext)
		}
	}
	accept := map[string]string{
		"png": "image/png", "jpg": "image/jpeg", "jpeg": "image/jpeg", "gif": "image/gif", "webp": "image/webp",
		"bmp": "image/x-ms-bmp", "pdf": "application/pdf", "doc": "application/msword",
		"docx": "application/zip", "xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"txt":  "text/plain", "md": "text/markdown", "csv": "text/csv", "json": "application/json",
	}
	for ext, ct := range accept {
		if !contentTypeAllowed(ext, ct) {
			t.Errorf("%q debe aceptar %q", ext, ct)
		}
	}
	for _, tc := range []struct{ ext, ct string }{
		{"png", "text/html"}, {"txt", "text/html"}, {"png", "application/xhtml+xml"}, {"png", "image/svg+xml"},
		{"png", "image/jpeg"}, {"pdf", "text/plain"}, {"md", "text/html"}, {"json", "text/csv"},
	} {
		if contentTypeAllowed(tc.ext, tc.ct) {
			t.Errorf("%q no debe aceptar %q", tc.ext, tc.ct)
		}
	}
	for _, ct := range []string{"text/html", "application/xhtml+xml", "image/svg+xml"} {
		if !neverAccepted(ct) {
			t.Errorf("%q nunca se acepta", ct)
		}
	}
	if neverAccepted("image/png") {
		t.Error("image/png sí es aceptable")
	}
}
