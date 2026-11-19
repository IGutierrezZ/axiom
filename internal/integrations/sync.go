package integrations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// SyncOptions ajusta un sync.
type SyncOptions struct {
	// Change limita el sync a un cambio: su carpeta ordinaria y su carpeta archivada más reciente,
	// si existen. Vacío trata todas las carpetas con cola.
	Change string
	// RetryAbandoned reactiva las operaciones abandonadas (pone attempts a 0).
	RetryAbandoned bool
	// Discard retira las operaciones con esas claves, pendientes o abandonadas, sin entregarlas.
	Discard []string
}

// ChangeReport es el resultado de sync sobre una carpeta.
type ChangeReport struct {
	Change    string
	Folder    string
	Archived  bool
	Delivered []OperationReport
	Pending   []OperationReport
	Abandoned []OperationReport
	Discarded []OperationReport
	Warnings  []string
	// Skipped indica que la guarda rechazó la carpeta: su cola no se ha tratado y sigue ahí.
	Skipped bool
}

// SyncReport es el resultado de sync. Remaining indica que queda alguna operación, pendiente o
// abandonada, en las colas tratadas: sync sale con 0 solo si es falso.
type SyncReport struct {
	Changes   []ChangeReport
	Remaining bool
}

// Sync reintenta las operaciones pendientes de las carpetas con cola, en orden de encolado, con un
// plazo por operación. Un fallo no impide las siguientes. En una carpeta archivada no se fusionan
// vínculos y cada escritura pasa por la guarda de inmutabilidad (D-18). Sin cola, no lee ni escribe
// nada más.
func (d *Dispatcher) Sync(ctx context.Context, opts SyncOptions) (SyncReport, error) {
	var report SyncReport
	for _, folder := range d.syncFolders(opts.Change) {
		change, err := d.syncFolder(ctx, folder, opts)
		if err != nil {
			return report, err
		}
		report.Changes = append(report.Changes, change)
		if change.Skipped || len(change.Pending) > 0 || len(change.Abandoned) > 0 {
			report.Remaining = true
		}
	}
	return report, nil
}

// syncFolders devuelve las carpetas con cola que trata un sync: con change, su carpeta ordinaria y
// la archivada más reciente; sin él, todas las ordinarias y archivadas con cola.
func (d *Dispatcher) syncFolders(change string) []Folder {
	changes := filepath.Join(d.WorkspaceRoot, "openspec", "changes")
	var candidates []Folder
	if change != "" {
		ordinary := filepath.Join(changes, change)
		if info, err := os.Stat(ordinary); err == nil && info.IsDir() {
			candidates = append(candidates, Folder{WorkspaceRoot: d.WorkspaceRoot, Root: ordinary})
		}
		if archived, ok := kickoff.ArchivedChangeRoot(d.WorkspaceRoot, change); ok {
			candidates = append(candidates, Folder{WorkspaceRoot: d.WorkspaceRoot, Root: archived})
		}
	} else {
		for _, dir := range subdirectories(changes, "archive") {
			candidates = append(candidates, Folder{WorkspaceRoot: d.WorkspaceRoot, Root: dir})
		}
		for _, dir := range subdirectories(filepath.Join(changes, "archive"), "") {
			candidates = append(candidates, Folder{WorkspaceRoot: d.WorkspaceRoot, Root: dir})
		}
	}
	var withQueue []Folder
	for _, f := range candidates {
		if _, err := os.Stat(f.path(outboxName)); err == nil {
			withQueue = append(withQueue, f)
		}
	}
	return withQueue
}

// subdirectories lista, ordenadas por nombre, las carpetas de dir salvo la llamada skip.
func subdirectories(dir, skip string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != skip {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	paths := make([]string, len(names))
	for i, n := range names {
		paths[i] = filepath.Join(dir, n)
	}
	return paths
}

func (d *Dispatcher) syncFolder(ctx context.Context, folder Folder, opts SyncOptions) (ChangeReport, error) {
	report := ChangeReport{Change: folder.Change(), Folder: folder.Root, Archived: folder.Archived()}

	if d.Guard != nil {
		if err := d.Guard(folder); err != nil {
			report.Skipped = true
			report.Warnings = append(report.Warnings, fmt.Sprintf("se salta la cola de %s: %v", folder.Change(), err))
			return report, nil
		}
	}

	release, err := folder.acquire()
	if err != nil {
		return report, err
	}
	defer release()

	outbox, err := readOutbox(folder)
	if err != nil {
		return report, err
	}

	// Descartar: retira la operación con esa clave (de cualquier plugin) sin entregarla.
	for _, key := range opts.Discard {
		var plugins []string
		for _, op := range outbox.Operations {
			if op.Key == key {
				plugins = append(plugins, op.Plugin)
			}
		}
		if len(plugins) == 0 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("no hay ninguna operación con la clave %q en la cola de %s", key, folder.Change()))
			continue
		}
		for _, plugin := range plugins {
			op := outbox.find(plugin, key)
			report.Discarded = append(report.Discarded, opReport(*op, ReportDiscarded))
			outbox.remove(plugin, key)
		}
	}

	if opts.RetryAbandoned {
		for i := range outbox.Operations {
			if outbox.Operations[i].State == StateAbandoned {
				outbox.Operations[i].State = StatePending
				outbox.Operations[i].Attempts = 0
			}
		}
	}

	type operationID struct{ plugin, key string }
	var order []operationID
	for _, op := range outbox.Operations {
		order = append(order, operationID{op.Plugin, op.Key})
	}
	var delivered []operationID
	for _, id := range order {
		op := outbox.find(id.plugin, id.key)
		if op == nil || op.State != StatePending {
			continue
		}
		persist := !folder.Archived() && PolicyFor(op.Event.Type).PersistOutcome
		opCtx, cancel := context.WithTimeout(ctx, d.perOp())
		ok := d.attempt(opCtx, folder, op, persist, &report.Warnings)
		cancel()
		if ok {
			delivered = append(delivered, id)
		}
	}

	for _, id := range delivered {
		report.Delivered = append(report.Delivered, opReport(*outbox.find(id.plugin, id.key), ReportDelivered))
		outbox.remove(id.plugin, id.key)
	}
	for _, op := range outbox.Operations {
		if op.State == StateAbandoned {
			report.Abandoned = append(report.Abandoned, opReport(op, ReportAbandoned))
		} else {
			report.Pending = append(report.Pending, opReport(op, ReportPending))
		}
	}

	if err := writeOutbox(folder, outbox); err != nil {
		return report, fmt.Errorf("actualizar la cola de %s: %w", folder.Change(), err)
	}
	return report, nil
}
