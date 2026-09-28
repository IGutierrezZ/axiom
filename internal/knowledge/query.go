package knowledge

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RunQuery ejecuta una consulta fundamentada bajo la estrategia Spec-First con auto-enriquecimiento.
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

	// 1. Paso Spec-First: comprobar si la respuesta ya vive en openspec/specs/
	if !opts.ForceDeep {
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

	// 2. Paso de Triangulación Profunda: investigar en el código fuente
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

	// 4. Auto-enriquecimiento obligatorio de la Spec Viva
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

	return res, nil
}

func extractKeywords(text string) []string {
	stopWords := map[string]bool{
		"de": true, "la": true, "el": true, "en": true, "y": true, "a": true, "los": true,
		"las": true, "un": true, "una": true, "por": true, "para": true, "con": true,
		"que": true, "como": true, "cómo": true, "cuál": true, "cuales": true, "se": true,
		"es": true, "son": true, "del": true, "al": true, "¿": true, "?": true, "sobre": true,
	}

	words := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '?' || r == '¿' || r == ',' || r == '.' || r == ':' || r == ';' || r == '(' || r == ')'
	})

	var result []string
	for _, w := range words {
		w = strings.ToLower(strings.TrimSpace(w))
		if len(w) > 2 && !stopWords[w] {
			result = append(result, w)
		}
	}
	return result
}

func searchLivingSpecs(specsRoot string, keywords []string) (string, string, string) {
	specsDir := filepath.Join(specsRoot, "specs")
	if _, err := os.Stat(specsDir); err != nil {
		return "", "", ""
	}

	domainDirs, err := os.ReadDir(specsDir)
	if err != nil {
		return "", "", ""
	}

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
		matchCount := 0
		for _, kw := range keywords {
			if strings.Contains(strings.ToLower(strContent), kw) {
				matchCount++
			}
		}

		// Si coincide una proporción significativa de palabras clave
		if len(keywords) > 0 && matchCount >= (len(keywords)+1)/2 {
			lines := strings.Split(strContent, "\n")
			var relevantLines []string
			recording := false
			for _, line := range lines {
				if strings.HasPrefix(line, "### Requirement:") {
					recording = false
					for _, kw := range keywords {
						if strings.Contains(strings.ToLower(line), kw) {
							recording = true
							break
						}
					}
				}
				if recording {
					relevantLines = append(relevantLines, line)
					if len(relevantLines) > 15 {
						break
					}
				}
			}

			if len(relevantLines) > 0 {
				return strings.Join(relevantLines, "\n"), specFile, strContent
			}
			return fmt.Sprintf("La especificación viva '%s' contiene la definición aplicable para este dominio.", d.Name()), specFile, strContent
		}
	}

	return "", "", ""
}

func searchCodebase(root string, keywords []string) ([]EvidenceItem, string) {
	var evidences []EvidenceItem
	domainFrequency := make(map[string]int)

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			name := filepath.Base(path)
			if ignoredDirs[name] || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".go" && ext != ".ts" && ext != ".js" && ext != ".py" && ext != ".cs" && ext != ".rs" {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		rel, _ := filepath.Rel(root, path)
		relSlash := filepath.ToSlash(rel)

		scanner := bufio.NewScanner(file)
		lineNum := 1
		for scanner.Scan() {
			lineText := scanner.Text()
			lineLower := strings.ToLower(lineText)

			for _, kw := range keywords {
				if strings.Contains(lineLower, kw) {
					trimmed := strings.TrimSpace(lineText)
					evidences = append(evidences, EvidenceItem{
						Source:  "code",
						File:    relSlash,
						Lines:   fmt.Sprintf("%d", lineNum),
						Context: trimmed,
					})

					// Determinar dominio a partir de la ruta
					parts := strings.Split(relSlash, "/")
					if len(parts) > 1 {
						domainCandidate := parts[0]
						if domainCandidate == "internal" || domainCandidate == "src" || domainCandidate == "pkg" {
							if len(parts) > 2 {
								domainCandidate = parts[1]
							}
						}
						domainFrequency[domainCandidate]++
					}

					if len(evidences) >= 5 {
						return nil
					}
					break
				}
			}
			lineNum++
		}
		return nil
	})

	// Elegir dominio más frecuente
	mostFreqDomain := "general"
	maxFreq := 0
	for d, count := range domainFrequency {
		if count > maxFreq {
			maxFreq = count
			mostFreqDomain = d
		}
	}

	return evidences, mostFreqDomain
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
