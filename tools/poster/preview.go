package main

import (
	"encoding/base64"
	"fmt"
	"html"
	"os"
	"strings"

	"github.com/signintech/gopdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// surface mirrors the PDF drawing operations into an SVG preview. Both outputs
// share text wrapping, coordinates, and embedded fonts to prevent layout drift.
type surface struct {
	*gopdf.GoPdf
	svg                   strings.Builder
	fill, stroke, ink     string
	lineWidth, x, y, size float64
	font                  string
}

func newSurface() *surface {
	obj := &surface{GoPdf: &gopdf.GoPdf{}, fill: "black", stroke: "black", ink: "black", lineWidth: 1}
	obj.svg.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="612" height="792" viewBox="0 0 612 792"><style>`)
	for _, font := range []struct {
		weight int
		data   []byte
	}{{400, goregular.TTF}, {700, gobold.TTF}} {
		fmt.Fprintf(&obj.svg, `@font-face{font-family:poster; font-weight:%d; src:url(data:font/ttf;base64,%s)}`, font.weight, base64.StdEncoding.EncodeToString(font.data))
	}
	obj.svg.WriteString(`</style>`)
	return obj
}

func (obj *surface) SetFillColor(r, g, b uint8) {
	obj.GoPdf.SetFillColor(r, g, b)
	obj.fill = fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func (obj *surface) SetStrokeColor(r, g, b uint8) {
	obj.GoPdf.SetStrokeColor(r, g, b)
	obj.stroke = fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func (obj *surface) SetTextColor(r, g, b uint8) {
	obj.GoPdf.SetTextColor(r, g, b)
	obj.ink = fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func (obj *surface) SetLineWidth(width float64) {
	obj.GoPdf.SetLineWidth(width)
	obj.lineWidth = width
}

func (obj *surface) SetFont(font, style string, size float64) error {
	obj.font, obj.size = font, size
	return obj.GoPdf.SetFont(font, style, size)
}

func (obj *surface) SetXY(x, y float64) {
	obj.GoPdf.SetXY(x, y)
	obj.x, obj.y = x, y
}

func (obj *surface) Text(text string) error {
	if err := obj.GoPdf.Text(text); err != nil {
		return err
	}
	weight := 400
	if obj.font == "bold" {
		weight = 700
	}
	fmt.Fprintf(&obj.svg, `<text x="%g" y="%g" font-family="poster" font-weight="%d" font-size="%g" fill="%s">%s</text>`, obj.x, obj.y, weight, obj.size, obj.ink, html.EscapeString(text))
	return nil
}

// The poster uses only filled rectangles and stroked lines, ovals, and curves.
func (obj *surface) RectFromUpperLeftWithStyle(x, y, width, height float64, style string) {
	obj.GoPdf.RectFromUpperLeftWithStyle(x, y, width, height, style)
	fmt.Fprintf(&obj.svg, `<rect x="%g" y="%g" width="%g" height="%g" fill="%s"/>`, x, y, width, height, obj.fill)
}

func (obj *surface) Line(x1, y1, x2, y2 float64) {
	obj.GoPdf.Line(x1, y1, x2, y2)
	fmt.Fprintf(&obj.svg, `<path d="M%g %g L%g %g" fill="none" stroke="%s" stroke-width="%g"/>`, x1, y1, x2, y2, obj.stroke, obj.lineWidth)
}

func (obj *surface) Oval(x1, y1, x2, y2 float64) {
	obj.GoPdf.Oval(x1, y1, x2, y2)
	fmt.Fprintf(&obj.svg, `<ellipse cx="%g" cy="%g" rx="%g" ry="%g" fill="none" stroke="%s" stroke-width="%g"/>`, (x1+x2)/2, (y1+y2)/2, (x2-x1)/2, (y2-y1)/2, obj.stroke, obj.lineWidth)
}

func (obj *surface) Curve(x0, y0, x1, y1, x2, y2, x3, y3 float64, style string) {
	obj.GoPdf.Curve(x0, y0, x1, y1, x2, y2, x3, y3, style)
	fmt.Fprintf(&obj.svg, `<path d="M%g %g C%g %g %g %g %g %g" fill="none" stroke="%s" stroke-width="%g"/>`, x0, y0, x1, y1, x2, y2, x3, y3, obj.stroke, obj.lineWidth)
}

func (obj *surface) writePreview(path string) error {
	obj.svg.WriteString(`</svg>`)
	return os.WriteFile(path, []byte(obj.svg.String()), 0644)
}
