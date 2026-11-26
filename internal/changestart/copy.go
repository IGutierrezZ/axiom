package changestart

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// copyTree copia el árbol src en dst, que no debe existir (D-15): ficheros con 0o644 y directorios con
// 0o755, sin seguir enlaces. Un enlace, o cualquier entrada que no sea fichero ni directorio regular,
// es un error: lo creó otro proceso y nunca lo escribe la fase 1. Cada fichero se escribe con la costura
// de escritura atómica de Env.FS.
func copyTree(seams fsSeams, src, dst string) error {
	seams = seams.withDefaults()
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			return fmt.Errorf("%s es un enlace: el staging solo contiene ficheros y directorios", rel)
		case info.IsDir():
			return seams.MkdirAll(target, contextDirMode)
		case info.Mode().IsRegular():
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return seams.WriteFile(target, data, contextFileMode)
		}
		return fmt.Errorf("%s no es un fichero ni un directorio regular", rel)
	})
}

// treeEntry es un fichero de un árbol: su tamaño y su SHA-256.
type treeEntry struct {
	size int64
	sum  string
}

// digestTree lista los ficheros de root (rutas relativas con «/») con tamaño y SHA-256. Un enlace o una
// entrada irregular es un error.
func digestTree(root string) (map[string]treeEntry, error) {
	out := map[string]treeEntry{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			return fmt.Errorf("%s es un enlace", rel)
		case info.IsDir():
			return nil
		case !info.Mode().IsRegular():
			return fmt.Errorf("%s no es un fichero regular", rel)
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		hash := sha256.New()
		n, err := io.Copy(hash, file)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = treeEntry{size: n, sum: hex.EncodeToString(hash.Sum(nil))}
		return nil
	})
	return out, err
}

// errTreeMismatch indica que la copia no coincide con su origen.
var errTreeMismatch = errors.New("la copia de context/ no coincide con el staging")

// verifyTree comprueba que dst contiene exactamente los mismos ficheros que src: mismo conjunto de
// rutas relativas, mismo tamaño y mismo SHA-256 (D-15).
func verifyTree(src, dst string) error {
	want, err := digestTree(src)
	if err != nil {
		return err
	}
	got, err := digestTree(dst)
	if err != nil {
		return err
	}
	var problems []string
	for rel, w := range want {
		g, ok := got[rel]
		switch {
		case !ok:
			problems = append(problems, "falta "+rel)
		case g != w:
			problems = append(problems, "difiere "+rel)
		}
	}
	for rel := range got {
		if _, ok := want[rel]; !ok {
			problems = append(problems, "sobra "+rel)
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("%w: %v", errTreeMismatch, problems)
	}
	return nil
}
