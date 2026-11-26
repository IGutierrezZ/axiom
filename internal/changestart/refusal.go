package changestart

// Códigos de rechazo de las fases 1 y 2 y del descarte (D-18). La lógica solo lleva el código y los datos de confianza;
// los textos y las continuaciones viven en internal/cli, que es lo que audita el trinquete de
// rechazos.
const (
	CodeWorkItemReferenceInvalid = "work_item_reference_invalid"
	CodeWorkItemNotFound         = "work_item_not_found"
	CodeWorkItemTypeUnmapped     = "work_item_type_unmapped"
	CodeWorkItemClosed           = "work_item_closed"
	CodeWorkItemLinked           = "work_item_linked"
	CodeTrackerUnavailable       = "tracker_unavailable"
	CodeStagingConflict          = "staging_conflict"
	CodeChangeStartBusy          = "change_start_busy"
	CodeWriteFailed              = "write_failed"
	CodeUsageInvalid             = "usage_invalid"
	CodeChangeNotPrepared        = "change_not_prepared"
	CodeStagingNotFound          = "staging_not_found"
)

// Refusal es un rechazo tipado del arranque: el código, los datos de confianza que el mensaje puede
// nombrar y la causa. Error no imprime la causa: puede contener texto del tracker.
type Refusal struct {
	Code   string
	Fields map[string]string
	Err    error
}

func (r *Refusal) Error() string { return r.Code }
func (r *Refusal) Unwrap() error { return r.Err }

// refuse construye un rechazo con pares clave/valor de confianza.
func refuse(code string, cause error, kv ...string) *Refusal {
	fields := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		fields[kv[i]] = kv[i+1]
	}
	return &Refusal{Code: code, Fields: fields, Err: cause}
}
