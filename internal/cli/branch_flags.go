package cli

import (
	"flag"
	"slices"
	"strings"
)

// branchRepeatedFlag es una bandera de texto que puede darse más de una vez (`--repo`).
type branchRepeatedFlag []string

func (values *branchRepeatedFlag) String() string { return strings.Join(*values, ",") }

func (values *branchRepeatedFlag) Set(value string) error {
	*values = append(*values, value)
	return nil
}

// branchBlankFlag devuelve el nombre de la primera de las banderas dadas que se pasó en blanco, o
// la cadena vacía. Una bandera presente pero en blanco (casi siempre una variable de entorno
// vacía) no se confunde con su ausencia: un prefijo, una rama, una base o un repositorio vacíos
// no son una excepción.
func branchBlankFlag(flags *flag.FlagSet, names ...string) string {
	blank := ""
	flags.Visit(func(given *flag.Flag) {
		if blank != "" || !slices.Contains(names, given.Name) {
			return
		}
		if repeated, ok := given.Value.(*branchRepeatedFlag); ok {
			if slices.ContainsFunc(*repeated, func(value string) bool { return strings.TrimSpace(value) == "" }) {
				blank = given.Name
			}
			return
		}
		if strings.TrimSpace(given.Value.String()) == "" {
			blank = given.Name
		}
	})
	return blank
}
