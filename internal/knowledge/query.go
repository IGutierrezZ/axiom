package knowledge

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// RunQuery ejecuta una consulta fundamentada bajo la estrategia Spec-First con evidencias de código.
// Por defecto es una operación estrictamente de solo lectura (CQS). Solo muta la spec viva si se indica opts.Enrich = true.
func RunQuery(ctx context.Context, opts QueryOptions) (*QueryResult, error) {
	if opts.WorkspaceRoot == "" {
		opts.WorkspaceRoot = "."
	}
	absWorkspace, err := filepath.Abs(opts.WorkspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("ruta workspace inválida: %w", err)
	}

	if opts.SpecsRoot == "" {
		opts.SpecsRoot = filepath.Join(absWorkspace, "openspec")
	}

	res := &QueryResult{
		Evidences:        make([]EvidenceItem, 0),
		ResolvedFromSpec: false,
		SpecUpdated:      false,
	}

	questionLower := strings.ToLower(opts.Question)
	keywords := extractKeywords(questionLower)

	// 1. Paso Spec-First: comprobar si la respuesta ya vive en openspec/specs/ con validación estricta
	if !opts.ForceDeep && len(keywords) > 0 {
		specMatch, specFile, _ := searchLivingSpecs(opts.SpecsRoot, keywords)
		if specMatch != "" {
			res.ResolvedFromSpec = true
			res.DirectAnswer = specMatch
			res.Evidences = append(res.Evidences, EvidenceItem{
				Source:  "spec",
				File:    filepath.ToSlash(specFile),
				Context: "Resuelto directamente desde la especificación viva canónica.",
			})
			return res, nil
		}
	}

	// 2. Paso de Triangulación Profunda: búsqueda léxica multitérmino con scoring ponderado
	codeEvidences, domainHint := searchCodebase(absWorkspace, keywords)
	res.Evidences = codeEvidences

	if len(codeEvidences) == 0 {
		res.DirectAnswer = fmt.Sprintf("No se encontraron evidencias directas ni en las especificaciones vivas ni en el código para la consulta: %q", opts.Question)
		return res, nil
	}

	// 3. Sintetizar respuesta directa a partir de las evidencias halladas
	var answerSb strings.Builder
	answerSb.WriteString(fmt.Sprintf("De acuerdo con el análisis del código en %s:\n\n", domainHint))
	for _, ev := range codeEvidences {
		answerSb.WriteString(fmt.Sprintf("- En `%s` (línea %s): %s\n", ev.File, ev.Lines, ev.Context))
	}
	res.DirectAnswer = answerSb.String()

	// 4. Auto-enriquecimiento explícito (Solo lectura por defecto; requiere opts.Enrich = true)
	if opts.Enrich && len(codeEvidences) > 0 {
		targetDomain := slugify(domainHint)
		if targetDomain == "" || targetDomain == "." {
			targetDomain = "general"
		}

		specDir := filepath.Join(opts.SpecsRoot, "specs", targetDomain)
		specPath := filepath.Join(specDir, "spec.md")
		_ = os.MkdirAll(specDir, 0755)

		if err := enrichSpecFile(specPath, targetDomain, opts.Question, codeEvidences); err == nil {
			res.SpecUpdated = true
			res.TargetSpecPath = filepath.ToSlash(specPath)
			projectName := filepath.Base(absWorkspace)
			_ = SyncIndex(projectName, opts.SpecsRoot)
		}
	}

	return res, nil
}

var queryStopWords = map[string]bool{
	// Artículos, preposiciones y partículas comunes en español
	"de": true, "la": true, "el": true, "en": true, "y": true, "a": true, "los": true,
	"las": true, "un": true, "una": true, "unos": true, "unas": true, "por": true,
	"para": true, "con": true, "sin": true, "sobre": true, "entre": true, "hacia": true,
	"hasta": true, "desde": true, "durante": true, "mediante": true,
	"del": true, "al": true, "que": true, "como": true, "cómo": true, "cuál": true,
	"cual": true, "cuales": true, "cuáles": true, "quien": true, "quién": true,
	"quienes": true, "quiénes": true, "donde": true, "dónde": true, "cuando": true,
	"cuándo": true, "se": true, "es": true, "son": true, "fue": true, "era": true,
	"ser": true, "estar": true, "está": true, "estan": true, "están": true,
	"este": true, "esta": true, "estos": true, "estas": true, "esto": true,
	"ese": true, "esa": true, "esos": true, "esas": true, "eso": true,
	"aquel": true, "aquella": true, "aquellos": true, "aquellas": true, "aquello": true,
	"pero": true, "o": true, "u": true, "e": true, "ni": true, "si": true, "no": true,
	"mas": true, "más": true, "ya": true, "todo": true, "toda": true, "todos": true,
	"todas": true, "otro": true, "otra": true, "otros": true, "otras": true,
	"cada": true, "mucho": true, "mucha": true, "muchos": true, "muchas": true,
	"poco": true, "poca": true, "pocos": true, "pocas": true, "mismo": true,
	"misma": true, "mismos": true, "mismas": true, "tan": true, "tanto": true,
	"tanta": true, "tantos": true, "tantas": true, "muy": true, "tambien": true,
	"también": true, "favor": true, "¿": true, "?": true, "¡": true, "!": true,

	// Verbos auxiliares e interrogativos genéricos propensos a falsos positivos
	"dar": true, "dame": true, "dime": true, "da": true, "dan": true, "dando": true, "dado": true,
	"hacer": true, "haz": true, "hace": true, "hacen": true, "haciendo": true, "hecho": true,
	"ver": true, "ve": true, "ves": true, "vemos": true, "viendo": true, "visto": true,
	"poder": true, "puede": true, "pueden": true, "puedo": true, "podemos": true,
	"mostrar": true, "muestra": true, "muestran": true, "mostrando": true,
	"listar": true, "lista": true, "listame": true, "lístame": true, "listan": true,
	"obtener": true, "obten": true, "obtiene": true, "obtienen": true,
	"buscar": true, "busca": true, "buscan": true, "buscame": true,
	"saber": true, "sé": true, "sabe": true, "saben": true,
	"conocer": true, "conoce": true, "conocen": true,
	"hay": true, "habia": true, "había": true, "hubo": true,
	"tiene": true, "tienen": true, "tengo": true, "tenemos": true,

	// Stop words comunes en inglés
	"the": true, "in": true, "on": true, "at": true, "to": true, "for": true,
	"of": true, "with": true, "by": true, "from": true, "and": true, "or": true,
	"is": true, "are": true, "was": true, "were": true, "be": true, "been": true,
	"what": true, "how": true, "where": true, "when": true, "which": true, "who": true,
	"show": true, "get": true, "give": true, "do": true, "does": true, "did": true,
	"can": true, "could": true, "would": true, "should": true, "will": true,
}

func extractKeywords(text string) []string {
	words := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '?' || r == '¿' || r == '!' || r == '¡' || r == ',' ||
			r == '.' || r == ':' || r == ';' || r == '(' || r == ')' || r == '[' ||
			r == ']' || r == '{' || r == '}' || r == '"' || r == '\'' || r == '`'
	})

	var result []string
	seen := make(map[string]bool)
	for _, w := range words {
		w = strings.ToLower(strings.TrimSpace(w))
		if len(w) > 2 && !queryStopWords[w] && !seen[w] {
			seen[w] = true
			result = append(result, w)
		}
	}
	return result
}

func isAlphaNum(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// matchesKeyword evalúa la coincidencia de kw en text respetando límites de palabra e identificadores en código
// (espacios, snake_case, delimitadores y transiciones de mayúsculas estilo camelCase / PascalCase).
func matchesKeyword(text, kw string) bool {
	if kw == "" || text == "" {
		return false
	}
	textLower := strings.ToLower(text)
	kwLower := strings.ToLower(kw)
	kwRunes := []rune(kwLower)
	textRunes := []rune(text)
	textLowerRunes := []rune(textLower)
	n := len(kwRunes)
	m := len(textLowerRunes)
	if n > m {
		return false
	}

	for i := 0; i <= m-n; i++ {
		match := true
		for j := 0; j < n; j++ {
			if textLowerRunes[i+j] != kwRunes[j] {
				match = false
				break
			}
		}
		if !match {
			continue
		}

		// Validar límite izquierdo (carácter previo a la posición i)
		leftOk := false
		if i == 0 {
			leftOk = true
		} else {
			prev := textRunes[i-1]
			curr := textRunes[i]
			if !isAlphaNum(prev) {
				leftOk = true
			} else if unicode.IsLower(prev) && unicode.IsUpper(curr) {
				leftOk = true
			} else if unicode.IsUpper(prev) && unicode.IsUpper(curr) && i+1 < m && unicode.IsLower(textRunes[i+1]) {
				leftOk = true
			}
		}

		if !leftOk {
			continue
		}

		// Validar límite derecho (carácter posterior a la posición i+n)
		rightOk := false
		if i+n == m {
			rightOk = true
		} else {
			last := textRunes[i+n-1]
			next := textRunes[i+n]
			if !isAlphaNum(next) {
				rightOk = true
			} else if unicode.IsLower(last) && unicode.IsUpper(next) {
				rightOk = true
			} else if unicode.IsUpper(last) && unicode.IsUpper(next) && i+n+1 < m && unicode.IsLower(textRunes[i+n+1]) {
				rightOk = true
			}
		}

		if rightOk {
			return true
		}
	}
	return false
}

func searchLivingSpecs(specsRoot string, keywords []string) (string, string, string) {
	if len(keywords) == 0 {
		return "", "", ""
	}
	specsDir := filepath.Join(specsRoot, "specs")
	if _, err := os.Stat(specsDir); err != nil {
		return "", "", ""
	}

	domainDirs, err := os.ReadDir(specsDir)
	if err != nil {
		return "", "", ""
	}

	type specCandidate struct {
		text     string
		filePath string
		score    int
	}
	var bestCandidate *specCandidate

	for _, d := range domainDirs {
		if !d.IsDir() {
			continue
		}
		specFile := filepath.Join(specsDir, d.Name(), "spec.md")
		content, err := os.ReadFile(specFile)
		if err != nil {
			continue
		}

		strContent := string(content)
		lines := strings.Split(strContent, "\n")

		var currentSection []string
		var currentTitle string
		recording := false

		checkSection := func() {
			if len(currentSection) == 0 {
				return
			}
			sectionText := strings.Join(currentSection, "\n")
			matchCount := 0
			titleMatches := 0
			for _, kw := range keywords {
				if matchesKeyword(sectionText, kw) {
					matchCount++
				}
				if currentTitle != "" && matchesKeyword(currentTitle, kw) {
					titleMatches++
				}
			}

			// Exigir alta correlación dentro del bloque del requerimiento
			requiredMatches := (len(keywords)*3 + 4) / 5 // ~60%
			if requiredMatches < 1 {
				requiredMatches = 1
			}

			if matchCount >= requiredMatches && (titleMatches > 0 || matchCount == len(keywords)) {
				score := matchCount*10 + titleMatches*15
				if bestCandidate == nil || score > bestCandidate.score {
					bestCandidate = &specCandidate{
						text:     sectionText,
						filePath: specFile,
						score:    score,
					}
				}
			}
		}

		for _, line := range lines {
			if strings.HasPrefix(line, "### Requirement:") {
				checkSection()
				currentSection = []string{line}
				currentTitle = strings.TrimPrefix(line, "### Requirement:")
				recording = true
				continue
			}
			if strings.HasPrefix(line, "## ") && recording {
				checkSection()
				currentSection = nil
				currentTitle = ""
				recording = false
				continue
			}
			if recording {
				currentSection = append(currentSection, line)
				if len(currentSection) > 30 {
					checkSection()
					recording = false
				}
			}
		}
		checkSection()
	}

	if bestCandidate != nil {
		return bestCandidate.text, bestCandidate.filePath, ""
	}

	return "", "", ""
}

type scoredEvidence struct {
	item  EvidenceItem
	score int
}

func searchCodebase(root string, keywords []string) ([]EvidenceItem, string) {
	if len(keywords) == 0 {
		return nil, "general"
	}

	var candidates []scoredEvidence
	domainScoreMap := make(map[string]int)

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			name := filepath.Base(path)
			if ignoredDirs[name] || strings.HasPrefix(name, ".") || name == "axiom-wt" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".go" && ext != ".ts" && ext != ".js" && ext != ".py" && ext != ".cs" && ext != ".rs" && ext != ".java" {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		rel, _ := filepath.Rel(root, path)
		relSlash := filepath.ToSlash(rel)

		parts := strings.Split(relSlash, "/")
		domainCandidate := "general"
		if len(parts) > 1 {
			domainCandidate = parts[0]
			if (domainCandidate == "internal" || domainCandidate == "src" || domainCandidate == "pkg") && len(parts) > 2 {
				domainCandidate = parts[1]
			}
		}

		scanner := bufio.NewScanner(file)
		lineNum := 1
		type lineMatch struct {
			num     int
			text    string
			matched int
		}
		var fileMatches []lineMatch
		fileKeywordsMatched := make(map[string]bool)

		for scanner.Scan() {
			lineText := scanner.Text()
			lineKwCount := 0
			for _, kw := range keywords {
				if matchesKeyword(lineText, kw) {
					lineKwCount++
					fileKeywordsMatched[kw] = true
				}
			}

			if lineKwCount > 0 {
				fileMatches = append(fileMatches, lineMatch{
					num:     lineNum,
					text:    strings.TrimSpace(lineText),
					matched: lineKwCount,
				})
			}
			lineNum++
		}

		if len(fileMatches) == 0 {
			return nil
		}

		fileBreadth := len(fileKeywordsMatched)
		for _, m := range fileMatches {
			score := (m.matched * 10) + (fileBreadth * 15)
			if len(keywords) > 1 && m.matched >= len(keywords) {
				score += 50
			}

			candidates = append(candidates, scoredEvidence{
				item: EvidenceItem{
					Source:  "code",
					File:    relSlash,
					Lines:   fmt.Sprintf("%d", m.num),
					Context: m.text,
				},
				score: score,
			})
			domainScoreMap[domainCandidate] += score
		}

		return nil
	})

	if len(candidates) == 0 {
		return nil, "general"
	}

	// Ordenar evidencias por score descendente
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Limitar a las mejores evidencias
	maxResults := 8
	if len(candidates) < maxResults {
		maxResults = len(candidates)
	}

	highestScore := candidates[0].score
	var result []EvidenceItem
	for i := 0; i < maxResults; i++ {
		if i > 2 && candidates[i].score < highestScore/3 {
			break
		}
		result = append(result, candidates[i].item)
	}

	// Determinar el dominio dominante según puntuación agregada
	bestDomain := "general"
	maxDomainScore := 0
	for d, s := range domainScoreMap {
		if s > maxDomainScore {
			maxDomainScore = s
			bestDomain = d
		}
	}

	return result, bestDomain
}

func enrichSpecFile(specPath, domain, question string, evidences []EvidenceItem) error {
	var sb strings.Builder
	titleDomain := strings.Title(domain)

	if !fileExists(specPath) {
		sb.WriteString(fmt.Sprintf("# Especificación Viva: %s\n\n", titleDomain))
		sb.WriteString(fmt.Sprintf("> **Dominio:** `%s`\n", domain))
		sb.WriteString("> **Estado:** Auto-enriquecida por consulta de conocimiento\n\n---\n\n")
		sb.WriteString("## Requerimientos y Reglas de Negocio\n\n")
	} else {
		existing, err := os.ReadFile(specPath)
		if err == nil {
			sb.WriteString(string(existing))
			if !strings.HasSuffix(string(existing), "\n\n") {
				sb.WriteString("\n\n")
			}
		}
	}

	reqTitle := cleanQuestionTitle(question)
	sb.WriteString(fmt.Sprintf("### Requirement: %s\n\n", reqTitle))
	sb.WriteString(fmt.Sprintf("Regla técnica/funcional descubierta a partir de la consulta %q:\n\n", question))
	for _, ev := range evidences {
		sb.WriteString(fmt.Sprintf("- Evidencia en `%s:%s`: `%s`\n", ev.File, ev.Lines, ev.Context))
	}
	sb.WriteString("\n#### Scenario: Validación empírica de la regla\n")
	sb.WriteString(fmt.Sprintf("- **DADO** el componente `%s`\n", domain))
	sb.WriteString("- **CUANDO** se evalúa la consulta formulada\n")
	sb.WriteString("- **ENTONCES** el comportamiento verificado en código satisface la regla documentada\n\n")

	return os.WriteFile(specPath, []byte(sb.String()), 0644)
}

func cleanQuestionTitle(q string) string {
	q = strings.Trim(q, "¿? ")
	if len(q) > 60 {
		q = q[:60] + "..."
	}
	if q == "" {
		return "Regla Funcional Descubierta"
	}
	return strings.Title(q)
}
