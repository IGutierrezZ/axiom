package changestart

import (
	"strings"
	"unicode"
)

// asciiFold es el plegado a ASCII de D-08 y D-19: las letras con signos diacríticos y las ligaduras
// habituales pasan a su forma ASCII. Se declara por grupos para que la tabla se pueda revisar.
var asciiFold = func() map[rune]string {
	groups := []struct{ to, from string }{
		{"a", "àáâãäåāăą"}, {"ae", "æ"}, {"c", "çćĉċč"}, {"d", "ďđ"}, {"e", "èéêëēĕėęě"},
		{"g", "ĝğġģ"}, {"h", "ĥħ"}, {"i", "ìíîïĩīĭįı"}, {"j", "ĵ"}, {"k", "ķ"}, {"l", "ĺļľŀł"},
		{"n", "ñńņň"}, {"o", "òóôõöøōŏő"}, {"oe", "œ"}, {"r", "ŕŗř"}, {"s", "śŝşš"}, {"ss", "ß"},
		{"t", "ţťŧ"}, {"u", "ùúûüũūŭůűų"}, {"w", "ŵ"}, {"y", "ýÿŷ"}, {"z", "źżž"},
	}
	m := map[rune]string{}
	for _, g := range groups {
		for _, r := range g.from {
			m[r] = g.to
		}
	}
	return m
}()

// foldASCII sustituye cada letra de asciiFold por su forma ASCII conservando la distinción entre
// mayúsculas y minúsculas. El resto de caracteres no ASCII queda tal cual: el llamador decide.
func foldASCII(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x80 {
			b.WriteRune(r)
			continue
		}
		folded, ok := asciiFold[unicode.ToLower(r)]
		switch {
		case !ok:
			b.WriteRune(r)
		case unicode.IsUpper(r):
			b.WriteString(strings.ToUpper(folded[:1]) + folded[1:])
		default:
			b.WriteString(folded)
		}
	}
	return b.String()
}
