package manifest

// AzureDevOpsID es el identificador del plugin de Azure DevOps.
const AzureDevOpsID = "azure-devops"

// AzureDevOps devuelve el manifiesto del plugin azure-devops. Es la única fuente: la usan a la vez
// el catálogo cerrado (que valida axiom.yaml) y el Manifest() del plugin, de modo que no pueden
// divergir. Cada llamada devuelve un valor nuevo.
//
// Declara lo que este adaptador entrega: lectura de work items, descubrimiento, etiquetas y adjuntos. La escritura
// con test /rev existe como operación del cliente, pero ningún evento la invoca todavía, así que no
// hay capacidades de escritura. El evento change.start es el único que escucha: ningún verbo lo
// emite aún.
func AzureDevOps() Manifest {
	return Manifest{
		Schema:       SchemaV1,
		ID:           AzureDevOpsID,
		Title:        "Azure DevOps",
		Capabilities: []Capability{CapItemsRead, CapDiscovery, CapTags, CapAttachments},
		RefAlias:     "ado",
		Events:       []EventType{EventChangeStart},
		Config: []ConfigField{
			{Key: "organization", Type: FieldString, Required: true,
				Description: "Organización de Azure DevOps: la de dev.azure.com/<organización>"},
			{Key: "project", Type: FieldString, Required: true,
				Description: "Proyecto de Azure DevOps"},
			{Key: "team", Type: FieldString, Required: true,
				Description: "Equipo del proyecto (iteraciones y miembros)"},
			{Key: "item_keys", Type: FieldStringList, Format: FormatTypePrefixPairs,
				Default:     []string{"User Story=US", "Bug=BUG"},
				Description: "Tipos de work item admitidos al arrancar un cambio, como Tipo=PREFIJO; el perfil bug solo se da al tipo Bug"},
		},
		Secrets: []SecretSpec{
			{Name: "pat", Required: true,
				Description: "PAT de una sola organización con los scopes vso.work_write (work items e iteraciones), vso.project y vso.code"},
		},
		Identity: IdentitySpec{Source: IdentityPlugin, Secret: "pat"},
	}
}
