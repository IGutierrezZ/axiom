package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// La guarda A en los escritores CLI de la carpeta de un cambio (RF-BRA-20, RF-ARC-01, design D-11
// y G-10, tabla de la §2.11). Cada escritor la llama una sola vez, antes de escribir, con el
// contexto de la tabla:
//
//   - `ordinary`: `handoff create`, `gate record` (salvo `--gate integration`) y el marcador de
//     instancia de `sdd continue`;
//   - `archive`: `gate record --gate integration`, `integrate prepare` y `verify` (escriben
//     `integration.yaml`) y `archive-compose` sobre un cambio.
//
// Ante `wrong_branch` el escritor se niega sin escribir nada y devuelve el *branch.RefusalError
// de la guarda tal cual, con la orden exacta de `axiom branch ensure`. Un cambio exento
// (RF-KOF-07, RF-BRA-19) y una ruta que no se puede comprobar no se bloquean: esta última avisa.

// writerBranchWarnings recibe los avisos de la guarda de los escritores (ruta fuera de git,
// `HEAD` ilegible): la guarda no bloquea lo que no puede comprobar, pero lo dice. Es una
// variable para que las pruebas los recojan.
var writerBranchWarnings io.Writer = os.Stderr

// guardChangeWriter aplica la guarda A al escritor op de la carpeta changeRoot del cambio change,
// en el contexto ctx. Falla en cerrado ante un fallo interno de la consulta (`config_invalid`).
func guardChangeWriter(changeRoot, change, op string, ctx branch.Context) error {
	return guardChangeWriterCollect(changeRoot, change, op, ctx, func(text string) {
		_, _ = fmt.Fprintln(writerBranchWarnings, "Aviso: "+text)
	})
}

// guardChangeWriterCollect es guardChangeWriter con el aviso de la guarda entregado a warn (sin el
// prefijo «Aviso: ») en lugar de escrito: RecordGate lo devuelve como dato.
func guardChangeWriterCollect(changeRoot, change, op string, ctx branch.Context, warn func(string)) error {
	warning, err := engine.Default().RequireChangeBranch(context.Background(), engine.WriteTarget{
		ChangeRoot: changeRoot,
		Change:     change,
		Lane:       branch.LaneSDD,
		Context:    ctx,
		Op:         op,
	})
	if warning != "" {
		warn(warning)
	}
	return err
}

// GuardChangeWriter es la guarda A en el contexto `ordinary` para los escritores que viven fuera
// de este paquete (`axiom handoff create`, en `cmd/axiom`).
func GuardChangeWriter(changeRoot, change, op string) error {
	return guardChangeWriter(changeRoot, change, op, branch.ContextOrdinary)
}

// guardComposeDelta aplica la guarda A, en el contexto `archive`, a la composición de un delta
// que vive en `<planning home>/changes/<cambio>/specs/<dominio>/` (RF-BRA-20). Un delta fuera de
// la carpeta de un cambio no escribe en ninguna: no hay nada que guardar.
func guardComposeDelta(loc workspace.Location, deltaPath, domain string) error {
	changeRoot, ok := changeRootOfDelta(loc, deltaPath, domain)
	if !ok {
		return nil
	}
	return guardChangeWriter(changeRoot, filepath.Base(changeRoot), "sdd archive-compose", branch.ContextArchive)
}

// guardComposeLooseDelta es guardComposeDelta para la composición sobre una especificación
// concreta (`--canonical --delta`), que no resuelve el espacio de trabajo: lo localiza desde la
// carpeta del delta y toma el dominio de su carpeta. Un delta que no se puede situar en ningún
// espacio de trabajo no pertenece a un cambio y no se guarda.
func guardComposeLooseDelta(deltaPath string) error {
	absolute, err := filepath.Abs(deltaPath)
	if err != nil {
		return nil
	}
	dir := filepath.Dir(absolute)
	loc, err := workspace.Locate(dir)
	if err != nil {
		return nil
	}
	return guardComposeDelta(loc, absolute, filepath.Base(dir))
}
