package adoption

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	// maxFiles es el número máximo de ficheros candidatos (por extensión) que Scan examina.
	maxFiles = 2000
	// maxFileSize es el tamaño máximo de un fichero candidato (2 MiB).
	maxFileSize = 2 << 20
)

var (
	// excludedDirs son los directorios que Scan nunca recorre, a cualquier profundidad;
	// tampoco recorre ningún directorio oculto.
	excludedDirs = map[string]bool{
		"openspec": true, ".openspec": true, "odd": true, ".git": true,
		"node_modules": true, "vendor": true, "testdata": true,
	}
	sourceNamePattern = regexp.MustCompile(`(?i)requisit|requirement|prd|srs|especificaci|specification|historias|user-?stor|casos?-de-uso|use-?case`)
	// toolingTextNames son ficheros .txt de herramientas (pip, CMake…) que nunca son un
	// origen aunque su nombre case con el patrón de requisitos.
	toolingTextNames = regexp.MustCompile(`(?i)^(requirements([-_.].*)?|constraints|cmakelists|robots|licen[cs]e|copying|manifest|version)\.txt$`)
	foreignIDPattern = regexp.MustCompile(`\b(?:RF|RNF|RN|REQ|US|HU|UC|CU|FR|NFR)[-_](?:[A-Z]{2,5}[-_])?\d{1,5}\b`)
	atxHeading       = regexp.MustCompile(`^ {0,3}(#{1,3})[ \t]+(.*?)(?:[ \t]+#+)?[ \t]*$`)
	fenceLine        = regexp.MustCompile("^ {0,3}(```+|~~~+)")
)

// Scan inventaría los orígenes de requisitos previos bajo root (el workspaceRoot): solo
// lee, es determinista y no entra en openspec/, .openspec/, odd/, .git/, node_modules/,
// vendor/, testdata/ ni en directorios ocultos. No exige que exista openspec/.
func Scan(root string) (Inventory, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Inventory{}, fmt.Errorf("la ruta %q no es válida: %w", root, err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return Inventory{}, fmt.Errorf("no se puede leer el directorio %s: %w", absRoot, err)
	}
	if !info.IsDir() {
		return Inventory{}, fmt.Errorf("%s no es un directorio", absRoot)
	}
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return Inventory{}, fmt.Errorf("no se puede resolver el directorio %s: %w", absRoot, err)
	}

	inv := Inventory{Schema: InventorySchema, Sources: []Source{}, ContextCandidates: []ContextCandidate{}, Warnings: []string{}}
	candidates := 0
	walkErr := filepath.WalkDir(absRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if p == absRoot {
			return nil
		}
		rel, relErr := filepath.Rel(absRoot, p)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		name := d.Name()

		if d.IsDir() {
			if excludedDirs[strings.ToLower(name)] || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			if c, ok := contextCandidate(rel, true); ok {
				inv.ContextCandidates = append(inv.ContextCandidates, c)
			}
			return nil
		}

		readPath := p
		if d.Type()&fs.ModeSymlink != 0 {
			target, ok := ResolveInside(realRoot, p)
			if !ok {
				return nil
			}
			readPath = target
		}
		if c, ok := contextCandidate(rel, false); ok {
			inv.ContextCandidates = append(inv.ContextCandidates, c)
		}

		ext := strings.ToLower(path.Ext(name))
		textual := ext == ".md" || ext == ".markdown" || ext == ".txt"
		binary := ext == ".docx" || ext == ".xlsx" || ext == ".pdf"
		if !textual && !binary {
			return nil
		}
		if ext == ".txt" && toolingTextNames.MatchString(name) {
			return nil
		}
		candidates++
		if candidates > maxFiles {
			inv.Warnings = append(inv.Warnings, fmt.Sprintf("se alcanzó el límite de %d ficheros candidatos: el resto del árbol no se ha inventariado; acota el escaneo con --path a la carpeta de documentación", maxFiles))
			return filepath.SkipAll
		}
		nameMatch := sourceNamePattern.MatchString(name)
		if binary && !nameMatch {
			return nil
		}
		stat, statErr := os.Stat(readPath)
		if statErr != nil || !stat.Mode().IsRegular() {
			return nil
		}
		if stat.Size() > maxFileSize {
			if nameMatch || textual {
				inv.Warnings = append(inv.Warnings, fmt.Sprintf("%s supera 2 MiB y no se inventaría; divídelo o migra sus requisitos a mano", rel))
			}
			return nil
		}
		data, readErr := os.ReadFile(readPath)
		if readErr != nil {
			inv.Warnings = append(inv.Warnings, fmt.Sprintf("%s no se pudo leer (%v) y no se inventaría", rel, readErr))
			return nil
		}
		sum := sha256.Sum256(data)
		source := Source{Path: rel, SHA256: hex.EncodeToString(sum[:])}
		if binary {
			source.Kind = KindFile
			source.Units = []Unit{{Anchor: "", Title: name, Line: 1}}
			inv.Sources = append(inv.Sources, source)
			return nil
		}
		text := string(data)
		if !nameMatch && len(foreignIDs(text)) == 0 {
			return nil
		}
		source.Kind = KindMarkdown
		source.Units = Units(text, name)
		inv.Sources = append(inv.Sources, source)
		return nil
	})
	if walkErr != nil {
		return Inventory{}, fmt.Errorf("no se pudo recorrer %s: %w", absRoot, walkErr)
	}
	sort.SliceStable(inv.Sources, func(i, j int) bool { return inv.Sources[i].Path < inv.Sources[j].Path })
	sort.SliceStable(inv.ContextCandidates, func(i, j int) bool { return inv.ContextCandidates[i].Path < inv.ContextCandidates[j].Path })
	return inv, nil
}

// ResolveInside resuelve un enlace simbólico y solo lo acepta si su destino está dentro
// de realRoot (el directorio raíz con sus enlaces ya resueltos).
func ResolveInside(realRoot, link string) (string, bool) {
	target, err := filepath.EvalSymlinks(link)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(realRoot, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", false
	}
	return target, true
}

// contextCandidate dice si una ruta relativa (con `/`) es un candidato a `technical_context`:
// ARCHITECTURE.md y docs/arquitectura*, docs/architecture*, docs/design* como maintained;
// adr/, docs/adr/ y docs/decisions/ como reference. Un directorio lleva `/` final.
func contextCandidate(rel string, isDir bool) (ContextCandidate, bool) {
	lower := strings.ToLower(rel)
	suffix := ""
	if isDir {
		suffix = "/"
		switch lower {
		case "adr", "docs/adr", "docs/decisions":
			return ContextCandidate{Path: rel + suffix, Mode: ModeReference, Reason: "registro de decisiones de arquitectura"}, true
		}
	}
	if !isDir && lower == "architecture.md" {
		return ContextCandidate{Path: rel, Mode: ModeMaintained, Reason: "documento de arquitectura"}, true
	}
	if dir, base := path.Split(lower); dir == "docs/" {
		for _, prefix := range []string{"arquitectura", "architecture", "design"} {
			if strings.HasPrefix(base, prefix) {
				return ContextCandidate{Path: rel + suffix, Mode: ModeMaintained, Reason: "documentación de arquitectura o diseño"}, true
			}
		}
	}
	return ContextCandidate{}, false
}

// foreignIDs devuelve, ordenados y sin repetir, los identificadores ajenos del texto
// (`RF-120`…), descartando los `REQ-n.m` de OpenSpec (los que siguen con `.<dígito>`).
func foreignIDs(text string) []string {
	seen := map[string]bool{}
	var ids []string
	for _, m := range foreignIDPattern.FindAllStringIndex(text, -1) {
		rest := text[m[1]:]
		if len(rest) >= 2 && rest[0] == '.' && rest[1] >= '0' && rest[1] <= '9' {
			continue
		}
		id := text[m[0]:m[1]]
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// Units devuelve las unidades de un documento Markdown: sus encabezados ATX H1–H3 fuera
// de vallas, cada uno con su ancla y los identificadores ajenos de su sección (hasta el
// siguiente encabezado H1–H3). Sin encabezados, el fichero entero es una unidad de ancla
// vacía y título fallbackTitle. Lo comparten Scan y Check, de modo que el ancla de un
// `Origen:` se resuelve con el mismo algoritmo que la generó.
func Units(text, fallbackTitle string) []Unit {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	slugs := newSlugger()
	var units []Unit
	var sections []*strings.Builder
	fence := ""
	for i, line := range lines {
		if m := fenceLine.FindStringSubmatch(line); m != nil {
			marker := m[1][:1]
			switch {
			case fence == "":
				fence = marker
			case fence == marker:
				fence = ""
			}
		} else if fence == "" {
			if m := atxHeading.FindStringSubmatch(line); m != nil && strings.TrimSpace(m[2]) != "" {
				title := strings.TrimSpace(m[2])
				units = append(units, Unit{Anchor: slugs.next(title), Title: title, Line: i + 1})
				sections = append(sections, &strings.Builder{})
			}
		}
		if len(sections) > 0 {
			sections[len(sections)-1].WriteString(line)
			sections[len(sections)-1].WriteByte('\n')
		}
	}
	if len(units) == 0 {
		return []Unit{{Anchor: "", Title: fallbackTitle, Line: 1, ForeignIDs: foreignIDs(text)}}
	}
	for i := range units {
		units[i].ForeignIDs = foreignIDs(sections[i].String())
	}
	return units
}
