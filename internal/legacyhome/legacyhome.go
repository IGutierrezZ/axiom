// Package legacyhome concentra la migración del estado de usuario que el
// producto guardaba bajo el nombre heredado `gentle-ai`.
//
// Existe solo para retirar ese nombre: el resto del código lee y escribe
// únicamente las rutas `axiom`. Aquí se reconocen las rutas viejas, una sola
// vez, y se MUEVEN a su ruta nueva cuando esta no existe. Nunca se sobrescribe
// ni se pierde información: si la ruta nueva ya existe, la vieja se deja sin
// tocar.
package legacyhome

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// DirName es el nombre del directorio de estado heredado bajo el HOME.
const DirName = ".gentle-ai"

// Dir devuelve el directorio de estado heredado (~/.gentle-ai).
func Dir(home string) string { return filepath.Join(home, DirName) }

// MoveIfAbsent mueve src a dst cuando src existe y dst no. Devuelve true si
// movió. Si dst ya existe no toca nada (se conserva la ruta nueva y la vieja
// queda intacta). Sirve para ficheros y directorios.
func MoveIfAbsent(src, dst string) (bool, error) {
	if _, err := os.Lstat(src); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if _, err := os.Lstat(dst); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return false, err
	}
	if err := os.Rename(src, dst); err != nil {
		return false, fmt.Errorf("mover %q a %q: %w", src, dst, err)
	}
	return true, nil
}

// RemoveIfEmpty retira dir solo si es un directorio vacío (nunca recursivo).
func RemoveIfEmpty(dir string) {
	_ = os.Remove(dir)
}

// MigrateHome mueve el estado heredado de ~/.gentle-ai a AxiomDir(home):
// state.json, pi-codegraph.json y las copias de seguridad (`backups`, sin
// pisar identificadores ya existentes). La caché heredada se descarta porque
// se regenera. El lanzador `bin` no se toca: lo retira `axiom sync` o el
// desinstalador. Es idempotente y no hace nada si ~/.gentle-ai no existe.
func MigrateHome(home string) error {
	legacy := Dir(home)
	info, err := os.Lstat(legacy)
	if err != nil || !info.IsDir() {
		return nil
	}
	target := system.AxiomDir(home)
	if filepath.Clean(target) == filepath.Clean(legacy) {
		return nil
	}
	var errs []error
	for _, name := range []string{"state.json", "pi-codegraph.json"} {
		if _, err := MoveIfAbsent(filepath.Join(legacy, name), filepath.Join(target, name)); err != nil {
			errs = append(errs, err)
		}
	}
	if err := migrateBackups(filepath.Join(legacy, "backups"), filepath.Join(target, "backups")); err != nil {
		errs = append(errs, err)
	}
	if err := os.RemoveAll(filepath.Join(legacy, "cache")); err != nil {
		errs = append(errs, err)
	}
	RemoveIfEmpty(legacy)
	return errors.Join(errs...)
}

// migrateBackups mueve cada copia de seguridad (un directorio por id) a la
// raíz nueva sin pisar ids existentes, y reescribe en su manifiesto las rutas
// absolutas que apuntaban a la raíz vieja.
func migrateBackups(oldRoot, newRoot string) error {
	entries, err := os.ReadDir(oldRoot)
	if err != nil {
		return nil
	}
	var errs []error
	for _, entry := range entries {
		oldDir := filepath.Join(oldRoot, entry.Name())
		newDir := filepath.Join(newRoot, entry.Name())
		moved, err := MoveIfAbsent(oldDir, newDir)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if moved && entry.IsDir() {
			if err := rewriteManifestRoots(filepath.Join(newDir, "manifest.json"), oldRoot, newRoot); err != nil {
				errs = append(errs, err)
			}
		}
	}
	RemoveIfEmpty(oldRoot)
	return errors.Join(errs...)
}

// rewriteManifestRoots sustituye el prefijo oldRoot por newRoot en `root_dir`
// y en los `snapshot_path` del manifiesto. Un manifiesto ilegible se deja tal
// cual.
func rewriteManifestRoots(path, oldRoot, newRoot string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		return nil
	}
	swap := func(v any) (string, bool) {
		s, ok := v.(string)
		if !ok {
			return "", false
		}
		if s == oldRoot || strings.HasPrefix(s, oldRoot+string(filepath.Separator)) {
			return newRoot + strings.TrimPrefix(s, oldRoot), true
		}
		return "", false
	}
	changed := false
	if s, ok := swap(doc["root_dir"]); ok {
		doc["root_dir"] = s
		changed = true
	}
	if entries, ok := doc["entries"].([]any); ok {
		for _, e := range entries {
			if m, ok := e.(map[string]any); ok {
				if s, ok := swap(m["snapshot_path"]); ok {
					m["snapshot_path"] = s
					changed = true
				}
			}
		}
	}
	if !changed {
		return nil
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
