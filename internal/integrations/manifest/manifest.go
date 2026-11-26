// Package manifest es la hoja pura del marco de integraciones: describe qué es un plugin
// (identificador, capacidades, eventos, configuración, secretos e identidad) y guarda el
// catálogo cerrado de manifiestos. Solo importa la biblioteca estándar, de modo que
// internal/workspace puede validar el bloque integrations: sin ciclos de importación.
package manifest

import (
	"fmt"
	"regexp"
	"strings"
)

// SchemaV1 es el esquema serializado del manifiesto.
const SchemaV1 = "axiom.integration-manifest/v1"

// ReservedConfigKey es la clave que activa o desactiva un plugin en axiom.yaml; ningún manifiesto
// puede declararla como configuración propia.
const ReservedConfigKey = "enabled"

const (
	maxIDLength        = 32
	maxConfigKeyLength = 64
)

var (
	idPattern     = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	configPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

	refAliasPattern = regexp.MustCompile(`^[a-z][a-z0-9]{1,15}$`)
)

// forbiddenKeySegments son los segmentos (separados por "_") que delatan un secreto declarado
// como configuración. Los secretos se declaran en Secrets, nunca en Config.
var forbiddenKeySegments = map[string]bool{
	"token":       true,
	"secret":      true,
	"secrets":     true,
	"password":    true,
	"passwd":      true,
	"pwd":         true,
	"pat":         true,
	"credential":  true,
	"credentials": true,
	"apikey":      true,
}

// FieldType es el tipo de un campo de configuración.
type FieldType string

const (
	FieldString     FieldType = "string"
	FieldBool       FieldType = "bool"
	FieldInt        FieldType = "int"
	FieldStringList FieldType = "string_list"
)

// FieldTypes devuelve la lista cerrada de tipos en orden estable.
func FieldTypes() []FieldType {
	return []FieldType{FieldString, FieldBool, FieldInt, FieldStringList}
}

// IdentitySource indica de dónde sale la identidad de persona que usa el plugin.
type IdentitySource string

const (
	// IdentityNone: el plugin no necesita identidad de persona.
	IdentityNone IdentitySource = "none"
	// IdentityGitActor: usa Event.Actor (identidad git de C-07-a).
	IdentityGitActor IdentitySource = "git_actor"
	// IdentityPlugin: la resuelve el plugin contra el tracker (C-07-c).
	IdentityPlugin IdentitySource = "plugin"
)

// IdentitySources devuelve la lista cerrada de orígenes de identidad en orden estable.
func IdentitySources() []IdentitySource {
	return []IdentitySource{IdentityNone, IdentityGitActor, IdentityPlugin}
}

// Manifest describe un plugin de integración. Es serializable en JSON.
type Manifest struct {
	Schema       string        `json:"schema"`
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Capabilities []Capability  `json:"capabilities"`
	Events       []EventType   `json:"events"`
	Config       []ConfigField `json:"config"`
	Secrets      []SecretSpec  `json:"secrets"`
	Identity     IdentitySpec  `json:"identity"`
	// RefAlias es el alias corto de la referencia externa de un cambio (external_ref: <alias>:<id>).
	RefAlias string `json:"ref_alias,omitempty"`
}

// ConfigField declara una clave de configuración admitida bajo integrations.<plugin>.
type ConfigField struct {
	Key         string    `json:"key"`
	Type        FieldType `json:"type"`
	Required    bool      `json:"required"`
	Description string    `json:"description"`
	Default     any       `json:"default,omitempty"`
	Enum        []string  `json:"enum,omitempty"`
	// Format declara el formato del valor (string) o de los elementos (string_list); lista cerrada.
	Format ValueFormat `json:"format,omitempty"`
	// Min y Max acotan un campo int (ambos inclusivos); nil significa sin cota.
	Min *int `json:"min,omitempty"`
	Max *int `json:"max,omitempty"`
}

// SecretSpec declara un secreto que el plugin necesita. Su valor nunca vive en axiom.yaml.
type SecretSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// IdentitySpec declara cómo obtiene el plugin la identidad de persona.
type IdentitySpec struct {
	Source IdentitySource `json:"source"`
	Secret string         `json:"secret,omitempty"` // secreto con el que el plugin resuelve la identidad
}

// ValidationError agrupa todos los problemas de un manifiesto (o de un catálogo).
type ValidationError struct {
	Problems []string
}

func (e *ValidationError) Error() string {
	return "manifiesto de integración inválido: " + strings.Join(e.Problems, "; ")
}

// Validate comprueba el manifiesto y devuelve un *ValidationError con todos los problemas.
func (m Manifest) Validate() error {
	var p []string

	if m.Schema != SchemaV1 {
		p = append(p, fmt.Sprintf("schema %q no admitido: debe ser %q", m.Schema, SchemaV1))
	}

	switch {
	case m.ID == "":
		p = append(p, "el identificador del plugin (id) no puede estar vacío")
	case len(m.ID) > maxIDLength:
		p = append(p, fmt.Sprintf("el identificador %q supera los %d caracteres", m.ID, maxIDLength))
	case !idPattern.MatchString(m.ID):
		p = append(p, fmt.Sprintf("el identificador %q debe cumplir %s", m.ID, idPattern.String()))
	}

	p = append(p, validateCapabilities(m.Capabilities)...)
	p = append(p, validateEvents(m.Events)...)

	secretNames := map[string]bool{}
	p = append(p, validateSecrets(m.Secrets, secretNames)...)
	p = append(p, validateConfig(m.Config, secretNames)...)

	if !validIdentitySource(m.Identity.Source) {
		p = append(p, fmt.Sprintf("identity.source %q no admitido; admitidos: %s", m.Identity.Source, joinIdentitySources()))
	}
	if m.Identity.Secret != "" && !secretNames[m.Identity.Secret] {
		p = append(p, fmt.Sprintf("identity.secret %q no es un secreto declarado", m.Identity.Secret))
	}

	if m.RefAlias != "" && !refAliasPattern.MatchString(m.RefAlias) {
		p = append(p, fmt.Sprintf("ref_alias %q debe cumplir %s", m.RefAlias, refAliasPattern.String()))
	}

	if len(p) == 0 {
		return nil
	}
	return &ValidationError{Problems: p}
}

func validateCapabilities(caps []Capability) []string {
	if len(caps) == 0 {
		return []string{"debe declarar al menos una capacidad"}
	}
	var p []string
	seen := map[Capability]bool{}
	for _, c := range caps {
		switch {
		case !ValidCapability(c):
			p = append(p, fmt.Sprintf("capacidad %q no admitida; admitidas: %s", c, joinCapabilities()))
		case seen[c]:
			p = append(p, fmt.Sprintf("capacidad %q repetida", c))
		}
		seen[c] = true
	}
	return p
}

func validateEvents(events []EventType) []string {
	if len(events) == 0 {
		return []string{"debe declarar al menos un evento"}
	}
	var p []string
	seen := map[EventType]bool{}
	for _, e := range events {
		switch {
		case !ValidEvent(e):
			p = append(p, fmt.Sprintf("evento %q no admitido; admitidos: %s", e, joinEvents()))
		case seen[e]:
			p = append(p, fmt.Sprintf("evento %q repetido", e))
		}
		seen[e] = true
	}
	return p
}

func validateSecrets(secrets []SecretSpec, names map[string]bool) []string {
	var p []string
	for _, s := range secrets {
		switch {
		case !configPattern.MatchString(s.Name):
			p = append(p, fmt.Sprintf("el secreto %q debe cumplir %s", s.Name, configPattern.String()))
		case names[s.Name]:
			p = append(p, fmt.Sprintf("secreto %q repetido", s.Name))
		}
		names[s.Name] = true
	}
	return p
}

func validateConfig(fields []ConfigField, secretNames map[string]bool) []string {
	var p []string
	seen := map[string]bool{}
	for _, f := range fields {
		switch {
		case !configPattern.MatchString(f.Key):
			p = append(p, fmt.Sprintf("la clave de configuración %q debe cumplir %s", f.Key, configPattern.String()))
		case len(f.Key) > maxConfigKeyLength:
			p = append(p, fmt.Sprintf("la clave de configuración %q supera los %d caracteres", f.Key, maxConfigKeyLength))
		case f.Key == ReservedConfigKey:
			p = append(p, fmt.Sprintf("la clave %q está reservada para activar o desactivar el plugin", ReservedConfigKey))
		case seen[f.Key]:
			p = append(p, fmt.Sprintf("clave de configuración %q repetida", f.Key))
		case secretNames[f.Key]:
			p = append(p, fmt.Sprintf("la clave de configuración %q coincide con un secreto declarado", f.Key))
		case looksLikeSecret(f.Key):
			p = append(p, fmt.Sprintf("la clave de configuración %q parece un secreto: los secretos se declaran en secrets, nunca como configuración", f.Key))
		}
		seen[f.Key] = true
		p = append(p, validateField(f)...)
	}
	return p
}

func looksLikeSecret(key string) bool {
	segments := strings.Split(key, "_")
	for i, s := range segments {
		if forbiddenKeySegments[s] {
			return true
		}
		if s == "api" && i+1 < len(segments) && segments[i+1] == "key" {
			return true
		}
	}
	return false
}

func validateField(f ConfigField) []string {
	var p []string
	if !validFieldType(f.Type) {
		return []string{fmt.Sprintf("la clave %q tiene un tipo %q no admitido; admitidos: %s", f.Key, f.Type, joinFieldTypes())}
	}
	if len(f.Enum) > 0 && f.Type != FieldString && f.Type != FieldStringList {
		p = append(p, fmt.Sprintf("la clave %q: enum solo se admite en los tipos string y string_list", f.Key))
	}
	if f.Default != nil && !defaultMatchesType(f.Default, f.Type) {
		p = append(p, fmt.Sprintf("la clave %q: el valor por defecto no es del tipo declarado %q", f.Key, f.Type))
	}
	if f.Format != "" {
		p = append(p, validateFormat(f)...)
	}
	p = append(p, validateBounds(f)...)
	return p
}

func validateBounds(f ConfigField) []string {
	if f.Min == nil && f.Max == nil {
		return nil
	}
	if f.Type != FieldInt {
		return []string{fmt.Sprintf("la clave %q: min y max solo se admiten en el tipo int", f.Key)}
	}
	if f.Min != nil && f.Max != nil && *f.Min > *f.Max {
		return []string{fmt.Sprintf("la clave %q: min (%d) no puede superar max (%d)", f.Key, *f.Min, *f.Max)}
	}
	if n, ok := defaultInt(f.Default); ok {
		if f.Min != nil && n < *f.Min || f.Max != nil && n > *f.Max {
			return []string{fmt.Sprintf("la clave %q: el valor por defecto %d queda fuera del rango declarado", f.Key, n)}
		}
	}
	return nil
}

func defaultInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), n == float64(int64(n))
	}
	return 0, false
}

func validateFormat(f ConfigField) []string {
	if !ValidFormat(f.Format) {
		return []string{fmt.Sprintf("la clave %q: formato %q no admitido; admitidos: %s", f.Key, f.Format, FormatNames())}
	}
	if want := FormatType(f.Format); f.Type != want {
		return []string{fmt.Sprintf("la clave %q: el formato %q solo se admite en el tipo %s", f.Key, f.Format, want)}
	}
	var reasons []FormatProblem
	switch f.Format {
	case FormatNonEmpty, FormatTagPrefix:
		s, ok := f.Default.(string)
		if !ok {
			return nil // sin valor por defecto, o ya reportado por defaultMatchesType
		}
		check := CheckNonEmpty
		if f.Format == FormatTagPrefix {
			check = CheckTagPrefix
		}
		if r := check(s); r != "" {
			reasons = append(reasons, FormatProblem{Element: s, Reason: r})
		}
	default:
		values, ok := defaultStrings(f.Default)
		if !ok {
			return nil
		}
		if f.Format == FormatWeekdaySet {
			reasons = ParseWeekdaySet(values)
		} else {
			_, reasons = ParseTypePrefixes(values)
		}
	}
	var p []string
	for _, pr := range reasons {
		p = append(p, fmt.Sprintf("la clave %q: el valor por defecto %q no cumple el formato %s: %s", f.Key, pr.Element, f.Format, pr.Reason))
	}
	return p
}

func defaultStrings(v any) ([]string, bool) {
	switch l := v.(type) {
	case []string:
		return l, true
	case []any:
		out := make([]string, 0, len(l))
		for _, e := range l {
			s, ok := e.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	}
	return nil, false
}

func defaultMatchesType(v any, t FieldType) bool {
	switch t {
	case FieldString:
		_, ok := v.(string)
		return ok
	case FieldBool:
		_, ok := v.(bool)
		return ok
	case FieldInt:
		switch n := v.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		case float64: // un JSON decodificado entrega los enteros como float64
			return n == float64(int64(n))
		}
		return false
	case FieldStringList:
		switch l := v.(type) {
		case []string:
			return true
		case []any:
			for _, e := range l {
				if _, ok := e.(string); !ok {
					return false
				}
			}
			return true
		}
		return false
	}
	return false
}

func validFieldType(t FieldType) bool {
	for _, k := range FieldTypes() {
		if k == t {
			return true
		}
	}
	return false
}

func validIdentitySource(s IdentitySource) bool {
	for _, k := range IdentitySources() {
		if k == s {
			return true
		}
	}
	return false
}

func joinEvents() string {
	names := make([]string, 0, len(Events()))
	for _, e := range Events() {
		names = append(names, string(e))
	}
	return strings.Join(names, ", ")
}

func joinCapabilities() string {
	names := make([]string, 0, len(Capabilities()))
	for _, c := range Capabilities() {
		names = append(names, string(c))
	}
	return strings.Join(names, ", ")
}

func joinFieldTypes() string {
	names := make([]string, 0, len(FieldTypes()))
	for _, t := range FieldTypes() {
		names = append(names, string(t))
	}
	return strings.Join(names, ", ")
}

func joinIdentitySources() string {
	names := make([]string, 0, len(IdentitySources()))
	for _, s := range IdentitySources() {
		names = append(names, string(s))
	}
	return strings.Join(names, ", ")
}

// EventNames devuelve los nombres de la lista cerrada de eventos separados por coma, en el orden
// de Events. Lo usan los errores tipados que listan los eventos admitidos.
func EventNames() string { return joinEvents() }

// Clone devuelve una copia profunda del manifiesto. Conserva la diferencia entre una lista nula y
// una vacía para que reflect.DeepEqual entre un manifiesto y su copia siga siendo verdadero.
func (m Manifest) Clone() Manifest {
	out := m
	out.Capabilities = cloneSlice(m.Capabilities)
	out.Events = cloneSlice(m.Events)
	out.Secrets = cloneSlice(m.Secrets)
	if m.Config != nil {
		out.Config = make([]ConfigField, len(m.Config))
		for i, f := range m.Config {
			f.Enum = cloneSlice(f.Enum)
			f.Default = cloneDefault(f.Default)
			f.Min = cloneInt(f.Min)
			f.Max = cloneInt(f.Max)
			out.Config[i] = f
		}
	}
	return out
}

func cloneSlice[T any](s []T) []T {
	if s == nil {
		return nil
	}
	return append(make([]T, 0, len(s)), s...)
}

func cloneInt(p *int) *int {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneDefault(v any) any {
	switch d := v.(type) {
	case []string:
		return cloneSlice(d)
	case []any:
		return cloneSlice(d)
	}
	return v
}
