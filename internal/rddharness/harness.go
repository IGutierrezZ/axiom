// Package rddharness decide si este binario admite la revisión formal (RDD).
//
// RDD está suspendido (C-01-c) hasta su retirada: sin la etiqueta de
// compilación rdd_harness el modo efectivo es siempre off, no se puede activar
// y las preferencias "on" guardadas no se aplican. Solo un binario compilado
// con -tags rdd_harness (bench, e2e y CI) o un test en proceso que active el
// punto de inyección admite RDD. No hay bandera de CLI ni variable de entorno
// que lo encienda.
package rddharness

import "sync/atomic"

var injected atomic.Bool

// Enabled informa si RDD está disponible: la etiqueta rdd_harness compilada o
// el punto de inyección de tests activo.
func Enabled() bool {
	return compiled || injected.Load()
}

// SetForTesting fija el punto de inyección y devuelve la función que restaura
// el valor anterior. Solo debe llamarlo código de tests: el TestMain de un
// paquete que ejercita RDD en proceso lo activa, y los tests del apagado lo
// desactivan de forma local.
func SetForTesting(on bool) (restore func()) {
	previous := injected.Swap(on)
	return func() { injected.Store(previous) }
}

// UnavailableMessage es el motivo que se muestra cuando se intenta activar RDD
// sin la etiqueta rdd_harness.
const UnavailableMessage = "RDD no está disponible: la revisión formal está suspendida en esta compilación"
