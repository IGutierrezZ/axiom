package changestart

import (
	"context"
	"errors"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/changestart/htmlmd"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

const (
	maxAttachmentBytes = 10 << 20 // tope de un adjunto
	attachmentBudget   = 50 << 20 // presupuesto total descargado por work item
	maxCandidates      = 100      // recursos distintos que se intentan; los siguientes se agrupan
	maxNameLen         = 48       // longitud máxima de un nombre de fichero
	maxLabelRunes      = 200      // tope del nombre del tracker que entra en la lista de adjuntos
	attachmentFileMode = 0o644
)

// Motivos de omisión: textos fijos de Axiom, nunca texto del tracker (D-19, D-26, D-27).
const (
	reasonExtension = "extensión no permitida"
	reasonType      = "tipo de contenido no permitido"
	reasonTooLarge  = "supera 10 MiB"
	reasonBudget    = "presupuesto total agotado"
	reasonFailed    = "descarga fallida"
	reasonInline    = "imagen en línea (data:)"
	reasonRelative  = "ruta relativa"
	reasonInvalid   = "URL no válida"
	reasonOverflow  = "tope de 100 recursos alcanzado"
	reasonScheme    = "esquema no permitido"
	reasonUserinfo  = "credenciales en la URL"
	reasonHost      = "host no permitido"
	reasonOrg       = "organización no permitida"
	reasonPath      = "ruta no permitida"
	reasonURL       = "URL no permitida"
)

// downloader descarga la URL de un recurso con un máximo de bytes. Lo implementa el puerto de work
// items; el pipeline no conoce ningún tracker.
type downloader func(ctx context.Context, rawURL string, limit int64) (integrations.Attachment, error)

// stagedFile es un adjunto aceptado, pendiente de escribir bajo context/attachments/.
type stagedFile struct {
	Name string
	Data []byte
}

// attachmentPipeline importa los adjuntos y las imágenes de un work item (D-19, D-26, D-27). Una
// primera aparición decide la descarga, el nombre y el resultado; las siguientes lo reutilizan.
type attachmentPipeline struct {
	download   downloader
	used       int64 // bytes descargados, aceptados o no
	candidates int
	byKey      map[string]*attachmentEntry
	order      []*attachmentEntry
	names      map[string]bool // nombres en uso, en minúsculas
	staged     []stagedFile
}

func newAttachmentPipeline(dl downloader) *attachmentPipeline {
	return &attachmentPipeline{download: dl, byKey: map[string]*attachmentEntry{}, names: map[string]bool{}}
}

// importAttached procesa los adjuntos AttachedFile en el orden de la API. Va antes que las imágenes.
func (p *attachmentPipeline) importAttached(ctx context.Context, atts []integrations.ItemAttachment) {
	for _, a := range atts {
		p.resolve(ctx, a.URL, a.Name, a.Size, false)
	}
}

// image decide el Markdown de una imagen: el enlace a su copia local o la nota de omisión, que nunca
// repite la URL original.
func (p *attachmentPipeline) image(ctx context.Context, img htmlmd.Image) string {
	e := p.resolve(ctx, img.Src, "", 0, true)
	if e.File == "" {
		return "[imagen omitida: " + e.Reason + "]"
	}
	return "![" + altText(img.Alt) + "](attachments/" + e.File + ")"
}

// entries devuelve la lista de recursos, cada uno una vez y en orden de primera aparición.
func (p *attachmentPipeline) entries() []attachmentEntry {
	out := make([]attachmentEntry, len(p.order))
	for i, e := range p.order {
		out[i] = *e
	}
	return out
}

// files devuelve los adjuntos aceptados.
func (p *attachmentPipeline) files() []stagedFile { return p.staged }

// shared devuelve la entrada agrupada de un motivo que no hace petición ni consume plaza.
func (p *attachmentPipeline) shared(key, label, reason string) *attachmentEntry {
	if e, ok := p.byKey[key]; ok {
		e.Count++
		return e
	}
	e := &attachmentEntry{Label: label, Reason: reason, Count: 1}
	p.byKey[key] = e
	p.order = append(p.order, e)
	return e
}

func (p *attachmentPipeline) resolve(ctx context.Context, rawURL, label string, declared int64, isImage bool) *attachmentEntry {
	raw := strings.TrimSpace(rawURL)
	if len(raw) >= 5 && strings.EqualFold(raw[:5], "data:") {
		return p.shared("\x00data", "", reasonInline)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return p.shared("\x00invalid", "", reasonInvalid)
	}
	if u.Scheme == "" && u.Host == "" {
		return p.shared("\x00relative", "", reasonRelative)
	}
	key, _ := dedupKey(raw)
	if e, ok := p.byKey[key]; ok {
		e.Count++
		return e
	}
	if p.candidates >= maxCandidates {
		return p.shared("\x00overflow", "recursos adicionales", reasonOverflow)
	}
	p.candidates++
	e := &attachmentEntry{Label: limitRunes(label, maxLabelRunes), Count: 1}
	p.byKey[key] = e
	p.order = append(p.order, e)
	p.fetch(ctx, e, raw, u, label, declared, isImage)
	return e
}

// fetch aplica las comprobaciones previas a la descarga, descarga y valida el resultado. Deja en e el
// nombre del fichero o el motivo de omisión; nunca devuelve error: un fallo omite el recurso.
func (p *attachmentPipeline) fetch(ctx context.Context, e *attachmentEntry, raw string, u *url.URL, name string, declared int64, isImage bool) {
	fileName := name
	if isImage {
		fileName = queryValue(u, "filename")
	}
	base, ext := "", ""
	if fileName != "" || !isImage {
		base = sanitizeName(fileName)
		_, ext = splitExt(base)
		if !extensionAllowed(ext) {
			e.Reason = reasonExtension
			return
		}
	}
	if declared > maxAttachmentBytes {
		e.Reason = reasonTooLarge
		return
	}
	remaining := int64(attachmentBudget) - p.used
	if remaining <= 0 || declared > remaining {
		e.Reason = reasonBudget
		return
	}
	limit := min(int64(maxAttachmentBytes), remaining)

	att, err := p.download(ctx, raw, limit)
	if err == nil && int64(len(att.Data)) > limit {
		err = integrations.ErrAttachmentTooLarge
	}
	if err != nil {
		if errors.Is(err, integrations.ErrAttachmentTooLarge) {
			p.used += limit // se leyó hasta el límite
		}
		e.Reason = downloadReason(err, limit)
		return
	}
	p.used += int64(len(att.Data))

	contentType := normalizeContentType(att.ContentType)
	if neverAccepted(contentType) {
		e.Reason = reasonType
		return
	}
	if ext == "" { // imagen sin fileName: la extensión sale de la firma
		if ext = imageExtFromSignature(att.Data); ext == "" {
			e.Reason = reasonType
			return
		}
		base = "imagen." + ext
	}
	if !contentTypeAllowed(ext, contentType) || !signatureMatches(ext, att.Data) {
		e.Reason = reasonType
		return
	}
	e.File = p.uniqueName(base)
	p.staged = append(p.staged, stagedFile{Name: e.File, Data: att.Data})
}

// downloadReason traduce el error de la descarga a un motivo fijo.
func downloadReason(err error, limit int64) string {
	switch {
	case errors.Is(err, integrations.ErrAttachmentTooLarge):
		if limit < maxAttachmentBytes {
			return reasonBudget
		}
		return reasonTooLarge
	case errors.Is(err, integrations.ErrResourceURLRejected):
		var rej *integrations.ResourceURLRejection
		if errors.As(err, &rej) {
			switch rej.Reason {
			case "scheme":
				return reasonScheme
			case "userinfo":
				return reasonUserinfo
			case "host":
				return reasonHost
			case "organization":
				return reasonOrg
			case "path":
				return reasonPath
			}
		}
		return reasonURL
	}
	return reasonFailed
}

// normalizeContentType deja el tipo en minúsculas y sin parámetros; ausente es octet-stream.
func normalizeContentType(ct string) string {
	if mt, _, err := mime.ParseMediaType(ct); err == nil {
		return mt
	}
	ct, _, _ = strings.Cut(ct, ";")
	if ct = strings.ToLower(strings.TrimSpace(ct)); ct != "" {
		return ct
	}
	return ctOctetStream
}

// queryValue devuelve el primer valor del parámetro name, sin distinguir mayúsculas.
func queryValue(u *url.URL, name string) string {
	for k, v := range u.Query() {
		if strings.EqualFold(k, name) && len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func altText(alt string) string {
	alt = strings.NewReplacer("[", " ", "]", " ").Replace(alt)
	return strings.Join(strings.Fields(alt), " ")
}

func limitRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// sanitizeName deja el nombre que da el tracker en un nombre de fichero seguro (D-19): la base tras
// el último / o \, solo [A-Za-z0-9._-] con el plegado ASCII de D-08 (el resto, `_`), sin puntos ni
// espacios en los extremos, `..` colapsado, un nombre de dispositivo de Windows con prefijo `_`, un
// nombre vacío como `adjunto`, la extensión en minúsculas y como mucho 48 caracteres.
func sanitizeName(raw string) string {
	base := raw
	if i := strings.LastIndexAny(base, `/\`); i >= 0 {
		base = base[i+1:]
	}
	base = foldASCII(strings.Trim(base, " ."))
	var b strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	name := b.String()
	for strings.Contains(name, "..") {
		name = strings.ReplaceAll(name, "..", ".")
	}
	name = strings.Trim(name, ".")
	if name == "" {
		return "adjunto"
	}
	stem, ext := splitExt(name)
	if isDeviceName(stem) {
		stem = "_" + stem
	}
	return joinName(stem, strings.ToLower(ext), "")
}

// splitExt separa la extensión (sin punto) del último punto; un nombre sin punto, o que solo lo tiene
// al principio, no tiene extensión.
func splitExt(name string) (stem, ext string) {
	i := strings.LastIndexByte(name, '.')
	if i <= 0 {
		return name, ""
	}
	return name[:i], name[i+1:]
}

// joinName compone stem + suffix + "." + ext recortando el stem para no superar maxNameLen.
func joinName(stem, ext, suffix string) string {
	if len(ext) > 10 {
		ext = ext[:10]
	}
	room := maxNameLen - len(suffix)
	if ext != "" {
		room -= len(ext) + 1
	}
	if room < 1 {
		room = 1
	}
	if len(stem) > room {
		stem = stem[:room]
	}
	name := stem + suffix
	if ext != "" {
		name += "." + ext
	}
	return name
}

// isDeviceName informa de si stem es un nombre de dispositivo reservado de Windows.
func isDeviceName(stem string) bool {
	head, _, _ := strings.Cut(stem, ".")
	switch up := strings.ToUpper(head); {
	case up == "CON", up == "PRN", up == "AUX", up == "NUL":
		return true
	case len(up) == 4 && (strings.HasPrefix(up, "COM") || strings.HasPrefix(up, "LPT")) && up[3] >= '1' && up[3] <= '9':
		return true
	}
	return false
}

// uniqueName resuelve las colisiones, sin distinguir mayúsculas, con -2, -3…
func (p *attachmentPipeline) uniqueName(name string) string {
	stem, ext := splitExt(name)
	cand := name
	for n := 2; p.names[strings.ToLower(cand)]; n++ {
		cand = joinName(stem, ext, "-"+strconv.Itoa(n))
	}
	p.names[strings.ToLower(cand)] = true
	return cand
}

// writeAttachmentFiles escribe los adjuntos aceptados bajo dir con permisos 0o644. Los nombres ya
// están saneados: ninguno sale de dir.
func writeAttachmentFiles(dir string, files []stagedFile) error {
	if len(files) == 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.Name), f.Data, attachmentFileMode); err != nil {
			return err
		}
	}
	return nil
}
