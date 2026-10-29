package dashboard

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// parseThemeTokens extrae las variables de color de un bloque de tokens.css.
func parseThemeTokens(t *testing.T, css, opener string) map[string]string {
	t.Helper()
	start := strings.Index(css, opener)
	if start < 0 {
		t.Fatalf("tokens.css no contiene el bloque %q", opener)
	}
	rest := css[start:]
	end := strings.Index(rest, "\n}")
	if end < 0 {
		t.Fatalf("el bloque %q no se cierra", opener)
	}
	vars := map[string]string{}
	re := regexp.MustCompile(`(--[a-z0-9-]+):\s*(#[0-9A-Fa-f]{6})\s*;`)
	for _, m := range re.FindAllStringSubmatch(rest[:end], -1) {
		vars[m[1]] = m[2]
	}
	return vars
}

func channel(hex string, from int) float64 {
	v, _ := strconv.ParseUint(hex[from:from+2], 16, 8)
	c := float64(v) / 255
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func luminance(hex string) float64 {
	return 0.2126*channel(hex, 1) + 0.7152*channel(hex, 3) + 0.0722*channel(hex, 5)
}

// contrastRatio es la razón de contraste WCAG entre dos colores #RRGGBB.
func contrastRatio(a, b string) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func TestContrastRatioKnownValues(t *testing.T) {
	if got := contrastRatio("#000000", "#FFFFFF"); math.Abs(got-21) > 0.01 {
		t.Errorf("negro sobre blanco = %.2f, se esperaba 21", got)
	}
	// Valor que motivó el ajuste de --text-3 en claro: #666A72 sobre #E6E6E1 quedaba por debajo de 4,5.
	if got := contrastRatio("#666A72", "#E6E6E1"); got >= 4.5 {
		t.Errorf("#666A72 sobre #E6E6E1 = %.2f; se esperaba el valor por debajo de 4,5 que justificó el ajuste", got)
	}
}

// Todo texto de los tokens cumple al menos 4,5:1 sobre las superficies donde se usa,
// en ambos temas (criterio de aceptación del rediseño).
func TestThemeTokensMeetContrast(t *testing.T) {
	css := readAsset(t, "tokens.css")
	themes := map[string]map[string]string{
		"claro":  parseThemeTokens(t, css, ":root[data-theme=\"light\"] {"),
		"oscuro": parseThemeTokens(t, css, ":root[data-theme=\"dark\"] {"),
	}

	surfaces := []string{"--bg", "--side", "--surface", "--surface-2", "--hover"}
	texts := []string{"--text", "--text-2", "--text-3", "--accent-text"}
	pairs := [][2]string{
		{"--on-accent", "--accent"},
		{"--accent-text", "--accent-soft"},
		{"--ok", "--ok-soft"}, {"--warn", "--warn-soft"}, {"--bad", "--bad-soft"},
		{"--ok", "--surface"}, {"--warn", "--surface"}, {"--bad", "--surface"},
		{"--accent", "--bg"},
		// Banners, avisos y chips del flujo de trabajo (T3): texto normal y secundario sobre los fondos suaves.
		{"--surface", "--ok"},
	}
	for _, soft := range []string{"--ok-soft", "--warn-soft", "--bad-soft", "--accent-soft"} {
		pairs = append(pairs, [2]string{"--text", soft}, [2]string{"--text-2", soft})
	}
	for _, fg := range texts {
		for _, bg := range surfaces {
			pairs = append(pairs, [2]string{fg, bg})
		}
	}

	for name, vars := range themes {
		for _, p := range pairs {
			fg, okFg := vars[p[0]]
			bg, okBg := vars[p[1]]
			if !okFg || !okBg {
				t.Fatalf("tema %s: falta el token %s o %s", name, p[0], p[1])
			}
			if got := contrastRatio(fg, bg); got < 4.5 {
				t.Errorf("tema %s: %s (%s) sobre %s (%s) = %.2f:1, mínimo 4,5:1", name, p[0], fg, p[1], bg, got)
			}
		}
	}
}
