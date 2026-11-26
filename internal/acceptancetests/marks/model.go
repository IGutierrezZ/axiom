package marks

// FileName es el nombre del fichero de marcas dentro de la carpeta del cambio.
const FileName = "pruebas-estado.yaml"

// Límites de los campos de texto libre de una marca.
const (
	MaxNoteRunes   = 500
	MaxCommitRunes = 100
)

// Result es el resultado de una marca.
type Result string

// Resultados admitidos.
const (
	ResultOK      Result = "ok"
	ResultKO      Result = "ko"
	ResultBlocked Result = "blocked"
	ResultNA      Result = "n/a"
)

// Valid dice si r pertenece al dominio cerrado de resultados.
func (r Result) Valid() bool {
	switch r {
	case ResultOK, ResultKO, ResultBlocked, ResultNA:
		return true
	}
	return false
}

// Entry es una marca de ejecución. El orden de los campos fija el de la serialización.
// Dos entradas son la misma cuando sus ocho campos son iguales.
type Entry struct {
	Case        string `yaml:"case"`
	Environment string `yaml:"environment"`
	Result      Result `yaml:"result"`
	By          string `yaml:"by"`
	At          string `yaml:"at"`
	Commit      string `yaml:"commit"`
	PlanDigest  string `yaml:"plan_digest"`
	Note        string `yaml:"note"`
}

// AtLayout es el formato de `at`: UTC con milisegundos.
const AtLayout = "2006-01-02T15:04:05.000Z"

// Issue es una entrada del fichero que se ignora por estar fuera de forma.
type Issue struct {
	// Index es la posición (desde 0) del elemento dentro de la secuencia.
	Index  int
	Reason string
}

// File es el resultado de leer `pruebas-estado.yaml`: las entradas válidas, en el orden del
// fichero, y las inválidas aparte.
type File struct {
	Entries []Entry
	Invalid []Issue
}
