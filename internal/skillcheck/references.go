package skillcheck

import (
	"path"
	"regexp"
	"strings"
)

var (
	fencedBlock = regexp.MustCompile("(?s)(```|~~~).*?(```|~~~)")
	inlineCode  = regexp.MustCompile("`[^`\n]*`")
	// Enlaces Markdown `[texto](destino)` e imágenes; el destino no lleva espacios.
	mdLink = regexp.MustCompile(`!?\[[^\]\n]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	// Rutas sueltas `references/x`, `assets/x`, `scripts/x` con al menos un
	// elemento tras la barra y sin marcadores de posición (`<`, `{`, `*`).
	barePath  = regexp.MustCompile("(?:^|[\\s`(\\[\"'])((?:references|assets|scripts)/[A-Za-z0-9_][A-Za-z0-9_./-]*)")
	urlScheme = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)
)

// missingReferences devuelve las referencias relativas del cuerpo de la skill
// que no existen. Es deliberadamente conservador: ignora URLs, anclas, rutas
// absolutas, bloques de código y marcadores de posición.
func missingReferences(body string, exists func(rel string) bool) []string {
	noFences := fencedBlock.ReplaceAllString(body, "")
	noCode := inlineCode.ReplaceAllString(noFences, "")

	var refs []string
	seen := map[string]bool{}
	add := func(ref string) {
		if ref == "" || seen[ref] {
			return
		}
		seen[ref] = true
		refs = append(refs, ref)
	}

	for _, m := range mdLink.FindAllStringSubmatch(noCode, -1) {
		if ref, ok := normalizeLink(m[1]); ok {
			add(ref)
		}
	}
	for _, m := range barePath.FindAllStringSubmatch(noFences, -1) {
		ref := strings.TrimRight(m[1], ".,:;)/")
		// Sin extensión en el último elemento suele ser prosa (`references/tests/docs`).
		if strings.Contains(ref, "...") || ref == "" || !strings.Contains(path.Base(ref), ".") {
			continue
		}
		add(path.Clean(ref))
	}

	var missing []string
	for _, ref := range refs {
		if !exists(ref) {
			missing = append(missing, ref)
		}
	}
	return missing
}

// normalizeLink filtra los destinos que no son ficheros locales relativos.
func normalizeLink(target string) (string, bool) {
	target = strings.Trim(target, "<>")
	if target == "" || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "/") ||
		strings.HasPrefix(target, "~") || urlScheme.MatchString(target) {
		return "", false
	}
	if i := strings.IndexAny(target, "#?"); i >= 0 {
		target = target[:i]
	}
	if target == "" || strings.ContainsAny(target, "<>{}*$") {
		return "", false
	}
	return path.Clean(target), true
}
