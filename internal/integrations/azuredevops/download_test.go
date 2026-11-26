package azuredevops_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
)

const attachmentPath = "/contoso/Fabrikam/_apis/wit/attachments/0f1e2d3c"

func TestDownloadOK(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedAttachment(attachmentPath, adotest.Blob{Data: []byte("\x89PNGdatos"), ContentType: "Image/PNG; charset=binary"})
	client, src := newTestClient(t, srv, nil)

	got, err := client.Download(context.Background(), srv.URL()+attachmentPath+"?fileName=a.png", 1<<20)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if string(got.Data) != "\x89PNGdatos" || got.ContentType != "image/png" {
		t.Fatalf("resultado = %q (%s); el Content-Type se normaliza sin parámetros y en minúsculas", got.Data, got.ContentType)
	}
	reqs := srv.Requests()
	if len(reqs) != 1 || reqs[0].Header.Get("Authorization") != basicFor(testPAT) || reqs[0].Query.Get("api-version") != "7.1" {
		t.Fatalf("la petición debe llevar el PAT y api-version: %+v", reqs)
	}
	if reqs[0].Query.Get("fileName") != "a.png" {
		t.Errorf("la consulta original se conserva: %q", reqs[0].RawQuery)
	}
	if src.calls.Load() != 1 {
		t.Errorf("el PAT se resuelve una vez: %d", src.calls.Load())
	}
}

func TestDownloadLimitAborts(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	// Sin Content-Length (siempre): solo el lector limitado puede detectar el exceso.
	srv.SeedAttachment(attachmentPath, adotest.Blob{Data: bytes.Repeat([]byte("x"), 1025), ContentType: "application/octet-stream"})
	client, _ := newTestClient(t, srv, nil)

	_, err := client.Download(context.Background(), srv.URL()+attachmentPath, 1024)
	var tooLarge *azuredevops.TooLargeError
	if !errors.As(err, &tooLarge) || tooLarge.Limit != 1024 {
		t.Fatalf("error = %v, esperaba *TooLargeError con límite 1024", err)
	}
	if !errors.Is(err, integrations.ErrAttachmentTooLarge) {
		t.Error("debe responder a integrations.ErrAttachmentTooLarge")
	}
	// Justo en el límite se acepta.
	srv.SeedAttachment(attachmentPath, adotest.Blob{Data: bytes.Repeat([]byte("x"), 1024)})
	if got, err := client.Download(context.Background(), srv.URL()+attachmentPath, 1024); err != nil || len(got.Data) != 1024 {
		t.Fatalf("1024 bytes con límite 1024 = %d, %v", len(got.Data), err)
	}
}

func TestDownloadForeignHostNoRequest(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	client, src := newTestClient(t, srv, nil)

	_, err := client.Download(context.Background(), "http://evil.example/contoso/Fabrikam/_apis/wit/attachments/x", 1024)
	var foreign *azuredevops.ForeignURLError
	if !errors.As(err, &foreign) || foreign.Reason != "host" {
		t.Fatalf("error = %v, esperaba *ForeignURLError host", err)
	}
	if len(srv.Requests()) != 0 || src.calls.Load() != 0 {
		t.Fatalf("sin red y sin PAT: peticiones=%d, consultas del secreto=%d", len(srv.Requests()), src.calls.Load())
	}
	if strings.Contains(err.Error(), "evil.example") {
		t.Errorf("el error no debe repetir la URL del tracker: %v", err)
	}
}

func TestDownloadRejectionReasons(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	client, src := newTestClient(t, srv, nil)
	hostPort := strings.TrimPrefix(srv.URL(), "http://")

	cases := []struct{ reason, url string }{
		{"scheme", "https://" + hostPort + "/contoso/Fabrikam/_apis/wit/attachments/x"},
		{"scheme", "file:///c:/x.png"},
		{"scheme", "ftp://" + hostPort + "/contoso/x"},
		{"userinfo", "http://usuario:clave@" + hostPort + "/contoso/Fabrikam/_apis/wit/attachments/x"},
		{"host", "http://otro.example/contoso/Fabrikam/_apis/wit/attachments/x"},
		{"organization", srv.URL() + "/otra/Fabrikam/_apis/wit/attachments/x"},
		{"organization", srv.URL() + "/contoso"},
		{"path", srv.URL() + "/contoso/../otra/x"},
		{"path", srv.URL() + "/contoso/%2e%2e/x"},
	}
	for _, tc := range cases {
		_, err := client.Download(context.Background(), tc.url, 1024)
		var foreign *azuredevops.ForeignURLError
		if !errors.As(err, &foreign) || foreign.Reason != tc.reason {
			t.Errorf("%s: error = %v, esperaba el motivo %s", tc.url, err, tc.reason)
			continue
		}
		if strings.Contains(err.Error(), "clave") || strings.Contains(err.Error(), "usuario") {
			t.Errorf("%s: el error repite credenciales: %v", tc.url, err)
		}
	}
	if len(srv.Requests()) != 0 || src.calls.Load() != 0 {
		t.Fatalf("ningún rechazo hace red ni resuelve el PAT: peticiones=%d, consultas=%d", len(srv.Requests()), src.calls.Load())
	}
}

func TestDownloadMissingContentType(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedAttachment(attachmentPath, adotest.Blob{Data: []byte("%PDF-1.7"), NoContentType: true})
	client, _ := newTestClient(t, srv, nil)

	got, err := client.Download(context.Background(), srv.URL()+attachmentPath, 1024)
	if err != nil || got.ContentType != "application/octet-stream" {
		t.Fatalf("sin Content-Type = %q, %v; esperaba application/octet-stream", got.ContentType, err)
	}
}

func TestDownloadRedirectNotFollowed(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	other := adotest.NewServer(t, nil) // el destino de la redirección: no debe recibir nada
	srv.SeedAttachment(attachmentPath, adotest.Blob{RedirectTo: other.URL() + "/login"})
	client, _ := newTestClient(t, srv, nil)

	_, err := client.Download(context.Background(), srv.URL()+attachmentPath, 1024)
	var auth *azuredevops.AuthError
	if !errors.As(err, &auth) {
		t.Fatalf("una redirección es *AuthError (no se sigue): %v", err)
	}
	if len(other.Requests()) != 0 {
		t.Fatalf("el destino de la redirección no debe recibir petición: %d", len(other.Requests()))
	}
}

func TestDownloadErrorSanitized(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", attachmentPath, adotest.Response{Status: http.StatusBadRequest,
		JSON: map[string]any{"message": "rechazado para " + testPAT}})
	client, _ := newTestClient(t, srv, nil)

	_, err := client.Download(context.Background(), srv.URL()+attachmentPath, 1024)
	if err == nil {
		t.Fatal("esperaba un error")
	}
	if strings.Contains(err.Error(), testPAT) {
		t.Fatalf("el error contiene el PAT: %v", err)
	}
	if !strings.Contains(err.Error(), "[oculto]") {
		t.Errorf("el error debe llevar el marcador de redacción: %v", err)
	}
}
