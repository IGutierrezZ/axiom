package workspace

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Windows PowerShell 5.1 lee un .ps1 sin BOM con la página de códigos ANSI, así que
// las tildes de un script UTF-8 sin BOM salen mal («encontrÃ³»). Todo script de
// scripts/ con bytes no ASCII debe empezar por el BOM UTF-8. Un script solo ASCII
// no lo necesita: scripts/install.ps1 debe quedarse sin él, porque `irm | iex`
// trata BOM+#Requires como una orden inválida (TestWindowsInstallScriptHasNoUTF8BOM).
func TestScriptsPowerShellWithNonASCIIStartWithUTF8BOM(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}
	root := filepath.Join("..", "..", "scripts")
	checked := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".ps1") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		body := bytes.TrimPrefix(content, bom)
		nonASCII := bytes.IndexFunc(body, func(r rune) bool { return r > 0x7F }) >= 0
		if !nonASCII {
			return nil
		}
		checked++
		if !bytes.HasPrefix(content, bom) {
			t.Errorf("%s tiene texto no ASCII y no empieza por el BOM UTF-8: PowerShell 5.1 mostrará mal sus tildes", filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("recorrer scripts/: %v", err)
	}
	if checked == 0 {
		t.Fatal("no se encontró ningún .ps1 con texto no ASCII en scripts/: la prueba no comprueba nada")
	}
}
