package workspace

import (
	"os"
	"path/filepath"
	"strings"
)

// DeclaredArtifactStore lee el almacén de artefactos que declara el proyecto en
// <workspaceRoot>/openspec/config.yaml (o config.yml): la clave `artifact_store`
// o `artifactStore`, con comentarios, comillas y mayúsculas tolerados. Devuelve
// el valor en minúsculas y true solo cuando es uno de los cuatro almacenes
// admitidos (`engram`, `openspec`, `hybrid`, `none`); sin declaración, o con un
// valor desconocido, devuelve false.
//
// Replica a propósito, línea a línea, las reglas de declaredArtifactStore de
// internal/sddstatus/status.go. No se reutiliza esa función porque vive en el
// motor del status, un código que el fork comparte con el upstream y que este
// incremento no toca (R9). La prueba de paridad de internal/sddstatus
// (artifact_store_parity_test.go) ejecuta los dos lectores sobre las mismas
// fixtures y es la red que impide que diverjan.
func DeclaredArtifactStore(workspaceRoot string) (store string, declared bool) {
	for _, name := range []string{"config.yaml", "config.yml"} {
		content, err := os.ReadFile(filepath.Join(workspaceRoot, planningHomeDirName, name))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(content), "\n") {
			trimmed := strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
			if !strings.HasPrefix(trimmed, "artifact_store:") && !strings.HasPrefix(trimmed, "artifactStore:") {
				continue
			}
			value := strings.ToLower(strings.Trim(strings.TrimSpace(strings.SplitN(trimmed, ":", 2)[1]), `"'`))
			switch value {
			case "engram", "openspec", "hybrid", "none":
				return value, true
			}
			return "", false
		}
	}
	return "", false
}
