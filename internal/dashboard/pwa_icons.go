package dashboard

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"sync"
)

// Iconos de la PWA: el triángulo del logo (m12 3 10 18H2L12 3z, trazo 2,6 sobre
// una rejilla de 24) rasterizado sobre el fondo de marca, con antialias por
// supersampling. Se generan una vez por variante y se sirven desde memoria.

const iconSamples = 4 // submuestras por eje y píxel

type iconKey struct {
	size     int
	maskable bool
}

type iconEntry struct {
	once sync.Once
	data []byte
}

var (
	iconMu    sync.Mutex
	iconCache = map[iconKey]*iconEntry{}
)

func iconPNG(size int, maskable bool) []byte {
	key := iconKey{size, maskable}
	iconMu.Lock()
	e, ok := iconCache[key]
	if !ok {
		e = &iconEntry{}
		iconCache[key] = e
	}
	iconMu.Unlock()
	e.once.Do(func() {
		var buf bytes.Buffer
		if err := png.Encode(&buf, renderIcon(size, maskable)); err == nil {
			e.data = buf.Bytes()
		}
	})
	return e.data
}

func iconHandler(size int, maskable bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := iconPNG(size, maskable)
		if len(data) == 0 {
			http.Error(w, "no se pudo generar el icono", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(data)
	}
}

type pt struct{ x, y float64 }

var logoTriangle = [3]pt{{12, 3}, {22, 21}, {2, 21}}

const logoHalfStroke = 1.3

func segDist(p, a, b pt) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	t := ((p.x-a.x)*dx + (p.y-a.y)*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(p.x-(a.x+t*dx), p.y-(a.y+t*dy))
}

func lerpColor(a, b [3]float64, t float64) [3]float64 {
	return [3]float64{a[0] + (b[0]-a[0])*t, a[1] + (b[1]-a[1])*t, a[2] + (b[2]-a[2])*t}
}

func gradient(stops [][3]float64, t float64) [3]float64 {
	t = math.Max(0, math.Min(1, t))
	if t < 0.5 {
		return lerpColor(stops[0], stops[1], t*2)
	}
	return lerpColor(stops[1], stops[2], (t-0.5)*2)
}

// renderIcon pinta un icono cuadrado opaco. En la variante maskable el logo
// queda dentro de la zona segura (círculo del 80 % central).
func renderIcon(size int, maskable bool) *image.RGBA {
	bgA, bgB := [3]float64{0x0a, 0x0e, 0x17}, [3]float64{0x0f, 0x17, 0x2a}
	marks := [][3]float64{{0x38, 0xbd, 0xf8}, {0x81, 0x8c, 0xf8}, {0xc0, 0x84, 0xfc}}

	// Unidades de la rejilla por píxel: el logo (con su trazo) debe caber.
	span := 30.0
	if maskable {
		span = 40.0
	}
	scale := float64(size) / span
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	n := float64(iconSamples * iconSamples)
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			var cover float64
			for sy := 0; sy < iconSamples; sy++ {
				for sx := 0; sx < iconSamples; sx++ {
					fx := float64(px) + (float64(sx)+0.5)/iconSamples
					fy := float64(py) + (float64(sy)+0.5)/iconSamples
					p := pt{(fx-float64(size)/2)/scale + 12, (fy-float64(size)/2)/scale + 12}
					d := math.Min(segDist(p, logoTriangle[0], logoTriangle[1]),
						math.Min(segDist(p, logoTriangle[1], logoTriangle[2]), segDist(p, logoTriangle[2], logoTriangle[0])))
					if d <= logoHalfStroke {
						cover++
					}
				}
			}
			cover /= n
			bg := lerpColor(bgA, bgB, (float64(px)+float64(py))/float64(2*size))
			// El degradado de la marca recorre la diagonal del logo (2,3)-(22,21).
			gx := (float64(px)+0.5-float64(size)/2)/scale + 12
			gy := (float64(py)+0.5-float64(size)/2)/scale + 12
			mk := gradient(marks, ((gx-2)+(gy-3))/38)
			c := lerpColor(bg, mk, cover)
			img.SetRGBA(px, py, color.RGBA{uint8(c[0] + 0.5), uint8(c[1] + 0.5), uint8(c[2] + 0.5), 255})
		}
	}
	return img
}
