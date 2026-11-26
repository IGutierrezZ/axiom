package branch

// ChangeBranchSource es una fuente con nombre del nombre de la rama del cambio: el origen se
// documenta en quien llama (`--branch`, instantánea del kickoff, registro, ...) y Name vacío
// significa que la fuente no tiene valor.
type ChangeBranchSource struct {
	Origin string
	Name   string
}

// NamedBranch construye una fuente con nombre.
func NamedBranch(origin, name string) ChangeBranchSource {
	return ChangeBranchSource{Origin: origin, Name: name}
}

// ResolveChangeBranch es el único sitio que compone el nombre de la rama del cambio (design D-14).
// Devuelve el nombre de la primera fuente con valor, en el orden en que cada consumidor las
// entrega, y si ninguna lo tiene, la composición `<prefijo>/<cambio>`. Sin fuentes con valor y
// sin prefijo devuelve "": no hay rama que resolver.
func ResolveChangeBranch(change, prefix string, sources ...ChangeBranchSource) string {
	for _, source := range sources {
		if source.Name != "" {
			return source.Name
		}
	}
	if prefix == "" {
		return ""
	}
	return prefix + "/" + change
}

// ArchiveBranchFor es el único sitio que añade `-archive` a la rama del cambio ya resuelta
// (design D-14).
func ArchiveBranchFor(changeBranch string) string {
	return changeBranch + archiveSuffix
}
