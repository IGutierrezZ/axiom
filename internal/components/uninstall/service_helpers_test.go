package uninstall

// NewService crea el servicio de desinstalación global de siempre. Solo lo usan las
// pruebas: el código de producción siempre indica el ámbito con NewScopedService.
func NewService(homeDir, workspaceDir, appVersion string) (*Service, error) {
	return NewScopedService(homeDir, workspaceDir, appVersion, Options{})
}
