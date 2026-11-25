package cli

import "testing"

// managedAssetDigest solo depende del contenido de los activos embebidos: dos
// llamadas seguidas devuelven el mismo valor no vacío.
func TestManagedAssetDigestIsStable(t *testing.T) {
	first, err := managedAssetDigest()
	if err != nil {
		t.Fatalf("managedAssetDigest() error = %v", err)
	}
	second, err := managedAssetDigest()
	if err != nil {
		t.Fatalf("managedAssetDigest() error = %v", err)
	}
	if first != second || first == "" {
		t.Fatalf("digest is not stable: %q then %q", first, second)
	}
}
