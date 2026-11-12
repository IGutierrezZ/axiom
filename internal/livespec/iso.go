package livespec

import (
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// ISOProjectError es el rechazo de la integración ODD en un proyecto que declara
// ISO (`sdd.iso: true`, REQ-30.7): el carril ODD no deja `Cubre:` ni matriz de
// verify, así que la spec viva de un proyecto ISO se compone por el carril SDD. Es
// distinguible con errors.As.
type ISOProjectError struct {
	ConfigPath    string // axiom.yaml que declara `sdd.iso: true`
	WorkspaceRoot string // raíz del espacio, para la continuación
	Feature       string // feature ODD rechazada
}

// Error nombra el axiom.yaml, el motivo y la continuación por el carril SDD.
func (e *ISOProjectError) Error() string {
	return fmt.Sprintf("archive integrate: %s declara `sdd.iso: true` y `axiom archive integrate --odd %s` no integra proyectos ISO: ODD no deja `Cubre:` ni matriz de verify; no se ha escrito nada. Continúa por el carril SDD: un bug con el perfil de bug y un incremento como cambio SDD, sellando antes su configuración con `axiom sdd kickoff seal --cwd %s --change <cambio> --execution-style continuous`",
		pathquote.Quote(e.ConfigPath), pathquote.ShellWord(e.Feature), pathquote.Quote(e.WorkspaceRoot))
}

// RefuseISOProject devuelve un *ISOProjectError si el proyecto declara ISO y nil si
// no: en un proyecto no ISO la integración ODD no se ve afectada (REQ-30.7). Se
// evalúa antes de leer el documento ODD y sin tocar el disco.
func RefuseISOProject(iso bool, configPath, workspaceRoot, feature string) error {
	if !iso {
		return nil
	}
	return &ISOProjectError{ConfigPath: configPath, WorkspaceRoot: workspaceRoot, Feature: feature}
}
