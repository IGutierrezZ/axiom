package cli

import (
	"fmt"
	"runtime"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/components/communitytool"
	"github.com/IGutierrezZ/axiom/v3/internal/components/engram"
	"github.com/IGutierrezZ/axiom/v3/internal/components/gga"
	"github.com/IGutierrezZ/axiom/v3/internal/components/opencodeplugin"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// Huella de máquina con el ámbito workspace.
//
// Con `--scope workspace` Axiom no escribe en la carpeta de usuario salvo lo que
// declara: el binario `axiom`, el binario de Engram y el registro del hub. Los
// pasos implícitos que escribían en HOME se omiten (engram setup, la copia de
// skills de compatibilidad, la limpieza de skills retiradas en HOME y el refresco
// de plugins de OpenCode en HOME). Los componentes que el usuario elige de forma
// explícita (GGA, CodeGraph, plugins comunitarios de OpenCode) conservan su
// huella en HOME, que es consentimiento expreso, pero se declara aquí con un
// aviso que enumera las rutas y la modificación del PATH.

// engramSetupSkippedNote avisa de que `engram setup <agente>` no se ha lanzado
// porque, con el ámbito workspace, escribe en la carpeta de usuario del agente.
// El MCP de Engram sí queda en el proyecto, así que el aviso solo ofrece el
// comando manual para quien quiera además el registro a nivel de usuario. Con
// otro ámbito, o sin Engram, o con AXIOM_ENGRAM_SETUP_MODE=off, no hay nada que
// avisar.
func engramSetupSkippedNote(scope InstallScope, components []model.ComponentID, agentIDs []model.AgentID) []string {
	if scope != ScopeWorkspace || !slices.Contains(components, model.ComponentEngram) {
		return nil
	}
	mode := engram.SetupModeFromEnv()
	var slugs []string
	for _, agent := range agentIDs {
		if !engram.ShouldAttemptSetup(mode, agent) {
			continue
		}
		slug, _ := engram.SetupAgentSlug(agent)
		if !slices.Contains(slugs, slug) {
			slugs = append(slugs, slug)
		}
	}
	if len(slugs) == 0 {
		return nil
	}
	commands := make([]string, len(slugs))
	for i, slug := range slugs {
		commands[i] = "engram setup " + slug
	}
	return []string{fmt.Sprintf(
		"Con --scope workspace no se ha ejecutado `engram setup` (%s) porque escribe en tu carpeta de usuario. El MCP de Engram ya está configurado en el proyecto; si el agente no lo encuentra, ejecuta a mano: %s.",
		strings.Join(slugs, ", "), strings.Join(commands, "; "))}
}

// optInFootprintNotes declara la huella en la carpeta de usuario de los
// componentes que el usuario ha elegido de forma explícita y que, con el ámbito
// workspace, siguen escribiendo fuera del proyecto. Con ámbito global la huella
// ya es la esperada y no se avisa.
func optInFootprintNotes(scope InstallScope, homeDir string, components []model.ComponentID, agentIDs []model.AgentID, selection model.Selection) []string {
	if scope != ScopeWorkspace {
		return nil
	}
	var notes []string
	if slices.Contains(components, model.ComponentGGA) {
		paths := gga.FootprintPaths(homeDir, runtime.GOOS)
		note := fmt.Sprintf(
			"GGA se ha elegido de forma explícita y escribe fuera del proyecto, en tu carpeta de usuario: %s, además del binario `gga` que instala su script en ~/bin o ~/.local/bin.",
			strings.Join(paths, ", "))
		if entries := gga.FootprintPathEntries(homeDir, runtime.GOOS); len(entries) > 0 {
			note += fmt.Sprintf(" También modifica el PATH de usuario: añade %s.", strings.Join(entries, ", "))
		}
		notes = append(notes, note+" Es la huella de máquina declarada de GGA; retírala a mano si dejas de usarlo.")
	}
	if selection.HasCommunityTool(model.CommunityToolCodeGraph) {
		note := "CodeGraph se ha elegido de forma explícita y Axiom no lo integra con ubicación de proyecto: se instala como paquete global y su MCP y su guía se escriben en la configuración de usuario de los agentes detectados"
		if paths := communitytool.CodeGraphManagedPaths(homeDir); len(paths) > 0 {
			note += ": " + strings.Join(paths, ", ")
		}
		notes = append(notes, note+". Es la huella de máquina declarada de CodeGraph.")
	}
	if paths := openCodeUserPluginPaths(homeDir, components, agentIDs, selection); len(paths) > 0 {
		notes = append(notes, fmt.Sprintf(
			"Los plugins de OpenCode elegidos de forma explícita se instalan en la configuración de usuario de OpenCode, fuera del proyecto: %s.",
			strings.Join(paths, ", ")))
	}
	return notes
}

// openCodeUserPluginPaths devuelve las rutas de HOME donde se instalan los
// plugins de OpenCode elegidos de forma explícita: los comunitarios de la
// selección y el logo, que es un componente propio. Sin OpenCode entre los
// agentes no se instala ninguno.
func openCodeUserPluginPaths(homeDir string, components []model.ComponentID, agentIDs []model.AgentID, selection model.Selection) []string {
	if !containsAgent(agentIDs, model.AgentOpenCode) {
		return nil
	}
	plugins := slices.Clone(selection.OpenCodePlugins)
	if slices.Contains(components, model.ComponentOpenCodeGentleLogo) && !slices.Contains(plugins, model.OpenCodePluginGentleLogo) {
		plugins = append(plugins, model.OpenCodePluginGentleLogo)
	}
	if len(plugins) == 0 {
		return nil
	}
	paths, err := opencodeplugin.InstallPaths(homeDir, plugins)
	if err != nil {
		return nil
	}
	return paths
}
