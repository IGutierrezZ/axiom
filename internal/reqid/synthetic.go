package reqid

import (
	"fmt"
	"strings"
)

// Synthetic devuelve el identificador informativo de livingdoc para un requisito que no
// lleva ninguno. Nunca es válido para ningún patrón (RF-RID-06): ningún identificador
// válido contiene minúsculas (la gramática de la plantilla solo admite mayúsculas,
// cifras y separadores) y todo sintético contiene al menos una. Cuando el dominio ya
// tiene una minúscula el prefijo es `REQ-`; si no, `req-`.
func Synthetic(domain string, ordinal int) string {
	prefix := "req-"
	if strings.ContainsAny(domain, "abcdefghijklmnopqrstuvwxyz") {
		prefix = "REQ-"
	}
	return fmt.Sprintf("%s%s-%d", prefix, domain, ordinal)
}
