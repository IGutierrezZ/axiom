package workspace

import (
	"fmt"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// BaselineRetiredCode es el código estable del error de un bloque `baseline:`
// residual (REQ-30.21).
const BaselineRetiredCode = "baseline_retired"

// BaselineRetiredError es la causa tipada de un `axiom.yaml` que aún declara el
// bloque de primer nivel `baseline:`, retirado: la spec viva es únicamente
// `openspec/specs/`. El bloque nunca se ignora en silencio ni se degrada a «sin
// baseline»: cada consumidor (carga, `workspace validate`, status, archivado,
// `init`) lo recibe como error y nombra la continuación (REQ-30.21).
type BaselineRetiredError struct {
	// Path es el axiom.yaml que declara el bloque; vacío cuando se analizan bytes
	// sin fichero (ParseConfig).
	Path string
	// Line es la línea (desde 1) de la clave `baseline`.
	Line int
}

// Code devuelve el código estable `baseline_retired`.
func (e *BaselineRetiredError) Code() string { return BaselineRetiredCode }

// Continuation es la guía de migración: retirar el bloque y adoptar sus
// requisitos con `axiom archive adopt`.
func (e *BaselineRetiredError) Continuation() string {
	target := "<ruta>"
	if e.Path != "" {
		target = pathquote.Quote(filepath.Dir(e.Path))
	}
	return fmt.Sprintf("retira `baseline:` de axiom.yaml y migra los requisitos a `openspec/specs/` con `axiom archive adopt scan --path %s --change <cambio>`", target)
}

// Error compone el código, la ubicación del bloque y la continuación.
func (e *BaselineRetiredError) Error() string {
	where := "axiom.yaml"
	if e.Path != "" {
		where = pathquote.Quote(e.Path)
	}
	if e.Line > 0 {
		where = fmt.Sprintf("%s (línea %d)", where, e.Line)
	}
	return fmt.Sprintf("%s: %s declara el bloque `baseline:`, retirado: la spec viva es únicamente `openspec/specs/` y el bloque no se ignora; %s", BaselineRetiredCode, where, e.Continuation())
}

// withPath devuelve una copia con la ruta del axiom.yaml que declara el bloque.
func (e *BaselineRetiredError) withPath(path string) *BaselineRetiredError {
	if e == nil {
		return nil
	}
	copied := *e
	copied.Path = path
	return &copied
}

// bindConfigPath fija la ruta del axiom.yaml en el error de baseline retirada,
// que parseLayered no conoce porque solo recibe bytes.
func (d Declarations) bindConfigPath(path string) Declarations {
	d.BaselineRetiredErr = d.BaselineRetiredErr.withPath(path)
	return d
}

// GuardSpecsWrite devuelve nil si op puede crear o actualizar
// <planning home>/specs o <planning home>/INDEX.md, y un *BaselineRetiredError si
// el axiom.yaml resuelto declara aún un bloque `baseline:` (REQ-30.21 d): la
// escritura se detiene sin tocar `openspec/specs/`.
func (l Location) GuardSpecsWrite(op string) error {
	if l.Declarations.BaselineRetiredErr == nil {
		return nil
	}
	return l.Declarations.BaselineRetiredErr.withPath(l.ConfigPath)
}
