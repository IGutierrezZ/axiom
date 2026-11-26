package changestart

import (
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

// LinkedIDs devuelve los identificadores externos que un cambio tiene vinculados en su
// `integrations.yaml`, filtrados por los plugins activos (en orden de declaración): ref es el `id`
// del vínculo `work_item` del primer plugin activo que lo tiene, y taskRefs el `id` del vínculo
// `task` de cada rol (en minúsculas; el primero por orden de plugin y de vínculo gana).
//
// Es la frontera por la que el motor de ramas (C-07-f, `{ref}` y `{task_ref}`) obtiene esos
// valores sin conocer las integraciones: solo recibe cadenas. Solo lee; un `integrations.yaml`
// ilegible devuelve el error de lectura para que el llamador falle cerrado.
func LinkedIDs(workspaceRoot, changeRoot string, active []string) (ref string, taskRefs map[string]string, err error) {
	folder := integrations.Folder{WorkspaceRoot: workspaceRoot, Root: changeRoot}
	workItems, err := integrations.WorkItemLinks(folder)
	if err != nil {
		return "", nil, err
	}
	tasks, err := integrations.TaskLinks(folder)
	if err != nil {
		return "", nil, err
	}
	return firstWorkItemID(workItems, active), taskIDsByRole(tasks, active), nil
}

// firstWorkItemID es el `id` del vínculo `work_item` del primer plugin activo que lo tiene.
func firstWorkItemID(links []integrations.Link, active []string) string {
	for _, plugin := range active {
		for _, link := range links {
			if link.Plugin == plugin && link.ID != "" {
				return link.ID
			}
		}
	}
	return ""
}

// taskIDsByRole es el `id` del vínculo `task` de cada rol (en minúsculas) de los plugins activos.
func taskIDsByRole(links []integrations.Link, active []string) map[string]string {
	var ids map[string]string
	for _, plugin := range active {
		for _, link := range links {
			role := strings.ToLower(link.Role)
			if link.Plugin != plugin || link.ID == "" || role == "" {
				continue
			}
			if ids == nil {
				ids = map[string]string{}
			}
			if _, taken := ids[role]; !taken {
				ids[role] = link.ID
			}
		}
	}
	return ids
}
