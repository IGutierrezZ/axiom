package livespec

import "sort"

// Lookup devuelve todas las apariciones de un identificador escrito exactamente así, en un
// orden estable. Es un auxiliar de las pruebas: la producción compara por identidad (serie y
// número) con LookupIdentity, de modo que `RF-1` y `RF-001` con `RF-<m>` son el mismo.
func (r *Registry) Lookup(id string) []IDRef {
	refs := append([]IDRef(nil), r.byID[id]...)
	sort.SliceStable(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
	return refs
}
