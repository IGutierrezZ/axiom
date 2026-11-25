package claude

import (
	"os"
	"reflect"
	"testing"
)

func TestUserMCPServerNamesReturnsOnlySortedKeys(t *testing.T) {
	home := t.TempDir()
	body := `{"oauthAccount":{"token":"no-debe-salir"},"mcpServers":{"work-copilot":{"command":"x"},"engram":{"command":"y","env":{"K":"secreto"}}},"projects":{}}`
	if err := os.WriteFile(UserConfigPath(home), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	got := UserMCPServerNames(home)
	if want := []string{"engram", "work-copilot"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("nombres = %v, quiero %v", got, want)
	}
}

func TestUserMCPServerNamesIsEmptyWhenTheFileIsMissingUnreadableOrHasNoServers(t *testing.T) {
	missing := t.TempDir()
	if got := UserMCPServerNames(missing); len(got) != 0 {
		t.Errorf("sin fichero = %v", got)
	}
	for name, body := range map[string]string{
		"JSON roto":    `{"mcpServers": `,
		"sin clave":    `{"projects":{}}`,
		"tipo erróneo": `{"mcpServers": ["a"]}`,
	} {
		home := t.TempDir()
		if err := os.WriteFile(UserConfigPath(home), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := UserMCPServerNames(home); len(got) != 0 {
			t.Errorf("%s = %v, quiero vacío", name, got)
		}
	}
}
