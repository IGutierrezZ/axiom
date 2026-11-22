package skillregistry

import "path/filepath"

// Helpers de solo lectura para quien inspecciona el registro sin regenerarlo
// (`axiom doctor`, `axiom skill check`). No cambian el comportamiento de
// Regenerate: replican su cálculo.

// Freshness es el estado de la caché del registro frente al disco.
type Freshness string

const (
	// FreshnessAbsent: no hay registro o caché legible (nunca se generó).
	FreshnessAbsent Freshness = "absent"
	// FreshnessStale: la huella de la caché no coincide con la de las skills actuales.
	FreshnessStale Freshness = "stale"
	// FreshnessFresh: la caché coincide; Regenerate respondería `cache-hit`.
	FreshnessFresh Freshness = "fresh"
)

// RegistryFreshness calcula la huella exactamente como Regenerate (escaneo de
// ProjectSkillDirs y UserSkillDirs más la vista versionada) y la compara con la
// guardada en CacheRelPath. No escribe nada.
func RegistryFreshness(cwd, home string) Freshness {
	cwd = cleanPathArg(cwd)
	home = filepath.Clean(home)

	registryPath := filepath.Join(cwd, RegistryRelPath)
	cached := readCachedFingerprint(filepath.Join(cwd, CacheRelPath))
	if cached == "" || !fileExists(registryPath) {
		return FreshnessAbsent
	}
	files, err := findAllSkillFiles(uniqueExistingDirs(append(ProjectSkillDirs(cwd), UserSkillDirs(home)...)))
	if err != nil {
		return FreshnessStale
	}
	if cached == versionedFingerprint(Fingerprint(files), versionableFiles(cwd, files)) {
		return FreshnessFresh
	}
	return FreshnessStale
}

// ScanDirs devuelve las carpetas de skills existentes que el registro escanea,
// en su orden de prioridad (proyecto antes que usuario), sin duplicados.
func ScanDirs(cwd, home string) []string {
	return uniqueExistingDirs(append(ProjectSkillDirs(cleanPathArg(cwd)), UserSkillDirs(filepath.Clean(home))...))
}

// IsReservedName indica si el registro descarta en silencio una skill con ese nombre.
func IsReservedName(name string) bool { return isExcluded(name) }

// ChosenCopy devuelve cuál de las copias (rutas de SKILL.md, en el orden del
// escaneo) elige el registro para una skill que existe varias veces. Aplica la
// misma resolución que Regenerate (dedupeBySkillName). Con una lista vacía
// devuelve "".
func ChosenCopy(cwd string, copies []string) string {
	if len(copies) == 0 {
		return ""
	}
	entries := make([]SkillEntry, 0, len(copies))
	for _, p := range copies {
		entries = append(entries, SkillEntry{Name: "x", Path: p})
	}
	chosen := dedupeBySkillName(entries, cleanPathArg(cwd))
	if len(chosen) == 0 {
		return ""
	}
	return chosen[0].Path
}
