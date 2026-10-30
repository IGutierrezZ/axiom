package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/components/communitytool"
	"github.com/IGutierrezZ/axiom/v3/internal/components/gga"
	"github.com/IGutierrezZ/axiom/v3/internal/components/sdd"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// footprintKind clasifica cada elemento de la huella de máquina según su origen.
type footprintKind string

const (
	// footprintDeclared es lo que Axiom necesita a nivel de usuario siempre: el
	// binario de Axiom, el de Engram y el registro del hub.
	footprintDeclared footprintKind = "declarado"
	// footprintOptional es lo que el usuario eligió de forma explícita (GGA,
	// CodeGraph, los lanzadores de OpenCode en segundo plano).
	footprintOptional footprintKind = "opcional"
	// footprintGlobalLeftover es lo que solo deja una instalación global y, si el
	// usuario trabaja únicamente con workspaces, es un resto que puede retirar.
	footprintGlobalLeftover footprintKind = "resto global"
)

// footprintEntry es un elemento que Axiom tiene a nivel de usuario.
type footprintEntry struct {
	Kind   footprintKind
	Name   string
	Detail string
}

// doctorUserPathEntriesFn lee las entradas del PATH de usuario. Es un seam para
// que las pruebas no dependan del PATH real de la máquina.
var doctorUserPathEntriesFn = func() []string {
	entries, err := userPathEntries(doctorGOOS)
	if err != nil {
		return nil
	}
	return entries
}

// collectMachineFootprint enumera lo que Axiom tiene a nivel de usuario: los tres
// elementos declarados (binario de Axiom, binario de Engram y registro del hub) y,
// si existen, GGA, la copia de skills de compatibilidad, los plugins de OpenCode
// en HOME, CodeGraph global, las entradas de PATH gestionadas y los directorios
// de estado. Marca cuáles son restos de una instalación global: sirve para
// detectarlos antes de pasar a trabajar solo con workspaces. Solo lee.
func collectMachineFootprint(homeDir string, installed state.InstallState) []footprintEntry {
	var entries []footprintEntry
	add := func(kind footprintKind, name, detail string) {
		entries = append(entries, footprintEntry{Kind: kind, Name: name, Detail: detail})
	}

	// optionalKind decide si un componente opcional es una elección vigente o un
	// resto: lo primero solo si el estado persistido lo recoge.
	optionalKind := func(selected bool) footprintKind {
		if selected {
			return footprintOptional
		}
		return footprintGlobalLeftover
	}
	ggaSelected := slices.Contains(installed.Components, model.ComponentGGA)
	codeGraphSelected := slices.Contains(installed.CommunityTools, string(model.CommunityToolCodeGraph))

	for _, tool := range []string{"axiom", "engram"} {
		if resolved, _, err := resolveDoctorTool(tool); err == nil {
			add(footprintDeclared, "binario "+tool, resolved)
		}
	}

	if existing := existingPaths(
		filepath.Join(system.AxiomDir(homeDir), "workspaces.json"),
		system.StatePath(homeDir),
		filepath.Join(system.AxiomDir(homeDir), "backups"),
		system.CacheDir(homeDir),
		system.BinDir(homeDir),
	); len(existing) > 0 {
		add(footprintDeclared, "hub y estado de Axiom", system.AxiomDir(homeDir)+" ("+joinBaseNames(existing)+")")
	}

	if paths := existingPaths(gga.FootprintPaths(homeDir, doctorGOOS)...); len(paths) > 0 {
		add(optionalKind(ggaSelected), "GGA", strings.Join(paths, ", "))
	}

	if skillDir := filepath.Join(homeDir, ".agents", "skills"); isDir(skillDir) {
		add(footprintGlobalLeftover, "~/.agents/skills", skillDir+" (copia de skills de compatibilidad)")
	}

	if plugins := openCodeUserPlugins(homeDir); len(plugins) > 0 {
		add(footprintGlobalLeftover, "plugins de OpenCode en HOME", strings.Join(plugins, ", "))
	}

	if files := communitytool.ManagedCodeGraphGuidanceFiles(homeDir); len(files) > 0 {
		add(optionalKind(codeGraphSelected), "CodeGraph global", strings.Join(files, ", "))
	}

	entries = append(entries, managedPathFootprint(homeDir, ggaSelected)...)

	if legacyDir := filepath.Join(homeDir, ".gentle-ai"); isDir(legacyDir) {
		add(footprintGlobalLeftover, "~/.gentle-ai", legacyDir+" (estado heredado de versiones anteriores)")
	}
	return entries
}

// managedPathFootprint lista las entradas del PATH de usuario que Axiom añade o
// de las que depende: el directorio del binario de Engram, `~/bin` cuando GGA
// dejó ahí sus shims y `~/.axiom/bin`, donde viven los lanzadores gestionados.
func managedPathFootprint(homeDir string, ggaSelected bool) []footprintEntry {
	pathEntries := doctorUserPathEntriesFn()
	if len(pathEntries) == 0 {
		return nil
	}
	var entries []footprintEntry
	for _, dir := range engramBinaryDirsOnPath(pathEntries, doctorGOOS) {
		entries = append(entries, footprintEntry{Kind: footprintDeclared, Name: "PATH: binario de Engram", Detail: dir})
	}
	ggaKind := footprintGlobalLeftover
	if ggaSelected {
		ggaKind = footprintOptional
	}
	for _, dir := range gga.FootprintPathEntries(homeDir, doctorGOOS) {
		if pathEntriesContainDir(pathEntries, dir) {
			entries = append(entries, footprintEntry{Kind: ggaKind, Name: "PATH: shims de GGA", Detail: dir})
		}
	}
	if binDir := system.BinDir(homeDir); pathEntriesContainDir(pathEntries, binDir) {
		entries = append(entries, footprintEntry{Kind: footprintOptional, Name: "PATH: lanzadores de Axiom", Detail: binDir})
	}
	return entries
}

// openCodeUserPlugins devuelve los plugins gestionados por Axiom que existen en
// la carpeta de usuario de OpenCode y de Kilo. Con el ámbito workspace SDD los
// escribe en el proyecto: los de HOME son de una instalación global.
func openCodeUserPlugins(homeDir string) []string {
	var found []string
	for _, id := range []model.AgentID{model.AgentOpenCode, model.AgentKilocode} {
		adapter, err := agents.NewAdapter(id)
		if err != nil {
			continue
		}
		pluginsDir := filepath.Join(adapter.GlobalConfigDir(homeDir), "plugins")
		for _, name := range sdd.OpenCodePluginLifecycleNames(id) {
			path := filepath.Join(pluginsDir, name)
			if info, err := os.Lstat(path); err == nil && !info.IsDir() && !slices.Contains(found, path) {
				found = append(found, path)
			}
		}
	}
	return found
}

func existingPaths(paths ...string) []string {
	var existing []string
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			existing = append(existing, path)
		}
	}
	return existing
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func joinBaseNames(paths []string) string {
	names := make([]string, len(paths))
	for i, path := range paths {
		names[i] = filepath.Base(path)
	}
	return strings.Join(names, ", ")
}

// renderMachineFootprint escribe el apartado de huella de máquina del doctor. Es
// informativo: no cuenta como aviso ni como fallo, porque un resto global no
// estropea el equipo, solo condiciona el trabajo con workspaces.
func renderMachineFootprint(w io.Writer, entries []footprintEntry) {
	fmt.Fprintln(w, "Huella de máquina (nivel de usuario)")
	fmt.Fprintln(w, "-------------------------------------")
	if len(entries) == 0 {
		fmt.Fprintln(w, "  Axiom no tiene nada a nivel de usuario.")
		fmt.Fprintln(w)
		return
	}
	leftovers := 0
	for _, entry := range entries {
		fmt.Fprintf(w, "  %-14s %-30s %s\n", "["+string(entry.Kind)+"]", entry.Name, entry.Detail)
		if entry.Kind == footprintGlobalLeftover {
			leftovers++
		}
	}
	if leftovers > 0 {
		fmt.Fprintf(w, "  %d elemento(s) propios de una instalación global: si trabajas solo con workspaces, son restos que puedes retirar (`axiom uninstall` o a mano).\n", leftovers)
	}
	fmt.Fprintln(w)
}

// doctorGGARequired indica si GGA debe figurar entre las herramientas exigidas:
// solo cuando el usuario lo eligió de forma explícita. Con GGA fuera de los
// presets su ausencia no es un fallo.
func doctorGGARequired(installed state.InstallState) bool {
	return slices.Contains(installed.Components, model.ComponentGGA)
}
