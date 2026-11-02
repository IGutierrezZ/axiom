package state

import (
	"errors"
	"log"
	"os"
	"path/filepath"
)

// Store identifica el documento de estado de instalación que una operación lee y
// escribe. Hay dos ubicaciones (decisión D2 de huella-usuario-workspace):
//
//   - global: el estado de usuario de siempre, `~/.axiom/state.json` (o
//     `AXIOM_STATE_DIR`), que sigue siendo el ámbito de `--scope global` y el de
//     lo que es de máquina;
//   - de proyecto: `<raíz del workspace>/.axiom/state.json`, sin versionar, que es
//     el de `--scope workspace`. Así instalar en el proyecto A no pisa la
//     selección, la persona ni la asignación de modelos del proyecto B.
//
// El valor cero no es válido: se construye con GlobalStore o ProjectStore.
type Store struct {
	home string
	root string
}

// GlobalStore devuelve el almacén de usuario de homeDir.
func GlobalStore(homeDir string) Store {
	return Store{home: homeDir}
}

// ProjectStore devuelve el almacén del proyecto cuya raíz es root. homeDir sigue
// haciendo falta porque el estado global actúa de semilla de solo lectura mientras
// el proyecto no tiene el suyo. Con root vacío devuelve el almacén global.
func ProjectStore(homeDir, root string) Store {
	return Store{home: homeDir, root: root}
}

// ProjectPath devuelve la ruta del estado de instalación del proyecto con raíz root.
func ProjectPath(root string) string {
	return filepath.Join(root, stateDir, stateFile)
}

// IsProject indica si el almacén es el de un proyecto y no el global.
func (s Store) IsProject() bool { return s.root != "" }

// Home devuelve el directorio personal con el que se construyó el almacén.
func (s Store) Home() string { return s.home }

// Root devuelve la raíz del proyecto, o "" en el almacén global.
func (s Store) Root() string { return s.root }

// Path devuelve la ruta del documento de estado de este almacén.
func (s Store) Path() string {
	if s.IsProject() {
		return ProjectPath(s.root)
	}
	return Path(s.home)
}

// sharesGlobalFile dice si el almacén de proyecto apunta al mismo fichero que el
// global: pasa cuando la raíz es el propio HOME. Entonces no hay semilla que leer
// aparte ni proyecto que distinguir.
func (s Store) sharesGlobalFile() bool {
	return s.IsProject() && filepath.Clean(s.Path()) == filepath.Clean(Path(s.home))
}

// Read lee el estado del almacén. En el global equivale a Read(homeDir).
//
// En el de proyecto, si el proyecto todavía no tiene `.axiom/state.json`, devuelve
// el estado global como semilla de solo lectura, sin los campos de máquina: no lo
// modifica y la primera escritura crea el documento del proyecto. Un documento de
// proyecto que existe pero no se puede leer es un error y nunca vuelve a la
// semilla: sustituirlo por el global ocultaría el estado real del proyecto.
// Si no existe ni el del proyecto ni el global, devuelve os.ErrNotExist.
func (s Store) Read() (InstallState, error) {
	if !s.IsProject() || s.sharesGlobalFile() {
		return Read(s.home)
	}
	current, err := readStateFile(s.Path())
	if err == nil {
		return current, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return InstallState{}, err
	}
	seed, seedErr := Read(s.home)
	if seedErr != nil {
		return InstallState{}, seedErr
	}
	return ProjectView(seed), nil
}

// Write persiste el estado en el almacén. El de proyecto prepara antes su
// directorio y la exclusión de git (Prepare).
func (s Store) Write(current InstallState) error {
	if !s.IsProject() {
		return Write(s.home, current)
	}
	s.Prepare()
	return writeStateFile(s.Path(), current)
}

// WriteReconciled es Write con la tolerancia a errores de WriteReconciled.
func (s Store) WriteReconciled(current InstallState) error {
	if !s.IsProject() {
		return WriteReconciled(s.home, current)
	}
	s.Prepare()
	return writeStateFileReconciled(s.Path(), current)
}

// Prepare deja listo el directorio del estado de proyecto y lo excluye de git. No
// hace nada en el almacén global. Los fallos solo se registran: que no se pueda
// tocar `.git/info/exclude` no debe impedir persistir el estado ni dejar la
// instalación a medias.
func (s Store) Prepare() {
	if !s.IsProject() {
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.Path()), 0o755); err != nil {
		log.Printf("state: no se pudo crear %s: %v", filepath.Dir(s.Path()), err)
		return
	}
	if err := EnsureProjectStateIgnored(s.root); err != nil {
		log.Printf("state: no se pudo excluir el estado de %s del control de versiones: %v", s.root, err)
	}
}

// ProjectView devuelve el estado reducido a lo que es del proyecto: quita los
// campos que son de máquina. Se aplica a la semilla que viene del estado global
// para que el documento del proyecto no congele una copia obsoleta de ellos.
//
// Campos de máquina, que siempre viven en el estado global:
//   - LastUpdateCheck: la comprobación de versión remota es de la máquina;
//   - PendingSync: lo fija una autoactualización del binario, que es de la máquina;
//   - RDDMode y RDDModeRecordedAt: el interruptor de revisión es del usuario.
//
// La semilla tampoco aporta InstalledAgents: la lista de agentes de un proyecto es la
// de lo que se instaló en él, y el estado global es la unión de todos los proyectos
// (y de las instalaciones globales). Heredarla haría que un sync en el proyecto
// escribiera en HOME la configuración de agentes sin modo workspace. Con nil el
// proyecto no ha registrado agentes (una instalación escribe siempre una lista no
// nula, aunque vacía); véase Store.AgentsRecorded.
//
// El resto (agentes instalados, selección, persona, asignación de modelos,
// herramientas de comunidad, intenciones de segundo plano, huella del binario y de
// los activos gestionados) es del proyecto.
func ProjectView(s InstallState) InstallState {
	s.InstalledAgents = nil
	s.LastUpdateCheck = nil
	s.PendingSync = false
	s.RDDMode = ""
	s.RDDModeRecordedAt = nil
	return s
}

// WithMachineFields devuelve project con los campos de máquina de machine. Sirve a
// quien lee el estado del proyecto pero necesita lo que sigue siendo del estado
// global (ProjectView los enumera).
func WithMachineFields(project, machine InstallState) InstallState {
	project.LastUpdateCheck = machine.LastUpdateCheck
	project.PendingSync = machine.PendingSync
	project.RDDMode = machine.RDDMode
	project.RDDModeRecordedAt = machine.RDDModeRecordedAt
	return project
}

// AgentsRecorded dice si current, leído de este almacén, trae una lista de agentes
// instalados que valga. Siempre es así en el almacén global. En el de proyecto
// solo cuando el proyecto la ha registrado: la semilla global no la aporta
// (ProjectView) y un estado de proyecto creado por un sync a partir de la semilla
// tampoco, y en ambos casos InstalledAgents es nil. Una instalación escribe una
// lista no nula, aunque esté vacía, así que nil significa «no registrada» y no
// «ningún agente».
func (s Store) AgentsRecorded(current InstallState) bool {
	if !s.IsProject() || s.sharesGlobalFile() {
		return true
	}
	return current.InstalledAgents != nil
}
