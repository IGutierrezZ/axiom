package manifest

import "strings"

// EnvVarPrefix es el prefijo de las variables de entorno que alimentan los secretos de integración.
const EnvVarPrefix = "AXIOM_INTEGRATION_"

// EnvVarName devuelve la variable de entorno de un secreto: el prefijo, el identificador del plugin
// en mayúsculas con "-" convertido en "_", un "_" y el nombre del secreto en mayúsculas
// (azure-devops + pat da AXIOM_INTEGRATION_AZURE_DEVOPS_PAT).
func EnvVarName(pluginID, secretName string) string {
	plugin := strings.ToUpper(strings.ReplaceAll(pluginID, "-", "_"))
	return EnvVarPrefix + plugin + "_" + strings.ToUpper(secretName)
}
