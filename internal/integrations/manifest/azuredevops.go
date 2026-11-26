package manifest

// AzureDevOpsID es el identificador del plugin de Azure DevOps.
const AzureDevOpsID = "azure-devops"

// AzureDevOps devuelve el manifiesto del plugin azure-devops. Es la única fuente: la usan a la vez
// el catálogo cerrado (que valida axiom.yaml) y el Manifest() del plugin, de modo que no pueden
// divergir. Cada llamada devuelve un valor nuevo.
//
// Declara lo que este adaptador entrega: lectura de work items, descubrimiento, etiquetas y
// adjuntos, y las capacidades de escritura del ciclo de vida (crear Tasks, relaciones, estimación,
// horas, transiciones y asignación) con sus claves de configuración. El evento change.start es el
// único que escucha por ahora: los eventos del ciclo de vida se declaran cuando el adaptador los
// entrega.
func AzureDevOps() Manifest {
	return Manifest{
		Schema: SchemaV1,
		ID:     AzureDevOpsID,
		Title:  "Azure DevOps",
		Capabilities: []Capability{
			CapItemsRead, CapDiscovery, CapTags, CapAttachments,
			CapItemsCreate, CapRelations, CapEstimation, CapTime, CapItemsTransition, CapAssignment,
		},
		RefAlias: "ado",
		Events:   []EventType{EventChangeStart},
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
			{Key: "task_state_start", Type: FieldString, Format: FormatNonEmpty, Default: "Active",
				Description: "Estado al que pasa la Task del rol cuando el rol arranca"},
			{Key: "task_state_apply_approved", Type: FieldString, Format: FormatNonEmpty, Default: "Tech Review",
				Description: "Estado al que pasa la Task del rol cuando se aprueba su apply"},
			{Key: "task_state_close", Type: FieldString, Format: FormatNonEmpty,
				Description: "Estado de cierre de la Task al imputar horas en continuous; sin valor, el efectivo de task_state_apply_approved"},
			{Key: "work_item_state_archive", Type: FieldString, Format: FormatNonEmpty,
				Description: "Estado al que pasa el work item de origen al archivar; sin valor, no hay transición"},
			{Key: "req_tag_prefix", Type: FieldString, Format: FormatTagPrefix, Default: "req:",
				Description: "Prefijo de las etiquetas con los IDs de requisito que Axiom mantiene en el work item"},
			{Key: "req_tag_max", Type: FieldInt, Default: 50, Min: intPtr(1), Max: intPtr(100),
				Description: "Máximo de etiquetas de requisito que Axiom escribe en un work item"},
			{Key: "workday_hours", Type: FieldInt, Default: 8, Min: intPtr(1), Max: intPtr(24),
				Description: "Horas de una jornada para proponer las horas imputables"},
			{Key: "workdays", Type: FieldStringList, Format: FormatWeekdaySet, Enum: Weekdays(),
				Default:     []string{"mon", "tue", "wed", "thu", "fri"},
				Description: "Días laborables (mon a sun) para proponer las horas imputables"},
		},
		Secrets: []SecretSpec{
			{Name: "pat", Required: true,
				Description: "PAT de una sola organización con los scopes vso.work_write (work items e iteraciones), vso.project y vso.code"},
		},
		Identity: IdentitySpec{Source: IdentityPlugin, Secret: "pat"},
	}
}

func intPtr(n int) *int { return &n }
