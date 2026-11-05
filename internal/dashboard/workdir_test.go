package dashboard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNeutralWorkDir(t *testing.T) {
	root := t.TempDir()
	ws := filepath.Join(root, "ws")
	inside := filepath.Join(ws, "sub")
	outside := filepath.Join(root, "otro")
	volumeRoot := filepath.VolumeName(ws) + string(filepath.Separator)

	cases := []struct {
		name       string
		candidates []string
		want       string
	}{
		{"primer candidato fuera", []string{outside, root}, outside},
		{"salta el propio workspace", []string{ws, outside}, outside},
		{"salta un subdirectorio del workspace", []string{inside, outside}, outside},
		{"salta candidatos vacíos", []string{"", outside}, outside},
		{"directorio padre del workspace vale", []string{root}, root},
		{"prefijo de nombre no es contención", []string{ws + "-bis"}, ws + "-bis"},
		{"sin candidatos válidos usa la raíz del volumen", []string{ws, inside}, volumeRoot},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := neutralWorkDir(ws, tc.candidates...); got != tc.want {
				t.Errorf("neutralWorkDir(%q, %v) = %q; quiero %q", ws, tc.candidates, got, tc.want)
			}
		})
	}
}

func TestServerWorkDirOutsideWorkspace(t *testing.T) {
	ws := t.TempDir()
	got := ServerWorkDir(ws)
	if isWithinDir(ws, got) {
		t.Fatalf("ServerWorkDir(%q) = %q, dentro del workspace", ws, got)
	}
	if home, err := os.UserHomeDir(); err == nil && !isWithinDir(ws, home) && got != home {
		t.Errorf("ServerWorkDir(%q) = %q; quiero el home %q", ws, got, home)
	}
}
