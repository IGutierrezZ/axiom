package reqid

import "strings"

// Series es una serie de identificadores: el ID sin su número. Es comparable y sirve
// de clave de mapa. `RF-OLS-04` pertenece a la serie `REQ-30.*`; `RF-USU-01`, a `RF-USU-*`.
type Series struct{ prefix, suffix, key string }

// Key devuelve los valores de `<DOM>` y `<n>` unidos por `-` (`30`, `USU`); vacía en
// una serie única (`RF-<m>`).
func (s Series) Key() string { return s.key }

// String devuelve la serie con `*` en el lugar del número: `REQ-30.*`.
func (s Series) String() string { return s.prefix + "*" + s.suffix }

// Compare ordena las series de forma natural: los tramos de cifras se comparan por
// valor, de modo que `REQ-9.*` va antes que `REQ-10.*`.
func (s Series) Compare(o Series) int {
	if c := naturalCompare(s.prefix, o.prefix); c != 0 {
		return c
	}
	if c := naturalCompare(s.suffix, o.suffix); c != 0 {
		return c
	}
	return strings.Compare(s.prefix+"\x00"+s.suffix, o.prefix+"\x00"+o.suffix)
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

// naturalCompare compara dos textos por tramos: los de cifras, por valor (sin ceros
// iniciales, primero la longitud y luego el texto); los demás, byte a byte.
func naturalCompare(a, b string) int {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if isDigit(a[i]) && isDigit(b[j]) {
			ei, ej := i, j
			for ei < len(a) && isDigit(a[ei]) {
				ei++
			}
			for ej < len(b) && isDigit(b[ej]) {
				ej++
			}
			da, db := strings.TrimLeft(a[i:ei], "0"), strings.TrimLeft(b[j:ej], "0")
			if len(da) != len(db) {
				if len(da) < len(db) {
					return -1
				}
				return 1
			}
			if c := strings.Compare(da, db); c != 0 {
				return c
			}
			i, j = ei, ej
			continue
		}
		if a[i] != b[j] {
			if a[i] < b[j] {
				return -1
			}
			return 1
		}
		i++
		j++
	}
	switch {
	case len(a)-i < len(b)-j:
		return -1
	case len(a)-i > len(b)-j:
		return 1
	}
	return 0
}
