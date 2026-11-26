package changestart

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/IGutierrezZ/axiom/v3/internal/changestart/htmlmd"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

const fakeBase = "https://dev.azure.test/contoso/Fabrikam/_apis/wit/attachments/"

type fakeBlob struct {
	data []byte
	ct   string
	err  error
}

// fakeTracker es un descargador en memoria que imita los límites del adaptador.
type fakeTracker struct {
	blobs  map[string]fakeBlob
	urls   []string
	limits []int64
}

func newFake(blobs map[string]fakeBlob) *fakeTracker { return &fakeTracker{blobs: blobs} }

func (f *fakeTracker) download(_ context.Context, rawURL string, limit int64) (integrations.Attachment, error) {
	f.urls = append(f.urls, rawURL)
	f.limits = append(f.limits, limit)
	b, ok := f.blobs[rawURL]
	if !ok {
		return integrations.Attachment{}, errors.New("404")
	}
	if b.err != nil {
		return integrations.Attachment{}, b.err
	}
	if int64(len(b.data)) > limit {
		return integrations.Attachment{}, fmt.Errorf("demasiado grande: %w", integrations.ErrAttachmentTooLarge)
	}
	ct := b.ct
	if ct == "" {
		ct = ctOctetStream
	}
	return integrations.Attachment{Data: b.data, ContentType: ct}, nil
}

func attached(name, rawURL string, size int64) integrations.ItemAttachment {
	return integrations.ItemAttachment{Name: name, URL: rawURL, Size: size}
}

// adoDownloader usa el adaptador real contra el servidor local de adotest.
func adoDownloader(srv *adotest.Server) downloader {
	p := azuredevops.New(srv.Options())
	req := integrations.PortRequest{
		Config:  map[string]any{"organization": "contoso", "project": "Fabrikam", "team": "Fabrikam Team"},
		Secrets: staticPAT{},
	}
	return func(ctx context.Context, rawURL string, limit int64) (integrations.Attachment, error) {
		return p.DownloadAttachment(ctx, integrations.AttachmentRequest{PortRequest: req, URL: rawURL, MaxBytes: limit})
	}
}

type staticPAT struct{}

func (staticPAT) Secret(_ context.Context, _ string) (secrets.Secret, error) {
	return secrets.NewSecret("ficticio-pat-0123456789abcdef"), nil
}

const adoAttachmentPath = "/contoso/Fabrikam/_apis/wit/attachments/"

func onlyEntry(t *testing.T, p *attachmentPipeline) attachmentEntry {
	t.Helper()
	entries := p.entries()
	if len(entries) != 1 {
		t.Fatalf("entradas = %+v, esperaba una", entries)
	}
	return entries[0]
}

func TestAttachmentAllowed(t *testing.T) {
	f := newFake(map[string]fakeBlob{fakeBase + "a": {data: pngBytes, ct: "image/png"}})
	p := newAttachmentPipeline(f.download)
	p.importAttached(context.Background(), []integrations.ItemAttachment{attached("plano.png", fakeBase+"a", 100)})

	e := onlyEntry(t, p)
	if e.File != "plano.png" || e.Count != 1 || e.Reason != "" {
		t.Fatalf("entrada = %+v", e)
	}
	files := p.files()
	if len(files) != 1 || files[0].Name != "plano.png" || !bytes.Equal(files[0].Data, pngBytes) {
		t.Fatalf("ficheros = %+v", files)
	}
	if len(f.limits) != 1 || f.limits[0] != maxAttachmentBytes {
		t.Fatalf("el límite de descarga debe ser 10 MiB: %v", f.limits)
	}
}

func TestAttachmentExtensionRejected(t *testing.T) {
	for _, name := range []string{"run.sh", "run.ps1", "setup.exe", "pagina.html", "sin-extension", "lib.dll", "x.js"} {
		t.Run(name, func(t *testing.T) {
			f := newFake(map[string]fakeBlob{fakeBase + "a": {data: []byte("x")}})
			p := newAttachmentPipeline(f.download)
			p.importAttached(context.Background(), []integrations.ItemAttachment{attached(name, fakeBase+"a", 1)})
			if len(f.urls) != 0 {
				t.Fatalf("no debe haber petición: %v", f.urls)
			}
			e := onlyEntry(t, p)
			if e.File != "" || e.Label != name || e.Reason != "extensión no permitida" {
				t.Fatalf("entrada = %+v", e)
			}
			if len(p.files()) != 0 {
				t.Fatal("no debe haber ficheros")
			}
		})
	}
}

func TestAttachmentSVGRejected(t *testing.T) {
	f := newFake(map[string]fakeBlob{
		fakeBase + "b": {data: []byte("<svg/>"), ct: "image/svg+xml"},
		fakeBase + "c": {data: []byte("<svg/>"), ct: "image/svg+xml"},
	})
	p := newAttachmentPipeline(f.download)
	ctx := context.Background()
	p.importAttached(ctx, []integrations.ItemAttachment{attached("logo.svg", fakeBase+"a", 10)})
	if len(f.urls) != 0 {
		t.Fatalf("un .svg no se descarga: %v", f.urls)
	}
	// Una imagen con fileName .svg tampoco.
	if got := p.image(ctx, htmlmd.Image{Src: fakeBase + "b?fileName=logo.svg"}); got != "[imagen omitida: extensión no permitida]" {
		t.Fatalf("imagen svg = %q", got)
	}
	// Una imagen sin nombre que se sirve como SVG se descarta tras la descarga, con cualquier firma.
	if got := p.image(ctx, htmlmd.Image{Src: fakeBase + "c"}); got != "[imagen omitida: tipo de contenido no permitido]" {
		t.Fatalf("imagen servida como svg = %q", got)
	}
	if len(p.files()) != 0 {
		t.Fatal("no debe quedar ningún fichero")
	}
}

func TestAttachmentNamedLikeArtifactStaysInSubdir(t *testing.T) {
	f := newFake(map[string]fakeBlob{
		fakeBase + "a": {data: []byte("# no soy la tarea\n"), ct: "text/markdown"},
		fakeBase + "b": {data: []byte("# no soy la spec\n"), ct: "text/markdown"},
	})
	p := newAttachmentPipeline(f.download)
	p.importAttached(context.Background(), []integrations.ItemAttachment{
		attached("tasks.md", fakeBase+"a", 0), attached("spec.md", fakeBase+"b", 0),
	})
	ctxDir := filepath.Join(t.TempDir(), "context")
	dir := filepath.Join(ctxDir, "attachments")
	if err := writeAttachmentFiles(dir, p.files()); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tasks.md", "spec.md"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("falta %s bajo attachments/: %v", name, err)
		}
		if _, err := os.Stat(filepath.Join(ctxDir, name)); err == nil {
			t.Fatalf("%s no debe estar fuera de attachments/", name)
		}
	}
	entries, err := os.ReadDir(ctxDir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "attachments" {
		t.Fatalf("context/ solo debe contener attachments/: %v, %v", entries, err)
	}
}

func TestAttachmentFilesWrittenReadable(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "attachments")
	if err := writeAttachmentFiles(dir, []stagedFile{{Name: "a.txt", Data: []byte("hola")}}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != attachmentFileMode {
		t.Fatalf("permisos = %v, quiero %v", info.Mode().Perm(), os.FileMode(attachmentFileMode))
	}
	if info.Mode().Perm()&0o111 != 0 && runtime.GOOS != "windows" {
		t.Fatal("un adjunto nunca es ejecutable")
	}
}

func TestAttachmentContentTypeMismatch(t *testing.T) {
	f := newFake(map[string]fakeBlob{fakeBase + "a": {data: pngBytes, ct: "text/html"}})
	p := newAttachmentPipeline(f.download)
	p.importAttached(context.Background(), []integrations.ItemAttachment{attached("foto.png", fakeBase+"a", 10)})
	e := onlyEntry(t, p)
	if e.File != "" || e.Reason != "tipo de contenido no permitido" {
		t.Fatalf("entrada = %+v", e)
	}
	if len(p.files()) != 0 {
		t.Fatal("no debe dejar fichero")
	}
	// Otro tipo propio de otra extensión también discrepa.
	f.blobs[fakeBase+"b"] = fakeBlob{data: pngBytes, ct: "application/pdf"}
	p.importAttached(context.Background(), []integrations.ItemAttachment{attached("otra.png", fakeBase+"b", 10)})
	if len(p.files()) != 0 {
		t.Fatal("application/pdf con .png no debe aceptarse")
	}
}

func TestAttachmentOctetStreamWithMatchingSignature(t *testing.T) {
	f := newFake(map[string]fakeBlob{fakeBase + "a": {data: pngBytes, ct: ctOctetStream}})
	p := newAttachmentPipeline(f.download)
	p.importAttached(context.Background(), []integrations.ItemAttachment{attached("foto.PNG", fakeBase+"a", 10)})
	if e := onlyEntry(t, p); e.File != "foto.png" {
		t.Fatalf("entrada = %+v (la extensión se escribe en minúsculas)", e)
	}
}

func TestAttachmentOctetStreamHTMLBody(t *testing.T) {
	for _, name := range []string{"foto.png", "nota.txt", "datos.json", "doc.pdf"} {
		t.Run(name, func(t *testing.T) {
			f := newFake(map[string]fakeBlob{fakeBase + "a": {data: []byte("<!DOCTYPE html><html><script>x</script></html>")}})
			p := newAttachmentPipeline(f.download)
			p.importAttached(context.Background(), []integrations.ItemAttachment{attached(name, fakeBase+"a", 10)})
			if e := onlyEntry(t, p); e.File != "" || e.Reason != "tipo de contenido no permitido" {
				t.Fatalf("entrada = %+v", e)
			}
			if len(p.files()) != 0 {
				t.Fatal("no debe dejar fichero")
			}
		})
	}
}

func TestAttachmentNoContentTypeIsOctetStream(t *testing.T) {
	// El adaptador devuelve octet-stream cuando falta la cabecera; el pipeline también tolera "".
	dl := func(ctx context.Context, rawURL string, limit int64) (integrations.Attachment, error) {
		return integrations.Attachment{Data: pngBytes, ContentType: ""}, nil
	}
	p := newAttachmentPipeline(dl)
	p.importAttached(context.Background(), []integrations.ItemAttachment{attached("foto.png", fakeBase+"a", 10)})
	if e := onlyEntry(t, p); e.File != "foto.png" {
		t.Fatalf("entrada = %+v", e)
	}

	srv := adotest.NewServer(t, nil)
	srv.SeedAttachment(adoAttachmentPath+"x", adotest.Blob{Data: pngBytes, NoContentType: true})
	p = newAttachmentPipeline(adoDownloader(srv))
	p.importAttached(context.Background(), []integrations.ItemAttachment{attached("foto.png", srv.URL()+adoAttachmentPath+"x", 0)})
	if e := onlyEntry(t, p); e.File != "foto.png" {
		t.Fatalf("sin Content-Type real: entrada = %+v", e)
	}
}

func TestImageSkippedWithoutRequest(t *testing.T) {
	tests := []struct {
		name, src, reason string
	}{
		{"data", "data:image/png;base64,iVBORw0KGgo=", "imagen en línea (data:)"},
		{"relative", "img/captura.png", "ruta relativa"},
		{"file", "file:///etc/passwd", "esquema no permitido"},
		{"ftp", "ftp://ftp.example.test/a.png", "esquema no permitido"},
		{"http", "http://evil.example.test/contoso/Fabrikam/_apis/wit/attachments/x?fileName=a.png", "host no permitido"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := adotest.NewServer(t, nil)
			p := newAttachmentPipeline(adoDownloader(srv))
			got := p.image(context.Background(), htmlmd.Image{Src: tc.src, Alt: "x"})
			if want := "[imagen omitida: " + tc.reason + "]"; got != want {
				t.Fatalf("image = %q, quiero %q", got, want)
			}
			if strings.Contains(got, "example") || strings.Contains(got, "etc/passwd") || strings.Contains(got, "iVBOR") {
				t.Fatalf("el marcado no debe contener la URL: %q", got)
			}
			if n := len(srv.Requests()); n != 0 {
				t.Fatalf("peticiones al servidor = %d, esperaba 0", n)
			}
			if e := onlyEntry(t, p); e.File != "" || e.Label != "" || e.Reason != tc.reason || e.Count != 1 {
				t.Fatalf("entrada = %+v", e)
			}
		})
	}
}

func TestImageDataAndRelativeDoNotDownload(t *testing.T) {
	f := newFake(nil)
	p := newAttachmentPipeline(f.download)
	for i := 0; i < 5; i++ {
		p.image(context.Background(), htmlmd.Image{Src: fmt.Sprintf("data:image/png;base64,AAAA%d", i)})
	}
	p.image(context.Background(), htmlmd.Image{Src: "a.png"})
	p.image(context.Background(), htmlmd.Image{Src: "b.png"})
	if len(f.urls) != 0 {
		t.Fatalf("data: y relativas no hacen petición: %v", f.urls)
	}
	entries := p.entries()
	if len(entries) != 2 || entries[0].Count != 5 || entries[1].Count != 2 {
		t.Fatalf("data: y rutas relativas se agrupan por motivo: %+v", entries)
	}
}

func TestImageDedupSingleDownload(t *testing.T) {
	f := newFake(map[string]fakeBlob{fakeBase + "Img1?fileName=a.png": {data: pngBytes}})
	p := newAttachmentPipeline(f.download)
	ctx := context.Background()
	first := p.image(ctx, htmlmd.Image{Src: fakeBase + "Img1?fileName=a.png", Alt: "uno"})
	// La misma URL con otro nombre, otra grafía del host y de la ruta es el mismo recurso.
	second := p.image(ctx, htmlmd.Image{Src: "HTTPS://DEV.azure.test:443/contoso/Fabrikam/_apis/wit/attachments/img1?fileName=b.png&api-version=7.1", Alt: "dos"})
	if first != "![uno](attachments/a.png)" || second != "![dos](attachments/a.png)" {
		t.Fatalf("first=%q second=%q", first, second)
	}
	if len(f.urls) != 1 {
		t.Fatalf("descargas = %d, esperaba 1", len(f.urls))
	}
	if e := onlyEntry(t, p); e.File != "a.png" || e.Count != 2 {
		t.Fatalf("entrada = %+v", e)
	}
	if len(p.files()) != 1 {
		t.Fatalf("ficheros = %d", len(p.files()))
	}
}

func TestImageDedupOmittedSharesReason(t *testing.T) {
	f := newFake(map[string]fakeBlob{fakeBase + "x": {data: []byte("<html>"), ct: ctOctetStream}})
	p := newAttachmentPipeline(f.download)
	a := p.image(context.Background(), htmlmd.Image{Src: fakeBase + "x"})
	b := p.image(context.Background(), htmlmd.Image{Src: fakeBase + "X?download=true"})
	if a != b || a != "[imagen omitida: tipo de contenido no permitido]" || len(f.urls) != 1 {
		t.Fatalf("a=%q b=%q descargas=%d", a, b, len(f.urls))
	}
}

func TestAttachmentAndImageSameURLDedup(t *testing.T) {
	f := newFake(map[string]fakeBlob{fakeBase + "g1": {data: pngBytes, ct: "image/png"}})
	p := newAttachmentPipeline(f.download)
	ctx := context.Background()
	p.importAttached(ctx, []integrations.ItemAttachment{attached("pantalla.png", fakeBase+"g1", 10)})
	got := p.image(ctx, htmlmd.Image{Src: fakeBase + "g1", Alt: "captura"})
	if got != "![captura](attachments/pantalla.png)" {
		t.Fatalf("image = %q", got)
	}
	if len(f.urls) != 1 {
		t.Fatalf("descargas = %d, esperaba 1", len(f.urls))
	}
	if e := onlyEntry(t, p); e.File != "pantalla.png" || e.Count != 2 {
		t.Fatalf("entrada = %+v", e)
	}
}

func TestImageNoFileNameUsesSignature(t *testing.T) {
	f := newFake(map[string]fakeBlob{
		fakeBase + "a": {data: pngBytes},
		fakeBase + "b": {data: jpegBytes},
		fakeBase + "c": {data: pdfBytes},
	})
	p := newAttachmentPipeline(f.download)
	ctx := context.Background()
	if got := p.image(ctx, htmlmd.Image{Src: fakeBase + "a"}); got != "![](attachments/imagen.png)" {
		t.Fatalf("png = %q", got)
	}
	if got := p.image(ctx, htmlmd.Image{Src: fakeBase + "b"}); got != "![](attachments/imagen.jpg)" {
		t.Fatalf("jpg = %q", got)
	}
	// Un PDF sin nombre no es una imagen: sin firma de imagen se omite.
	if got := p.image(ctx, htmlmd.Image{Src: fakeBase + "c"}); got != "[imagen omitida: tipo de contenido no permitido]" {
		t.Fatalf("pdf = %q", got)
	}
}

func TestImageAltIsEscaped(t *testing.T) {
	f := newFake(map[string]fakeBlob{fakeBase + "a": {data: pngBytes}})
	p := newAttachmentPipeline(f.download)
	got := p.image(context.Background(), htmlmd.Image{Src: fakeBase + "a", Alt: "a](http://evil.test)\n[b"})
	if strings.Contains(got, "](http") || strings.Contains(got, "\n") {
		t.Fatalf("el alt rompió el enlace: %q", got)
	}
}

func TestAttachmentOverTenMiBNoContentLength(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	big := append(append([]byte{}, pngBytes...), make([]byte, maxAttachmentBytes)...)
	srv.SeedAttachment(adoAttachmentPath+"big", adotest.Blob{Data: big}) // adotest nunca envía Content-Length
	p := newAttachmentPipeline(adoDownloader(srv))
	p.importAttached(context.Background(), []integrations.ItemAttachment{attached("big.png", srv.URL()+adoAttachmentPath+"big", 0)})
	if e := onlyEntry(t, p); e.File != "" || e.Reason != "supera 10 MiB" {
		t.Fatalf("entrada = %+v", e)
	}
	if len(p.files()) != 0 {
		t.Fatal("no debe dejar fichero")
	}

	// El tamaño declarado mayor que 10 MiB se rechaza sin petición.
	f := newFake(nil)
	p = newAttachmentPipeline(f.download)
	p.importAttached(context.Background(), []integrations.ItemAttachment{attached("grande.pdf", fakeBase+"z", maxAttachmentBytes+1)})
	if e := onlyEntry(t, p); e.Reason != "supera 10 MiB" || len(f.urls) != 0 {
		t.Fatalf("entrada = %+v, peticiones = %v", e, f.urls)
	}
}

func TestAttachmentTotalBudget(t *testing.T) {
	const each = 9 << 20
	body := make([]byte, each)
	copy(body, pngBytes)
	blobs := map[string]fakeBlob{}
	var atts []integrations.ItemAttachment
	for i := 0; i < 7; i++ {
		u := fmt.Sprintf("%sb%d", fakeBase, i)
		blobs[u] = fakeBlob{data: body}
		atts = append(atts, attached(fmt.Sprintf("f%d.png", i), u, each))
	}
	f := newFake(blobs)
	p := newAttachmentPipeline(f.download)
	p.importAttached(context.Background(), atts)

	entries := p.entries()
	if len(entries) != 7 {
		t.Fatalf("entradas = %d", len(entries))
	}
	for i, e := range entries {
		if i < 5 && e.File == "" {
			t.Fatalf("la entrada %d debía descargarse: %+v", i, e)
		}
		if i >= 5 && (e.File != "" || e.Reason != "presupuesto total agotado") {
			t.Fatalf("la entrada %d debía agotar el presupuesto: %+v", i, e)
		}
	}
	if len(f.urls) != 5 {
		t.Fatalf("descargas = %d; las que superan el presupuesto declarado no hacen petición", len(f.urls))
	}

	// Sin tamaño declarado: el límite de la descarga es el presupuesto restante y el motivo es el
	// presupuesto, no los 10 MiB.
	f = newFake(blobs)
	p = newAttachmentPipeline(f.download)
	for i := range atts {
		atts[i].Size = 0
	}
	p.importAttached(context.Background(), atts)
	last := p.entries()[5]
	if last.File != "" || last.Reason != "presupuesto total agotado" {
		t.Fatalf("sexta entrada = %+v", last)
	}
	if lim := f.limits[5]; lim >= maxAttachmentBytes || lim != attachmentBudget-5*each {
		t.Fatalf("límite de la sexta descarga = %d", lim)
	}
}

func TestAttachmentFailureDoesNotAbort(t *testing.T) {
	f := newFake(map[string]fakeBlob{
		fakeBase + "a": {err: errors.New("503 servicio no disponible")},
		fakeBase + "b": {data: pngBytes},
	})
	p := newAttachmentPipeline(f.download)
	p.importAttached(context.Background(), []integrations.ItemAttachment{
		attached("uno.png", fakeBase+"a", 0), attached("dos.png", fakeBase+"b", 0), attached("tres.png", fakeBase+"zz", 0),
	})
	entries := p.entries()
	if len(entries) != 3 {
		t.Fatalf("entradas = %+v", entries)
	}
	if entries[0].Reason != "descarga fallida" || entries[1].File != "dos.png" || entries[2].Reason != "descarga fallida" {
		t.Fatalf("entradas = %+v", entries)
	}
	if strings.Contains(entries[0].Reason, "503") {
		t.Fatal("el motivo es un texto fijo, no el error del tracker")
	}
}

func TestImageForeignHostNoRequest(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	p := newAttachmentPipeline(adoDownloader(srv))
	got := p.image(context.Background(), htmlmd.Image{Src: "http://attacker.example.test/contoso/Fabrikam/_apis/wit/attachments/x"})
	if got != "[imagen omitida: host no permitido]" || len(srv.Requests()) != 0 {
		t.Fatalf("got=%q peticiones=%d", got, len(srv.Requests()))
	}
}

func TestAttachmentSameOrgAuthenticated(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedAttachment(adoAttachmentPath+"ok", adotest.Blob{Data: pngBytes})
	p := newAttachmentPipeline(adoDownloader(srv))
	p.importAttached(context.Background(), []integrations.ItemAttachment{attached("ok.png", srv.URL()+adoAttachmentPath+"ok", 0)})
	if e := onlyEntry(t, p); e.File != "ok.png" {
		t.Fatalf("entrada = %+v", e)
	}
	reqs := srv.Requests()
	if len(reqs) != 1 || reqs[0].Header.Get("Authorization") == "" {
		t.Fatalf("la descarga de la misma organización debe llevar la credencial: %+v", reqs)
	}
}

func TestAttachmentOtherOrgSkipped(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedAttachment("/otra/Fabrikam/_apis/wit/attachments/x", adotest.Blob{Data: pngBytes})
	p := newAttachmentPipeline(adoDownloader(srv))
	p.importAttached(context.Background(), []integrations.ItemAttachment{attached("x.png", srv.URL()+"/otra/Fabrikam/_apis/wit/attachments/x", 0)})
	if e := onlyEntry(t, p); e.File != "" || e.Reason != "organización no permitida" {
		t.Fatalf("entrada = %+v", e)
	}
	if len(srv.Requests()) != 0 {
		t.Fatalf("no debe haber peticiones: %+v", srv.Requests())
	}
}

func TestAttachmentHTTPAndUserinfoSkipped(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	host := strings.TrimPrefix(srv.URL(), "http://")
	userinfo := "http://" + "u" + ":" + "p" + "@" + host + adoAttachmentPath + "x"
	otherScheme := "https://" + host + adoAttachmentPath + "x"
	p := newAttachmentPipeline(adoDownloader(srv))
	p.importAttached(context.Background(), []integrations.ItemAttachment{
		attached("a.png", userinfo, 0), attached("b.png", otherScheme, 0),
	})
	entries := p.entries()
	if len(entries) != 2 || entries[0].Reason != "credenciales en la URL" || entries[1].Reason != "esquema no permitido" {
		t.Fatalf("entradas = %+v", entries)
	}
	if len(srv.Requests()) != 0 {
		t.Fatalf("no debe haber peticiones: %+v", srv.Requests())
	}
}

func TestAttachmentRedirectNotFollowed(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedAttachment(adoAttachmentPath+"r", adotest.Blob{RedirectTo: srv.URL() + adoAttachmentPath + "destino"})
	srv.SeedAttachment(adoAttachmentPath+"destino", adotest.Blob{Data: pngBytes})
	p := newAttachmentPipeline(adoDownloader(srv))
	p.importAttached(context.Background(), []integrations.ItemAttachment{attached("r.png", srv.URL()+adoAttachmentPath+"r", 0)})
	if e := onlyEntry(t, p); e.File != "" || e.Reason != "descarga fallida" {
		t.Fatalf("entrada = %+v", e)
	}
	if n := srv.Count("GET", adoAttachmentPath+"destino"); n != 0 {
		t.Fatalf("la redirección se siguió: %d lecturas del destino", n)
	}
}

func TestAttachmentCandidateCap(t *testing.T) {
	blobs := map[string]fakeBlob{}
	var atts []integrations.ItemAttachment
	for i := 0; i < maxCandidates+30; i++ {
		u := fmt.Sprintf("%sc%d", fakeBase, i)
		blobs[u] = fakeBlob{data: []byte("x")}
		atts = append(atts, attached(fmt.Sprintf("n%d.txt", i), u, 0))
	}
	f := newFake(blobs)
	p := newAttachmentPipeline(f.download)
	p.importAttached(context.Background(), atts)
	if len(f.urls) != maxCandidates {
		t.Fatalf("descargas = %d, esperaba %d", len(f.urls), maxCandidates)
	}
	entries := p.entries()
	last := entries[len(entries)-1]
	if len(entries) != maxCandidates+1 || last.Count != 30 || last.Reason != "tope de 100 recursos alcanzado" {
		t.Fatalf("entradas = %d, última = %+v", len(entries), last)
	}
	// Una repetición de un recurso ya aceptado sigue funcionando pasado el tope y no consume plaza.
	got := p.image(context.Background(), htmlmd.Image{Src: fakeBase + "c0"})
	if got != "![](attachments/n0.txt)" {
		t.Fatalf("la repetición pasado el tope = %q", got)
	}
}

func TestAttachmentInvalidURLGrouped(t *testing.T) {
	f := newFake(nil)
	p := newAttachmentPipeline(f.download)
	got := p.image(context.Background(), htmlmd.Image{Src: "http://[::1"})
	if got != "[imagen omitida: URL no válida]" || len(f.urls) != 0 {
		t.Fatalf("got=%q descargas=%v", got, f.urls)
	}
}

func TestAttachmentNamesSanitized(t *testing.T) {
	long := strings.Repeat("x", 100) + ".png"
	tests := []struct{ in, want string }{
		{"foto.png", "foto.png"},
		{"../../etc/passwd.txt", "passwd.txt"},
		{`C:\temp\a b.png`, "a_b.png"},
		{"dir/sub\\otro.pdf", "otro.pdf"},
		{"Informe Ñandú.PDF", "Informe_Nandu.pdf"},
		{"straße æon œuvre łódź ø.txt", "strasse_aeon_oeuvre_lodz_o.txt"},
		{"ÁÉÍÓÚ.png", "AEIOU.png"},
		{"CON.txt", "_CON.txt"},
		{"nul", "_nul"},
		{"Lpt1.log", "_Lpt1.log"},
		{"com9.png", "_com9.png"},
		{"console.txt", "console.txt"},
		{"  ..oculto.png..  ", "oculto.png"},
		{"a..b...png", "a.b.png"},
		{"", "adjunto"},
		{"...", "adjunto"},
		{"/", "adjunto"},
		{"日本語.png", "___.png"},
		{"a\x00b\x1b.png", "a_b_.png"},
		{"sin extension", "sin_extension"},
		{long, strings.Repeat("x", 44) + ".png"},
		{"FOTO.JPEG", "FOTO.jpeg"},
	}
	for _, tc := range tests {
		got := sanitizeName(tc.in)
		if got != tc.want {
			t.Errorf("sanitizeName(%q) = %q, quiero %q", tc.in, got, tc.want)
		}
		if utf8.RuneCountInString(got) > maxNameLen {
			t.Errorf("sanitizeName(%q) = %q supera %d caracteres", tc.in, got, maxNameLen)
		}
	}
}

func TestAttachmentNamesSanitizedInPipeline(t *testing.T) {
	f := newFake(map[string]fakeBlob{fakeBase + "a": {data: []byte("hola")}})
	p := newAttachmentPipeline(f.download)
	p.importAttached(context.Background(), []integrations.ItemAttachment{attached("../../../Windows/System32/léeme.txt", fakeBase+"a", 0)})
	if e := onlyEntry(t, p); e.File != "leeme.txt" {
		t.Fatalf("entrada = %+v", e)
	}
}

func TestAttachmentNameCollisions(t *testing.T) {
	blobs := map[string]fakeBlob{}
	var atts []integrations.ItemAttachment
	for i, name := range []string{"a.txt", "a.txt", "A.TXT", "a.txt"} {
		u := fmt.Sprintf("%sk%d", fakeBase, i)
		blobs[u] = fakeBlob{data: []byte("x")}
		atts = append(atts, attached(name, u, 0))
	}
	longName := strings.Repeat("y", 60) + ".txt"
	for i := 0; i < 3; i++ {
		u := fmt.Sprintf("%sl%d", fakeBase, i)
		blobs[u] = fakeBlob{data: []byte("x")}
		atts = append(atts, attached(longName, u, 0))
	}
	f := newFake(blobs)
	p := newAttachmentPipeline(f.download)
	p.importAttached(context.Background(), atts)

	var names []string
	for _, e := range p.entries() {
		names = append(names, e.File)
	}
	want := []string{"a.txt", "a-2.txt", "A-3.txt", "a-4.txt"}
	if fmt.Sprint(names[:4]) != fmt.Sprint(want) {
		t.Fatalf("nombres = %v, quiero %v", names[:4], want)
	}
	seen := map[string]bool{}
	for _, n := range names[4:] {
		if utf8.RuneCountInString(n) > maxNameLen || !strings.HasSuffix(n, ".txt") || seen[strings.ToLower(n)] {
			t.Fatalf("nombre largo con colisión inválido: %q (%v)", n, names[4:])
		}
		seen[strings.ToLower(n)] = true
	}
	if len(seen) != 3 {
		t.Fatalf("las tres colisiones largas deben ser distintas: %v", names[4:])
	}
}
