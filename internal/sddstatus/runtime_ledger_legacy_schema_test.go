package sddstatus

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeLegacyGrantRecord escribe en store un registro de grant tal y como lo
// dejaba un binario con el nombre heredado: esquema `gentle-ai.*` y digest de
// la petición calculado con el dominio heredado.
func writeLegacyGrantRecord(t *testing.T, store RuntimeStore, root, instance string) string {
	t.Helper()
	request := GrantRootsRequest{
		RequestID: "legacy-grant-1", Roots: []string{root},
		Reason: "autorizado antes de renombrar", Actor: "maintainer", ChangeInstance: instance,
	}
	record := runtimeRecord{
		Schema: legacyRuntimeRecordSchema, Change: store.Change,
		Operation: runtimeOperationGrant, RequestID: request.RequestID,
		RequestDigest: runtimeValueHash(legacyRuntimeGrantRequestDomain, request),
		Grant: &runtimeGrantEvent{
			Roots: request.Roots, Actor: request.Actor, Reason: request.Reason,
			GrantedAt: "2026-08-05T00:00:00Z", Instance: instance,
		},
	}
	revision, payload, err := runtimeRecordRevision(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ensureDirectories(); err != nil {
		t.Fatal(err)
	}
	if err := store.publishRecord(revision, payload); err != nil {
		t.Fatal(err)
	}
	if err := store.publishHead(revision); err != nil {
		t.Fatal(err)
	}
	return revision
}

// Un ledger escrito con los nombres heredados se lee: la autoridad concedida
// sigue vigente y una escritura nueva lleva el esquema axiom encadenada tras el
// registro viejo.
func TestRuntimeLedgerReadsGrantWrittenWithLegacySchemaAndAppendsAxiomSchema(t *testing.T) {
	repo := initRuntimeLedgerRepo(t)
	opened, err := OpenRuntimeStore(context.Background(), repo, "legacy-schema-grant")
	if err != nil {
		t.Fatal(err)
	}
	store, err := opened.ForInstance("legacy-schema-instance")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	legacyRevision := writeLegacyGrantRecord(t, store, root, "legacy-schema-instance")

	status, err := store.Status()
	if err != nil {
		t.Fatalf("lectura del ledger con esquema heredado: %v", err)
	}
	if len(status.GrantedRoots) != 1 || status.GrantedRoots[0] != root || status.Revision != legacyRevision {
		t.Fatalf("status = %#v, want la autoridad heredada vigente en %q", status, root)
	}
	if status.Schema != RuntimeStatusSchema || !strings.HasPrefix(status.Schema, "axiom.") {
		t.Fatalf("status.Schema = %q, want un esquema axiom", status.Schema)
	}

	second, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	granted, err := store.Grant(context.Background(), GrantRootsRequest{
		ExpectedRevision: legacyRevision, RequestID: "axiom-grant-2", Roots: []string{second},
		Reason: "nueva autorizacion", Actor: "maintainer",
	})
	if err != nil {
		t.Fatalf("grant encadenado tras un registro heredado: %v", err)
	}
	if len(granted.GrantedRoots) != 2 {
		t.Fatalf("granted roots = %#v, want las dos autoridades", granted.GrantedRoots)
	}
	head, err := store.loadRevision(granted.Revision)
	if err != nil || len(head.Status.GrantedRoots) != 2 {
		t.Fatalf("replay de la cadena mixta = %#v, %v", head.Status, err)
	}
	record, err := store.loadRecord(granted.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if record.Schema != runtimeRecordSchema {
		t.Fatalf("registro nuevo con esquema %q, want %q", record.Schema, runtimeRecordSchema)
	}
}

// Un registro heredado con el digest recalculado con el dominio nuevo (o al
// revés) no valida: cada registro se comprueba con el dominio con el que se
// escribió.
func TestRuntimeLedgerLegacySchemaRecordRejectsAxiomDomainDigest(t *testing.T) {
	repo := initRuntimeLedgerRepo(t)
	opened, err := OpenRuntimeStore(context.Background(), repo, "legacy-schema-mismatch")
	if err != nil {
		t.Fatal(err)
	}
	store, err := opened.ForInstance("legacy-mismatch-instance")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	request := GrantRootsRequest{
		RequestID: "mismatch-1", Roots: []string{root}, Reason: "x", Actor: "maintainer",
		ChangeInstance: "legacy-mismatch-instance",
	}
	record := runtimeRecord{
		Schema: legacyRuntimeRecordSchema, Change: store.Change,
		Operation: runtimeOperationGrant, RequestID: request.RequestID,
		RequestDigest: runtimeValueHash(runtimeGrantRequestDomain, request),
		Grant: &runtimeGrantEvent{
			Roots: request.Roots, Actor: request.Actor, Reason: request.Reason,
			GrantedAt: "2026-08-05T00:00:00Z", Instance: request.ChangeInstance,
		},
	}
	revision, payload, err := runtimeRecordRevision(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ensureDirectories(); err != nil {
		t.Fatal(err)
	}
	if err := store.publishRecord(revision, payload); err != nil {
		t.Fatal(err)
	}
	if err := store.publishHead(revision); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Status(); !isRuntimeRecordRejection(err, "grant_request_digest_match") {
		t.Fatalf("status = %v, want el rechazo por digest", err)
	}
}

// El directorio de un cambio cuyo nombre no es un slug lleva el digest de su
// identidad en la ruta. Con el dominio heredado en disco, abrir el almacén lo
// mueve a la ruta axiom y la autoridad concedida sigue vigente.
func TestOpenRuntimeStoreMigratesLegacyChangeIdentityDirectory(t *testing.T) {
	repo := initRuntimeLedgerRepo(t)
	const change = "Cambio con Mayusculas"
	probe, err := OpenRuntimeStore(context.Background(), repo, change)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(probe.commonDir, runtimeContainerName, "sdd-runtime")
	newDir := runtimeChangeLedgerDir(base, change)
	legacyDir := runtimeChangeLedgerDirFor(base, change, legacyRuntimeChangeIdentityDomain)
	if newDir == legacyDir || probe.Dir != newDir {
		t.Fatalf("rutas: nueva %q, heredada %q, abierta %q", newDir, legacyDir, probe.Dir)
	}
	// Se simula un ledger previo: el almacén vive en la ruta heredada.
	if err := os.RemoveAll(newDir); err != nil {
		t.Fatal(err)
	}
	legacyStore := probe
	legacyStore.Dir = legacyDir
	legacyStore, err = legacyStore.ForInstance("legacy-dir-instance")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeLegacyGrantRecord(t, legacyStore, root, "legacy-dir-instance")

	opened, err := OpenRuntimeStore(context.Background(), repo, change)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Dir != newDir {
		t.Fatalf("store.Dir = %q, want %q", opened.Dir, newDir)
	}
	if _, err := os.Lstat(legacyDir); !os.IsNotExist(err) {
		t.Fatalf("el directorio heredado sigue en %q (err=%v)", legacyDir, err)
	}
	store, err := opened.ForInstance("legacy-dir-instance")
	if err != nil {
		t.Fatal(err)
	}
	status, err := store.Status()
	if err != nil || len(status.GrantedRoots) != 1 || status.GrantedRoots[0] != root {
		t.Fatalf("status tras migrar = %#v, %v, want la autoridad vigente", status, err)
	}

	// Idempotente: reabrir no cambia nada y no pisa.
	again, err := OpenRuntimeStore(context.Background(), repo, change)
	if err != nil || again.Dir != newDir {
		t.Fatalf("reapertura = %q, %v", again.Dir, err)
	}
}
