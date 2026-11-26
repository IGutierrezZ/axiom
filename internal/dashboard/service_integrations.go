package dashboard

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/identitystore"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/settings"
)

// Integraciones del dashboard (C-07-h, D-04 a D-06): el estado de cada plugin, el guardado del PAT
// en el almacén de secretos y las llamadas al tracker por la pasarela inyectada. Ninguna respuesta
// lleva el valor de un secreto, un fragmento ni su longitud; los textos que pueden derivar de un
// error externo pasan por redactSecrets antes de salir.

// integrationsError es el fallo tipado de las rutas de integraciones: se traduce tal cual a
// `{code, error, envVar?, candidates?}` con su estado HTTP. Los métodos del servicio lo devuelven
// como puntero concreto (nil sin fallo), nunca como `error`.
type integrationsError struct {
	Status     int
	Code       string
	Message    string
	EnvVar     string
	Candidates []IntegrationCandidateDTO
}

func integrationsFail(status int, code, message string) *integrationsError {
	return &integrationsError{Status: status, Code: code, Message: message}
}

// secretLookupBudget acota la lectura del almacén de secretos (un keyring bloqueado no cuelga el GET).
const secretLookupBudget = 5 * time.Second

// redactedMarker sustituye al valor de un secreto en cualquier texto que salga del servidor.
const redactedMarker = "[redactado]"

// integrationSecretRef valida que el plugin está en el catálogo y que declara el secreto.
func integrationSecretRef(plugin, name string) (manifest.Manifest, manifest.SecretSpec, *integrationsError) {
	m, ok := manifest.Lookup(plugin)
	if !ok {
		return manifest.Manifest{}, manifest.SecretSpec{}, unknownPluginError(plugin)
	}
	for _, spec := range m.Secrets {
		if spec.Name == name {
			return m, spec, nil
		}
	}
	return manifest.Manifest{}, manifest.SecretSpec{}, integrationsFail(http.StatusUnprocessableEntity, "unknown_secret",
		"el plugin "+plugin+" no declara el secreto indicado: usa uno de los secretos de su manifiesto")
}

func unknownPluginError(plugin string) *integrationsError {
	ids := make([]string, 0, 1)
	for _, m := range manifest.Catalog() {
		ids = append(ids, m.ID)
	}
	list := "ninguno en esta versión"
	if len(ids) > 0 {
		list = strings.Join(ids, ", ")
	}
	return integrationsFail(http.StatusNotFound, "unknown_plugin",
		"plugin de integración desconocido: los admitidos en esta versión son "+list)
}

// secretValues devuelve los valores de los secretos del plugin que hoy resuelve el almacén; sirven
// solo para redactar. Ignora los que no se pueden leer.
func secretValues(ctx context.Context, m manifest.Manifest) []string {
	var out []string
	store := secrets.Default()
	for _, spec := range m.Secrets {
		if s, _, err := store.Get(ctx, secrets.Ref{Plugin: m.ID, Name: spec.Name}); err == nil {
			if v := s.Reveal(); v != "" {
				out = append(out, v)
			}
		}
	}
	return out
}

// redactSecrets sustituye cada valor por [redactado].
func redactSecrets(text string, values ...string) string {
	for _, v := range values {
		if v != "" {
			text = strings.ReplaceAll(text, v, redactedMarker)
		}
	}
	return text
}

// GetIntegrations devuelve el estado de integraciones del proyecto activo. Sin pasarela se degrada:
// el manifiesto, los valores y el estado del secreto se sirven y los work items no aplican.
func (s *Service) GetIntegrations(ctx context.Context) (*IntegrationsDTO, error) {
	loc, err := s.settingsLocation()
	if err != nil {
		return nil, err
	}
	snap, err := settings.Read(loc.ConfigPath)
	if err != nil {
		return nil, err
	}
	dto := &IntegrationsDTO{
		ConfigPath:     snap.ConfigPath,
		Revision:       snap.Revision,
		Writable:       snap.Writable,
		WriteBlocked:   snap.WriteBlockedReason,
		DocumentErrors: append([]string{}, snap.DocumentErrors...),
		Plugins:        []IntegrationPluginDTO{},
	}

	lookupCtx, cancel := context.WithTimeout(ctx, secretLookupBudget)
	defer cancel()
	identities, identityErr := loadIdentities()
	store := secrets.Default()
	for _, m := range manifest.Catalog() {
		plugin := IntegrationPluginDTO{Manifest: m, Entries: []SettingDTO{}, Secrets: []SecretStatusDTO{}, Identities: []IdentityDTO{}}
		explicitlyOff := false
		for _, e := range snap.Entries {
			if e.Plugin != m.ID {
				continue
			}
			plugin.Entries = append(plugin.Entries, settingDTOFrom(e))
			plugin.Declared = plugin.Declared || e.Declared
			if off, isBool := e.Value.(bool); e.Key == "integrations["+m.ID+"].enabled" && isBool && !off {
				explicitlyOff = true
			}
		}
		// Como workspace.Active: un plugin declarado está activo salvo `enabled: false`.
		plugin.Enabled = plugin.Declared && !explicitlyOff
		for _, spec := range m.Secrets {
			ref := secrets.Ref{Plugin: m.ID, Name: spec.Name}
			_, source, _ := store.Get(lookupCtx, ref)
			plugin.Secrets = append(plugin.Secrets, SecretStatusDTO{Name: spec.Name, Source: string(source), EnvVar: ref.EnvVar(), Required: spec.Required})
		}
		if identityErr != nil {
			plugin.IdentityError = identityErr.Error()
		} else {
			plugin.Identities = identitiesOf(identities, m.ID)
		}
		dto.Plugins = append(dto.Plugins, plugin)
	}

	if gw := s.integrationsGateway(); gw != nil {
		app := gw.WorkItemApplicability(s.getRootPath())
		dto.WorkItems = WorkItemApplicabilityDTO{Applicable: app.Applicable, Plugin: app.Plugin, RefAlias: app.RefAlias, Reason: app.Reason}
	} else {
		dto.WorkItems = WorkItemApplicabilityDTO{Reason: "el servidor no tiene la pasarela de integraciones: el alta desde un work item no está disponible"}
	}
	return dto, nil
}

func loadIdentities() (identitystore.File, error) {
	path, err := identitystore.DefaultPath()
	if err != nil {
		return identitystore.File{}, err
	}
	return identitystore.Store{Path: path}.Load()
}

func identitiesOf(file identitystore.File, plugin string) []IdentityDTO {
	accounts := make([]string, 0, len(file.Plugins[plugin]))
	for account := range file.Plugins[plugin] {
		accounts = append(accounts, account)
	}
	sort.Strings(accounts)
	out := make([]IdentityDTO, 0, len(accounts))
	for _, account := range accounts {
		e := file.Plugins[plugin][account]
		out = append(out, identityDTO(account, e))
	}
	return out
}

func identityDTO(account string, e identitystore.Entry) IdentityDTO {
	return IdentityDTO{Account: account, ID: e.ID, DisplayName: e.DisplayName, Source: e.Source, ResolvedAt: e.ResolvedAt.UTC().Format("2006-01-02T15:04:05Z")}
}

// validSecretValue aplica las reglas del PAT y devuelve el valor a guardar (sin espacios
// laterales): sin caracteres de control (< 0x20 y 0x7f, saltos de línea incluidos, comprobados
// sobre el valor recibido), no vacío ni solo blancos y de como mucho secrets.MaxValueBytes.
func validSecretValue(value string, ref secrets.Ref) (string, *integrationsError) {
	if !utf8.ValidString(value) {
		return "", integrationsFail(http.StatusBadRequest, "invalid_request", "el valor del PAT no es texto válido")
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return "", integrationsFail(http.StatusBadRequest, "invalid_request", "el valor del PAT no puede contener saltos de línea ni caracteres de control")
		}
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", integrationsFail(http.StatusBadRequest, "invalid_request", "el valor del PAT no puede estar vacío")
	}
	if len(trimmed) > secrets.MaxValueBytes {
		return "", tooLargeError(ref)
	}
	return trimmed, nil
}

func tooLargeError(ref secrets.Ref) *integrationsError {
	err := integrationsFail(http.StatusBadRequest, "value_too_large",
		"el PAT supera el tamaño que admiten los almacenes del sistema: guárdalo en la variable de entorno "+ref.EnvVar()+" y reinicia Axiom")
	err.EnvVar = ref.EnvVar()
	return err
}

func noWritableStoreError(ref secrets.Ref) *integrationsError {
	err := integrationsFail(http.StatusUnprocessableEntity, "no_writable_store",
		"este sistema no tiene almacén de secretos propio: define la variable de entorno "+ref.EnvVar()+" con el PAT y reinicia Axiom; Axiom nunca guarda secretos en texto plano")
	err.EnvVar = ref.EnvVar()
	return err
}

func storeFailedError(ref secrets.Ref, verb string, cause error, redact ...string) *integrationsError {
	detail := redactSecrets(cause.Error(), redact...)
	err := integrationsFail(http.StatusInternalServerError, "secret_store_failed",
		"no se pudo "+verb+" el PAT en el almacén de secretos del sistema ("+detail+"): puedes definir la variable de entorno "+ref.EnvVar()+" como alternativa")
	err.EnvVar = ref.EnvVar()
	return err
}

// secretOutcome relee la procedencia resultante: es lo único que sale de guardar o eliminar.
func secretOutcome(ctx context.Context, ref secrets.Ref) IntegrationSecretResponse {
	_, source, _ := secrets.Default().Get(ctx, ref)
	return IntegrationSecretResponse{Plugin: ref.Plugin, Name: ref.Name, Source: string(source), EnvVar: ref.EnvVar(), ShadowedByEnv: source == secrets.SourceEnv}
}

// SaveIntegrationSecret guarda el PAT en el almacén del sistema y responde su procedencia.
func (s *Service) SaveIntegrationSecret(ctx context.Context, plugin, name, value string) (*IntegrationSecretResponse, *integrationsError) {
	if _, _, bad := integrationSecretRef(plugin, name); bad != nil {
		return nil, bad
	}
	ref := secrets.Ref{Plugin: plugin, Name: name}
	trimmed, bad := validSecretValue(value, ref)
	if bad != nil {
		return nil, bad
	}
	if err := secrets.Default().Set(ctx, ref, secrets.NewSecret(trimmed)); err != nil {
		var tooLarge *secrets.ValueTooLargeError
		switch {
		case errors.As(err, &tooLarge):
			return nil, tooLargeError(ref)
		case errors.Is(err, secrets.ErrNoWritableStore):
			return nil, noWritableStoreError(ref)
		default:
			return nil, storeFailedError(ref, "guardar", err, value, trimmed)
		}
	}
	resp := secretOutcome(ctx, ref)
	return &resp, nil
}

// DeleteIntegrationSecret elimina el PAT del almacén. Es idempotente.
func (s *Service) DeleteIntegrationSecret(ctx context.Context, plugin, name string) (*IntegrationSecretResponse, *integrationsError) {
	if _, _, bad := integrationSecretRef(plugin, name); bad != nil {
		return nil, bad
	}
	ref := secrets.Ref{Plugin: plugin, Name: name}
	if err := secrets.Default().Delete(ctx, ref); err != nil {
		if errors.Is(err, secrets.ErrNoWritableStore) {
			return nil, noWritableStoreError(ref)
		}
		return nil, storeFailedError(ref, "eliminar", err)
	}
	resp := secretOutcome(ctx, ref)
	return &resp, nil
}

// requireIntegrationSecrets comprueba, antes de llamar a la pasarela, que cada secreto obligatorio
// del manifiesto tiene procedencia env o keyring (prov:WEB:5); si no, no se llama al tracker.
func (s *Service) requireIntegrationSecrets(ctx context.Context, m manifest.Manifest) *integrationsError {
	store := secrets.Default()
	for _, spec := range m.Secrets {
		if !spec.Required {
			continue
		}
		ref := secrets.Ref{Plugin: m.ID, Name: spec.Name}
		_, source, _ := store.Get(ctx, ref)
		if source == secrets.SourceEnv || source == secrets.SourceKeyring {
			continue
		}
		msg := "falta el PAT de " + m.ID + ": guárdalo en esta pantalla o define la variable de entorno " + ref.EnvVar()
		if source == secrets.SourceKeyringUnavailable {
			msg = "el almacén de secretos del sistema no está disponible para " + m.ID + ": define la variable de entorno " + ref.EnvVar() + " con el PAT"
		}
		err := integrationsFail(http.StatusUnprocessableEntity, "secret_missing", msg)
		err.EnvVar = ref.EnvVar()
		return err
	}
	return nil
}

// ConnectIntegration prueba la conexión por la pasarela: resuelve y persiste la identidad.
func (s *Service) ConnectIntegration(ctx context.Context, plugin, user string) (*IntegrationConnectResponse, *integrationsError) {
	m, gw, bad := s.prepareGatewayCall(ctx, plugin)
	if bad != nil {
		return nil, bad
	}
	outcome, err := gw.Connect(ctx, s.getRootPath(), plugin, user)
	if err != nil {
		return nil, mapGatewayError(ctx, m, err, true)
	}
	resp := &IntegrationConnectResponse{
		Identity:   IdentityDTO{Account: outcome.Identity.Account, ID: outcome.Identity.ID, DisplayName: outcome.Identity.DisplayName, Source: outcome.Identity.Source, ResolvedAt: outcome.ResolvedAt},
		ResolvedAt: outcome.ResolvedAt,
	}
	if outcome.Replaced != nil {
		replaced := identityDTO(outcome.Identity.Account, *outcome.Replaced)
		resp.Replaced = &replaced
	}
	return resp, nil
}

// DiscoverIntegration pide el descubrimiento del tracker por la pasarela. No escribe nada.
func (s *Service) DiscoverIntegration(ctx context.Context, plugin string) (*integrations.Discovery, *integrationsError) {
	m, gw, bad := s.prepareGatewayCall(ctx, plugin)
	if bad != nil {
		return nil, bad
	}
	discovery, err := gw.Discover(ctx, s.getRootPath(), plugin)
	if err != nil {
		return nil, mapGatewayError(ctx, m, err, false)
	}
	return &discovery, nil
}

// prepareGatewayCall valida el plugin, que haya pasarela y que el PAT obligatorio esté disponible,
// todo antes de llamar al tracker.
func (s *Service) prepareGatewayCall(ctx context.Context, plugin string) (manifest.Manifest, IntegrationsGateway, *integrationsError) {
	m, ok := manifest.Lookup(plugin)
	if !ok {
		return manifest.Manifest{}, nil, unknownPluginError(plugin)
	}
	gw := s.integrationsGateway()
	if gw == nil {
		return manifest.Manifest{}, nil, integrationsFail(http.StatusServiceUnavailable, "integrations_gateway_missing",
			"el servidor no tiene configurada la pasarela de integraciones: reinicia Axiom con una versión completa")
	}
	if bad := s.requireIntegrationSecrets(ctx, m); bad != nil {
		return manifest.Manifest{}, nil, bad
	}
	return m, gw, nil
}

// mapGatewayError traduce un error de la pasarela a su código tipado y redacta de él cualquier
// valor de secreto del plugin. forConnect distingue el almacén de identidad, que solo conecta.
func mapGatewayError(ctx context.Context, m manifest.Manifest, err error, forConnect bool) *integrationsError {
	redact := secretValues(context.WithoutCancel(ctx), m)
	text := func() string { return redactSecrets(err.Error(), redact...) }

	var selection *integrations.IdentitySelectionError
	var missing *secrets.MissingError
	var unavailable *secrets.KeyringUnavailableError
	var corrupt *identitystore.CorruptError
	switch {
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded):
		return integrationsFail(http.StatusGatewayTimeout, "integration_timeout",
			"el servicio externo no respondió a tiempo: comprueba la red y vuelve a intentarlo")
	case errors.As(err, &selection):
		out := integrationsFail(http.StatusConflict, "identity_selection_required",
			"no se pudo resolver tu identidad sola: elige tu usuario entre los miembros del equipo")
		if selection.User != "" {
			out = integrationsFail(http.StatusUnprocessableEntity, "identity_user_unknown",
				"el usuario elegido no coincide con ningún miembro del equipo: elige uno de la lista")
		}
		out.Candidates = make([]IntegrationCandidateDTO, 0, len(selection.Candidates))
		for _, c := range selection.Candidates {
			out.Candidates = append(out.Candidates, IntegrationCandidateDTO{ID: c.ID, DisplayName: c.DisplayName})
		}
		return out
	case errors.Is(err, cli.ErrIntegrationPluginUnknown):
		return integrationsFail(http.StatusNotFound, "unknown_plugin", text())
	case errors.Is(err, cli.ErrIntegrationNotConfigured):
		return integrationsFail(http.StatusUnprocessableEntity, "plugin_not_configured", text())
	case errors.Is(err, cli.ErrIntegrationDisabled):
		return integrationsFail(http.StatusUnprocessableEntity, "plugin_disabled", text())
	case errors.Is(err, cli.ErrIntegrationConfigInvalid):
		return integrationsFail(http.StatusUnprocessableEntity, "config_invalid", text())
	case errors.Is(err, cli.ErrIntegrationNotRegistered):
		return integrationsFail(http.StatusUnprocessableEntity, "plugin_not_compiled", text())
	case errors.Is(err, integrations.ErrPortUnsupported):
		return integrationsFail(http.StatusUnprocessableEntity, "port_unsupported", text())
	case errors.As(err, &missing) || errors.As(err, &unavailable):
		out := integrationsFail(http.StatusUnprocessableEntity, "secret_missing", text())
		if missing != nil {
			out.EnvVar = missing.Ref.EnvVar()
		} else {
			out.EnvVar = unavailable.EnvVar
		}
		return out
	case forConnect && errors.As(err, &corrupt):
		return integrationsFail(http.StatusInternalServerError, "identity_store_failed", text())
	default:
		// Incluye *integrations.PortError: su mensaje ya sale saneado del plugin.
		return integrationsFail(http.StatusBadGateway, "integration_failed", text())
	}
}
