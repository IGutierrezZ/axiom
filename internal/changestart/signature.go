package changestart

import (
	"archive/zip"
	"bytes"
	"net/http"
	"unicode/utf8"
)

const (
	ctOctetStream = "application/octet-stream"
	ctZIP         = "application/zip"
	ctOOXMLWord   = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	ctOOXMLExcel  = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	ctOOXMLSlides = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
)

// extRule es la regla de una extensión permitida (D-27): los Content-Type propios (además de
// application/octet-stream) y la comprobación de la firma de los bytes.
type extRule struct {
	types []string
	check func(data []byte) bool
}

var (
	oleMagic = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
	pngMagic = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
)

// extRules es la lista blanca de extensiones. svg no figura: nunca se acepta.
var extRules = map[string]extRule{
	"png":  {[]string{"image/png"}, isPNG},
	"jpg":  {[]string{"image/jpeg"}, isJPEG},
	"jpeg": {[]string{"image/jpeg"}, isJPEG},
	"gif":  {[]string{"image/gif"}, isGIF},
	"webp": {[]string{"image/webp"}, isWEBP},
	"bmp":  {[]string{"image/bmp", "image/x-ms-bmp"}, isBMP},
	"pdf":  {[]string{"application/pdf"}, func(d []byte) bool { return bytes.HasPrefix(d, []byte("%PDF-")) }},
	"docx": {[]string{ctOOXMLWord, ctZIP}, ooxml("word/document.xml")},
	"xlsx": {[]string{ctOOXMLExcel, ctZIP}, ooxml("xl/workbook.xml")},
	"pptx": {[]string{ctOOXMLSlides, ctZIP}, ooxml("ppt/presentation.xml")},
	"doc":  {[]string{"application/msword"}, isOLE},
	"xls":  {[]string{"application/vnd.ms-excel"}, isOLE},
	"ppt":  {[]string{"application/vnd.ms-powerpoint"}, isOLE},
	"txt":  {[]string{"text/plain"}, isText},
	"log":  {[]string{"text/plain"}, isText},
	"md":   {[]string{"text/markdown", "text/x-markdown", "text/plain"}, isText},
	"csv":  {[]string{"text/csv", "text/plain"}, isText},
	"json": {[]string{"application/json", "text/json", "text/plain"}, isText},
}

// extensionAllowed informa de si ext (en minúsculas y sin punto) está en la lista blanca.
func extensionAllowed(ext string) bool {
	_, ok := extRules[ext]
	return ok
}

// neverAccepted son los tipos que no se aceptan con ninguna extensión.
func neverAccepted(contentType string) bool {
	switch contentType {
	case "text/html", "application/xhtml+xml", "image/svg+xml":
		return true
	}
	return false
}

// contentTypeAllowed informa de si el Content-Type declarado (normalizado) es uno propio de ext o
// application/octet-stream. Los tipos de neverAccepted no valen con ninguna extensión.
func contentTypeAllowed(ext, contentType string) bool {
	rule, ok := extRules[ext]
	if !ok || neverAccepted(contentType) {
		return false
	}
	if contentType == ctOctetStream {
		return true
	}
	for _, t := range rule.types {
		if t == contentType {
			return true
		}
	}
	return false
}

// signatureMatches comprueba que los bytes concuerdan con ext según la tabla de D-27.
// http.DetectContentType solo se usa como comprobación negativa en los formatos de texto.
func signatureMatches(ext string, data []byte) bool {
	rule, ok := extRules[ext]
	return ok && rule.check(data)
}

// imageExtFromSignature devuelve la extensión de una imagen por su firma (solo png, jpg, gif, webp y
// bmp), o "" si los bytes no son una de ellas.
func imageExtFromSignature(data []byte) string {
	for _, ext := range []string{"png", "jpg", "gif", "webp", "bmp"} {
		if extRules[ext].check(data) {
			return ext
		}
	}
	return ""
}

func isPNG(d []byte) bool  { return bytes.HasPrefix(d, pngMagic) }
func isJPEG(d []byte) bool { return bytes.HasPrefix(d, []byte{0xFF, 0xD8, 0xFF}) }
func isGIF(d []byte) bool {
	return bytes.HasPrefix(d, []byte("GIF87a")) || bytes.HasPrefix(d, []byte("GIF89a"))
}
func isWEBP(d []byte) bool {
	return len(d) >= 12 && bytes.Equal(d[0:4], []byte("RIFF")) && bytes.Equal(d[8:12], []byte("WEBP"))
}
func isBMP(d []byte) bool { return len(d) >= 14 && d[0] == 'B' && d[1] == 'M' }
func isOLE(d []byte) bool { return bytes.HasPrefix(d, oleMagic) }

// isText exige UTF-8 válido (con BOM opcional), sin bytes NUL, y que http.DetectContentType no lo
// tome por HTML ni XML. El texto en UTF-16 queda fuera (limitación documentada).
func isText(d []byte) bool {
	body := bytes.TrimPrefix(d, []byte{0xEF, 0xBB, 0xBF})
	if !utf8.Valid(body) || bytes.IndexByte(body, 0) >= 0 {
		return false
	}
	switch detected := http.DetectContentType(body); {
	case len(detected) >= 9 && detected[:9] == "text/html", len(detected) >= 8 && detected[:8] == "text/xml":
		return false
	}
	return true
}

// ooxml comprueba el contenedor ZIP de un documento de Office solo por los nombres del directorio
// central: requiere [Content_Types].xml y la parte principal. No descomprime ninguna entrada, así que
// una bomba de descompresión no tiene dónde actuar.
func ooxml(mainPart string) func([]byte) bool {
	return func(d []byte) bool {
		if !bytes.HasPrefix(d, []byte("PK\x03\x04")) {
			return false
		}
		zr, err := zip.NewReader(bytes.NewReader(d), int64(len(d)))
		if err != nil {
			return false
		}
		var types, main bool
		for _, f := range zr.File {
			switch f.Name {
			case "[Content_Types].xml":
				types = true
			case mainPart:
				main = true
			}
		}
		return types && main
	}
}
