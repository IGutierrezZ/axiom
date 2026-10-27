package engine

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Inventario de escritores de la carpeta de un cambio (INC-25, REQ-25.19, design D-11 y D-23).
//
// La guarda A (RequireChangeBranch) solo protege lo que la llama. Esta prueba fija, con go/parser
// sobre el código de producción, que toda función que escribe en la carpeta de un cambio figura en
// la tabla de abajo, con su guarda o con el motivo de su exención. Una función nueva que escriba
// sin figurar en la tabla hace fallar la prueba: quien la añade decide entonces, a propósito, si
// la guarda o la exime.
//
// QUÉ DETECTA (de forma determinista, solo por sintaxis):
//
//  1. Llamadas por selector, resueltas por la ruta de importación y no por el alias local, a
//     kickoff.Seal, kickoff.AppendGate, handoff.WriteFile y sddstatus.PrepareChangeInstanceConsent,
//     más specbaseline.Prepare, Verify y ApplyCompose (escriben integration.yaml y los documentos
//     del baseline; el plan nombra las cuatro primeras y esta ampliación cubre `integrate prepare`,
//     `integrate verify` y `archive-compose --in-place`). También la llamada sin cualificar dentro
//     del propio paquete que define la función.
//  2. Una aproximación de «escribe bajo changes/»: una función que llama a una primitiva de
//     escritura del paquete os (WriteFile, MkdirAll, Mkdir, Rename, Remove, RemoveAll, Create,
//     OpenFile) y, en su propio cuerpo, menciona un literal de cadena «changes» o con
//     «changes/» o «/changes».
//
// QUÉ NO DETECTA (límites conocidos, no garantías):
//
//   - Una ruta bajo changes/ construida sin ese literal en la función que escribe (por ejemplo,
//     recibida por parámetro): quedaría sin detectar si la escritura no pasa por una de las
//     funciones del punto 1. Las escrituras de la carpeta del cambio que hace este repositorio
//     pasan por ellas o por una función que sí nombra «changes».
//   - Escrituras con funciones de otros paquetes (por ejemplo, un ayudante de escritura atómica)
//     sin literal «changes», y llamadas por reflexión, por interfaz o desde un literal de función
//     asignado a una variable de paquete (el literal se atribuye a la función que lo contiene).
//   - Que la guarda se llame antes de escribir en todos los caminos: la prueba comprueba que el
//     cuerpo de la función la nombra, no el flujo de control; eso lo prueban las pruebas de cada
//     escritor (sdd_writer_guard_test.go, service_branch_test.go).
//   - Los ficheros _test.go y el resto de paquetes (internal/kickoff, internal/handoff y similares
//     son quienes escriben, y por diseño no conocen el cambio ni el contexto: la guarda vive en
//     quien los llama).

// writerRoots son los árboles de producción que se inventarían, relativos a la raíz del repositorio.
var writerRoots = []string{"cmd/axiom", "internal/cli", "internal/dashboard", "internal/tui", "internal/sddstatus"}

const inventoryModule = "github.com/IGutierrezZ/axiom/v3/"

// writerCalls son las funciones de escritura que el inventario sigue por selector.
var writerCalls = []struct{ pkgPath, name string }{
	{"internal/kickoff", "Seal"},
	{"internal/kickoff", "AppendGate"},
	{"internal/handoff", "WriteFile"},
	{"internal/sddstatus", "PrepareChangeInstanceConsent"},
	// integration.yaml y los documentos del baseline: los escribe specbaseline, no una primitiva de
	// os visible desde el escritor, así que se siguen por selector (ampliación de S8.19).
	{"internal/specbaseline", "Prepare"},
	{"internal/specbaseline", "Verify"},
	{"internal/specbaseline", "ApplyCompose"},
}

// osWritePrimitives son las primitivas del paquete os que el inventario cuenta como escritura.
var osWritePrimitives = map[string]bool{
	"WriteFile": true, "MkdirAll": true, "Mkdir": true, "Rename": true,
	"Remove": true, "RemoveAll": true, "Create": true, "OpenFile": true,
}

// guardCalls son las llamadas que cuentan como guarda A en el cuerpo de una función guardada.
var guardCalls = map[string]bool{
	"guardChangeWriter":      true, // internal/cli, contexto de la tabla del diseño §2.11
	"GuardChangeWriter":      true, // exportada para cmd/axiom y el dashboard
	"guardComposeDelta":      true, // archive-compose con delta bajo changes/<c>/
	"guardComposeLooseDelta": true,
	"guardIncrementFolder":   true, // dashboard: carpeta de un cambio existente
	"RequireChangeBranch":    true, // el motor, directamente
	"applySealBranchPolicy":  true, // kickoff seal: guarda y después instantánea
}

type writerStatus string

const (
	// statusGuarded: el cuerpo de la función llama a una guarda de guardCalls.
	statusGuarded writerStatus = "guarded"
	// statusVia: la guarda la llama la función `via`, que también figura como guardada y nombra
	// a esta en su cuerpo (una sola consulta cubre al escritor y a sus ayudantes).
	statusVia writerStatus = "via"
	// statusExempt: exento por diseño; el motivo es obligatorio.
	statusExempt writerStatus = "exempt"
)

type writerEntry struct {
	status writerStatus
	via    string // clave de la función que guarda, solo con statusVia
	reason string // motivo, obligatorio con statusExempt; recordatorio en el resto
}

// guardedWriters es la tabla de escritores guardados o exentos. La clave es
// «<directorio>.<función>» o «<directorio>.<Tipo>.<método>».
var guardedWriters = map[string]writerEntry{
	// --- CLI: la guarda A con el contexto de la tabla del diseño §2.11 (S8-c1 y S8-c3) ---
	"internal/cli.runSDDKickoffSeal":              {status: statusGuarded, reason: "kickoff seal: guarda A y después la instantánea (applySealBranchPolicy); `--infer` y el kickoff sin instantánea quedan exentos dentro de la guarda (R15)"},
	"internal/cli.runSDDGateRecord":               {status: statusGuarded, reason: "gate record: contexto ordinary, archive con --gate integration (G-10); cubre AppendGate y el handoff de integración"},
	"internal/cli.writeIntegrationHandoff":        {status: statusVia, via: "internal/cli.runSDDGateRecord", reason: "relevo de integración de la última compuerta de rol: lo cubre la guarda de gate record"},
	"internal/cli.RunSDDContinue":                 {status: statusGuarded, reason: "marcador de instancia: guarda solo si el marcador se va a escribir (ChangeInstanceMarkerPending)"},
	"internal/cli.runSDDIntegratePrepare":         {status: statusGuarded, reason: "integration.yaml: contexto archive"},
	"internal/cli.runSDDIntegrateVerify":          {status: statusGuarded, reason: "integration.yaml: contexto archive"},
	"internal/cli.runSDDArchiveComposeInPlace":    {status: statusGuarded, reason: "archive-compose --change --in-place: contexto archive"},
	"internal/cli.runSDDArchiveComposeWithStderr": {status: statusGuarded, reason: "archive-compose con --output a fichero (canonical/document sobre un delta bajo changes/<c>/): guarda archive; la previsualización a stdout no escribe y no se guarda"},
	// --- cmd/axiom ---
	"cmd/axiom.runHandoffCreate": {status: statusGuarded, reason: "handoff create: contexto ordinary; los sumideros cumulative-<rol> quedan exentos por nombre dentro de la guarda"},
	// --- dashboard (S8-c4) ---
	"internal/dashboard.Service.writeIncrement":                     {status: statusGuarded, reason: "carpeta de un cambio nuevo: guarda A tras ensure"},
	"internal/dashboard.Service.CreateHandoff":                      {status: statusGuarded, reason: "handoff.md de un cambio existente"},
	"internal/dashboard.Service.MigrateIncompleteTasksToCumulative": {status: statusGuarded, reason: "guarda el cambio de origen activo; el destino cumulative-<rol> es un sumidero exento por nombre"},
}

// writerSite es una función de producción que escribe en la carpeta de un cambio, con lo que la
// prueba necesita saber de su cuerpo.
type writerSite struct {
	key     string
	reasons []string        // por qué se considera escritora
	callees map[string]bool // nombres de las funciones que llama (identificador o selector)
}

// scanWriters devuelve los escritores de un fichero fuente de producción. dir es el directorio
// del paquete relativo a la raíz del repositorio; filename y src permiten analizar fuentes en
// memoria (la prueba sintética) y en disco con el mismo código.
func scanWriters(t *testing.T, dir, filename string, src any) []writerSite {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("analizar %s: %v", filename, err)
	}
	// Nombre local de cada importación → su ruta relativa al módulo, para casar por ruta y no por alias.
	imports := map[string]string{}
	for _, spec := range file.Imports {
		importPath, _ := strconv.Unquote(spec.Path.Value)
		rel := strings.TrimPrefix(importPath, inventoryModule)
		local := path.Base(importPath)
		if spec.Name != nil {
			local = spec.Name.Name
		}
		imports[local] = rel
	}
	ownPackage := dir // el paquete de este fichero, para la llamada sin cualificar

	var sites []writerSite
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		site := writerSite{key: dir + "." + functionName(function), callees: map[string]bool{}}
		var osWrite, mentionsChanges bool
		ast.Inspect(function.Body, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.BasicLit:
				if n.Kind == token.STRING {
					value, _ := strconv.Unquote(n.Value)
					if value == "changes" || strings.Contains(value, "changes/") || strings.Contains(value, "/changes") {
						mentionsChanges = true
					}
				}
			case *ast.CallExpr:
				switch callee := n.Fun.(type) {
				case *ast.SelectorExpr:
					site.callees[callee.Sel.Name] = true
					if pkg, ok := callee.X.(*ast.Ident); ok {
						if imports[pkg.Name] == "os" && osWritePrimitives[callee.Sel.Name] {
							osWrite = true
						}
						for _, call := range writerCalls {
							if imports[pkg.Name] == call.pkgPath && callee.Sel.Name == call.name {
								site.reasons = append(site.reasons, call.pkgPath+"."+call.name)
							}
						}
					}
				case *ast.Ident:
					site.callees[callee.Name] = true
					for _, call := range writerCalls {
						if ownPackage == call.pkgPath && callee.Name == call.name {
							site.reasons = append(site.reasons, call.pkgPath+"."+call.name)
						}
					}
				}
			}
			return true
		})
		if osWrite && mentionsChanges {
			site.reasons = append(site.reasons, "escritura de os bajo un literal changes/")
		}
		if len(site.reasons) > 0 {
			sites = append(sites, site)
		}
	}
	return sites
}

// functionName es el nombre de una función o «Tipo.método» para un método.
func functionName(function *ast.FuncDecl) string {
	if function.Recv == nil || len(function.Recv.List) == 0 {
		return function.Name.Name
	}
	expression := function.Recv.List[0].Type
	for {
		switch e := expression.(type) {
		case *ast.StarExpr:
			expression = e.X
			continue
		case *ast.IndexExpr:
			expression = e.X
			continue
		case *ast.IndexListExpr:
			expression = e.X
			continue
		case *ast.Ident:
			return e.Name + "." + function.Name.Name
		}
		return "?." + function.Name.Name
	}
}

// checkInventory contrasta los escritores hallados con la tabla y devuelve cada incumplimiento.
// Es pura: la prueba sobre el código real y la sintética comparten esta comprobación.
func checkInventory(sites []writerSite, table map[string]writerEntry) []string {
	found := make(map[string]writerSite, len(sites))
	for _, site := range sites {
		found[site.key] = site
	}
	var problems []string
	for _, site := range sites {
		entry, listed := table[site.key]
		if !listed {
			problems = append(problems, fmt.Sprintf("%s escribe en la carpeta de un cambio (%s) y no figura en la tabla de escritores guardados: guárdala con la guarda A o añádela como exenta con su motivo", site.key, strings.Join(site.reasons, ", ")))
			continue
		}
		switch entry.status {
		case statusGuarded:
			if !callsGuard(site) {
				problems = append(problems, fmt.Sprintf("%s figura como guardada pero su cuerpo no llama a ninguna guarda (%s)", site.key, guardNames()))
			}
		case statusVia:
			cover, covered := found[entry.via]
			switch {
			case entry.via == "":
				problems = append(problems, fmt.Sprintf("%s figura como cubierta por otra función sin nombrarla", site.key))
			case !covered:
				problems = append(problems, fmt.Sprintf("%s figura como cubierta por %s, que no es un escritor hallado en el código", site.key, entry.via))
			case covered && !callsGuard(cover):
				problems = append(problems, fmt.Sprintf("%s figura como cubierta por %s, pero esta no llama a ninguna guarda", site.key, entry.via))
			case covered && !cover.callees[simpleName(site.key)]:
				problems = append(problems, fmt.Sprintf("%s figura como cubierta por %s, pero esta no la llama", site.key, entry.via))
			}
		case statusExempt:
			if strings.TrimSpace(entry.reason) == "" {
				problems = append(problems, fmt.Sprintf("%s está exenta sin motivo: toda exención declara por qué", site.key))
			}
		default:
			problems = append(problems, fmt.Sprintf("%s tiene un estado desconocido %q en la tabla", site.key, entry.status))
		}
	}
	return problems
}

func callsGuard(site writerSite) bool {
	for name := range guardCalls {
		if site.callees[name] {
			return true
		}
	}
	return false
}

func guardNames() string {
	names := make([]string, 0, len(guardCalls))
	for name := range guardCalls {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// simpleName es el último segmento de una clave: la función, sin directorio ni tipo.
func simpleName(key string) string {
	return key[strings.LastIndex(key, ".")+1:]
}

// repositoryRoot sube desde el directorio de la prueba hasta el go.mod del módulo.
func repositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("directorio de trabajo: %v", err)
	}
	for current := dir; ; {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatalf("no se encontró go.mod subiendo desde %s", dir)
		}
		current = parent
	}
}

// productionWriters recorre los árboles de writerRoots (sin _test.go) y devuelve sus escritores.
func productionWriters(t *testing.T) []writerSite {
	t.Helper()
	root := repositoryRoot(t)
	var sites []writerSite
	for _, tree := range writerRoots {
		walkRoot := filepath.Join(root, filepath.FromSlash(tree))
		err := filepath.WalkDir(walkRoot, func(file string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") {
				return nil
			}
			relative, relErr := filepath.Rel(root, filepath.Dir(file))
			if relErr != nil {
				return relErr
			}
			sites = append(sites, scanWriters(t, filepath.ToSlash(relative), file, nil)...)
			return nil
		})
		if err != nil {
			t.Fatalf("recorrer %s: %v", tree, err)
		}
	}
	return sites
}

// TestWriterInventoryEveryWriterIsGuardedOrExempt: el código de producción de hoy cumple la
// tabla, y cada entrada de la tabla sigue correspondiendo a un escritor real (sin entradas
// obsoletas).
func TestWriterInventoryEveryWriterIsGuardedOrExempt(t *testing.T) {
	sites := productionWriters(t)
	if len(sites) == 0 {
		t.Fatal("el inventario no encontró ningún escritor: el análisis está roto")
	}
	problems := checkInventory(sites, guardedWriters)
	found := map[string]bool{}
	for _, site := range sites {
		found[site.key] = true
	}
	for key := range guardedWriters {
		if !found[key] {
			problems = append(problems, fmt.Sprintf("la tabla lista %s, que ya no es un escritor de la carpeta de un cambio: retira la entrada", key))
		}
	}
	sort.Strings(problems)
	for _, problem := range problems {
		t.Error(problem)
	}
}

// Fuente sintética: cada función es un caso del análisis.
const syntheticWriters = `package cli

import (
	"os"
	"path/filepath"

	kick "github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/handoff"
)

func newSealer(root string) { kick.Seal(root, nil) }

func newHandoffWriter(root string) { handoff.WriteFile(root, nil) }

func newFolderWriter(home string) {
	os.MkdirAll(filepath.Join(home, "openspec", "changes", "x"), 0o755)
}

func guardedSealer(root string) {
	guardChangeWriter(root, "x", "op", 0)
	kick.Seal(root, nil)
}

func unrelatedWriter(dir string) { os.WriteFile(filepath.Join(dir, "other.txt"), nil, 0o644) }

func readsChanges(home string) { os.ReadFile(filepath.Join(home, "changes")) }
`

// TestWriterInventoryRejectsAnUnguardedWriter (RED sintético de S8.19): un escritor nuevo sin
// guarda ni exención hace fallar la comprobación, y las entradas de la tabla mal formadas también.
func TestWriterInventoryRejectsAnUnguardedWriter(t *testing.T) {
	sites := scanWriters(t, "internal/cli", "synthetic.go", syntheticWriters)
	keys := map[string]bool{}
	for _, site := range sites {
		keys[site.key] = true
	}
	for _, want := range []string{"internal/cli.newSealer", "internal/cli.newHandoffWriter", "internal/cli.newFolderWriter", "internal/cli.guardedSealer"} {
		if !keys[want] {
			t.Errorf("el análisis no detectó %s como escritor (alias de importación, literal changes/ o guarda)", want)
		}
	}
	for _, notWant := range []string{"internal/cli.unrelatedWriter", "internal/cli.readsChanges"} {
		if keys[notWant] {
			t.Errorf("el análisis marcó %s como escritor sin serlo (escritura fuera de changes/ o lectura)", notWant)
		}
	}

	problems := checkInventory(sites, map[string]writerEntry{
		"internal/cli.guardedSealer": {status: statusGuarded},
	})
	if len(problems) != 3 {
		t.Fatalf("con solo guardedSealer en la tabla se esperaban 3 incumplimientos (newSealer, newHandoffWriter y newFolderWriter sin guarda), hay %d: %v", len(problems), problems)
	}
	for _, problem := range problems {
		if !strings.Contains(problem, "no figura en la tabla") {
			t.Errorf("incumplimiento inesperado: %s", problem)
		}
	}

	table := map[string]writerEntry{
		"internal/cli.newSealer":        {status: statusGuarded},                        // dice guardada y no llama a ninguna guarda
		"internal/cli.newHandoffWriter": {status: statusExempt},                         // exenta sin motivo
		"internal/cli.newFolderWriter":  {status: statusVia, via: "internal/cli.nadie"}, // cubierta por quien no existe
		"internal/cli.guardedSealer":    {status: statusGuarded},
	}
	problems = checkInventory(sites, table)
	if len(problems) != 3 {
		t.Fatalf("se esperaban 3 incumplimientos de entradas mal formadas, hay %d: %v", len(problems), problems)
	}
	table["internal/cli.newSealer"] = writerEntry{status: statusExempt, reason: "sumidero de prueba"}
	table["internal/cli.newHandoffWriter"] = writerEntry{status: statusExempt, reason: "sumidero de prueba"}
	table["internal/cli.newFolderWriter"] = writerEntry{status: statusVia, via: "internal/cli.guardedSealer"}
	if problems = checkInventory(sites, table); len(problems) != 1 || !strings.Contains(problems[0], "no la llama") {
		t.Fatalf("newFolderWriter cubierta por una función que no la llama debe ser el único incumplimiento: %v", problems)
	}
}
