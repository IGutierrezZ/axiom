package azuredevops

import "context"

// Este fichero solo se compila con las pruebas: expone el punto de salida HTTP genérico para las
// pruebas externas (azuredevops_test), que no pueden alcanzar lo no exportado y que, al importar
// adotest, no pueden vivir en este paquete (adotest importa azuredevops).

// DoRequest es la petición lógica interna.
type DoRequest = request

// NewReadRequest construye un GET reintentable bajo la organización: segments son los segmentos
// tras la organización, sin escapar.
func NewReadRequest(scope, operation string, workItemID int, segments ...string) DoRequest {
	return request{method: "GET", segments: segments, scope: scope, operation: operation, retryable: true, workItemID: workItemID}
}

// Do ejecuta una petición lógica por el único punto de salida HTTP.
func (c *Client) Do(ctx context.Context, r DoRequest, out any) error { return c.do(ctx, r, out) }
