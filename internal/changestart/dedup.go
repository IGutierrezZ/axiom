package changestart

import (
	"net/url"
	"strings"
)

// dedupIgnoredParams son los parámetros que no identifican el recurso (D-26): la versión de la API y
// los que solo cambian el nombre o la disposición de la descarga.
var dedupIgnoredParams = map[string]bool{"api-version": true, "filename": true, "download": true}

// dedupKey normaliza la URL de un recurso para deduplicarlo (D-26): esquema y host en minúsculas,
// puerto por defecto eliminado, ruta desescapada y en minúsculas (Azure DevOps no distingue
// mayúsculas en la ruta de adjuntos), sin fragmento ni los parámetros ignorados, y el resto de
// parámetros ordenados. Una URL que no se puede analizar devuelve false.
func dedupKey(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", false
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	if port := u.Port(); port != "" && !(scheme == "https" && port == "443") && !(scheme == "http" && port == "80") {
		host += ":" + port
	}
	query := url.Values{}
	for name, values := range u.Query() {
		if !dedupIgnoredParams[strings.ToLower(name)] {
			query[name] = values
		}
	}
	return scheme + "://" + host + strings.ToLower(u.Path) + "?" + query.Encode(), true
}
