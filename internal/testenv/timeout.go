package testenv

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// TimeoutScaleEnv es la variable de entorno que escala los presupuestos de
// tiempo de las pruebas. Es un número real mayor o igual que 1.
const TimeoutScaleEnv = "AXIOM_TEST_TIMEOUT_SCALE"

// defaultScale es el factor cuando la variable no está definida. En Windows
// vale 2 porque ahí se midieron los agotamientos de tiempo con la máquina
// cargada: el primer exec de un binario recién compilado y los procesos git
// en paralelo superan los presupuestos pensados para Linux. En el resto de
// plataformas se mantiene el presupuesto tal cual.
func defaultScale(goos string) float64 {
	if goos == "windows" {
		return 2
	}
	return 1
}

// ParseTimeoutScale interpreta el valor crudo de TimeoutScaleEnv. Un valor
// vacío usa el factor por defecto de la plataforma. Un valor que no es un
// número finito mayor o igual que 1 es un error: nunca se reduce un
// presupuesto ni se ignora en silencio una configuración equivocada.
func ParseTimeoutScale(raw, goos string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultScale(goos), nil
	}
	scale, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s=%q is not a number: %w", TimeoutScaleEnv, raw, err)
	}
	if math.IsNaN(scale) || math.IsInf(scale, 0) || scale < 1 {
		return 0, fmt.Errorf("%s=%q must be a finite number >= 1; a scale below 1 would shrink the test budgets", TimeoutScaleEnv, raw)
	}
	return scale, nil
}

// TimeoutScale lee el factor de escala de los presupuestos de tiempo del
// entorno del proceso.
func TimeoutScale() (float64, error) {
	return ParseTimeoutScale(os.Getenv(TimeoutScaleEnv), runtime.GOOS)
}

// Scaled multiplica el presupuesto d por el factor de escala. El resultado
// nunca es menor que d.
func Scaled(d time.Duration) (time.Duration, error) {
	scale, err := TimeoutScale()
	if err != nil {
		return 0, err
	}
	return time.Duration(math.Round(float64(d) * scale)), nil
}

// MustScaled es Scaled para TestMain y para variables de paquete: una
// configuración inválida aborta el paquete con un mensaje claro en lugar de
// ejecutar las pruebas con un presupuesto que nadie eligió.
func MustScaled(d time.Duration) time.Duration {
	scaled, err := Scaled(d)
	if err != nil {
		panic("testenv: " + err.Error())
	}
	return scaled
}
