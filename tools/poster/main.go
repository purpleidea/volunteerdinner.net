// Command poster draws the bilingual dinner poster from Hugo's JSON export.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/signintech/gopdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

type localEvent struct {
	Date, Venue, Campus, Hours string
}

type copyText struct {
	Title, Welcome, Note, Help string
}

type posterData struct {
	Year  int
	Event struct {
		Address string
		En, Fr  localEvent
	}
	Copy struct {
		En, Fr copyText
	}
	Email, URL string
}

// Drawing uses points on Letter paper. All text is measured before placement.
type drawing struct {
	pdf *surface
	err error
}

func (obj *drawing) block(text, font string, size, x, y, width float64) float64 {
	if obj.err != nil {
		return y
	}
	if obj.err = obj.pdf.SetFont(font, "", size); obj.err != nil {
		return y
	}
	lines := []string{}
	line := ""
	for _, word := range strings.Fields(text) {
		candidate := strings.TrimSpace(line + " " + word)
		var measured float64
		measured, obj.err = obj.pdf.MeasureTextWidth(candidate)
		if obj.err != nil {
			return y
		}
		if measured > width {
			if line == "" {
				obj.err = fmt.Errorf("word exceeds poster column: %q", word)
				return y
			}
			lines = append(lines, line)
			line = word
			measured, obj.err = obj.pdf.MeasureTextWidth(word)
			if obj.err != nil || measured > width {
				obj.err = fmt.Errorf("word exceeds poster column: %q", word)
				return y
			}
		} else {
			line = candidate
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	for _, line := range lines {
		obj.pdf.SetXY(x, y+size)
		if obj.err = obj.pdf.Text(line); obj.err != nil {
			return y
		}
		y += size * 1.3
	}
	return y
}

func render(data posterData, output, preview string) error {
	if data.Year < 2000 || data.Email == "" || data.URL == "" || data.Event.Address == "" {
		return fmt.Errorf("poster requires a year, contact email, website URL, and address")
	}
	for _, event := range []localEvent{data.Event.En, data.Event.Fr} {
		if event.Date == "" || event.Hours == "" || event.Venue == "" {
			return fmt.Errorf("both languages require a date, hours, and venue")
		}
	}
	for _, text := range []copyText{data.Copy.En, data.Copy.Fr} {
		if text.Title == "" || text.Welcome == "" || text.Note == "" || text.Help == "" {
			return fmt.Errorf("both languages require poster title, welcome, note, and help text")
		}
	}
	pdf := newSurface()
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeLetter})
	if err := pdf.AddTTFFontData("regular", goregular.TTF); err != nil {
		return err
	}
	if err := pdf.AddTTFFontData("bold", gobold.TTF); err != nil {
		return err
	}
	pdf.AddPage()
	pdf.SetFillColor(251, 247, 239)
	pdf.RectFromUpperLeftWithStyle(0, 0, 612, 792, "F")
	pdf.SetFillColor(37, 75, 62)
	pdf.RectFromUpperLeftWithStyle(18, 18, 576, 6, "F")
	pdf.SetTextColor(37, 75, 62)
	obj := &drawing{pdf: pdf}
	y := obj.block(fmt.Sprintf("VOLUNTEER DINNER  /  LE DÎNER BÉNÉVOLE  ·  %d", data.Year), "bold", 12, 42, 39, 528)
	y += 25
	// Each row starts below the taller of its two translated columns.
	row := func(en, fr, font string, size, gap float64) {
		left := obj.block(en, font, size, 42, y, 249)
		right := obj.block(fr, font, size, 321, y, 249)
		y = max(left, right) + gap
	}
	row(data.Copy.En.Title, data.Copy.Fr.Title, "bold", 32, 16)
	pdf.SetTextColor(166, 63, 39)
	row(data.Copy.En.Welcome, data.Copy.Fr.Welcome, "bold", 21, 27)
	pdf.SetTextColor(37, 75, 62)
	row(fmt.Sprintf("%s %d", data.Event.En.Date, data.Year), fmt.Sprintf("%s %d", data.Event.Fr.Date, data.Year), "bold", 24, 7)
	row(data.Event.En.Hours, data.Event.Fr.Hours, "regular", 22, 24)
	pdf.SetStrokeColor(217, 217, 203)
	pdf.Line(42, y, 570, y)
	y += 20
	row(data.Event.En.Venue, data.Event.Fr.Venue, "bold", 18, 6)
	row(data.Event.En.Campus, data.Event.Fr.Campus, "regular", 15, 16)
	y = obj.block(data.Event.Address, "regular", 15, 42, y, 528) + 24
	row(data.Copy.En.Note, data.Copy.Fr.Note, "regular", 14, 18)
	if y > 630 {
		return fmt.Errorf("poster content reaches %.0f points; shorten wording to fit above the footer at 630", y)
	}
	pdf.SetFillColor(224, 235, 221)
	pdf.RectFromUpperLeftWithStyle(18, 648, 576, 126, "F")
	pdf.SetTextColor(37, 75, 62)
	pdf.SetStrokeColor(37, 75, 62)
	pdf.SetLineWidth(0.5)
	pdf.Line(42, 648, 570, 648)
	// An outline version of static/images/plate.svg keeps the graphic crisp
	// and uses little ink, including when printed in black and white.
	drawPlate(pdf, 506, 713)
	if obj.block(data.Copy.En.Help, "bold", 14, 42, 664, 528) > 686 {
		return fmt.Errorf("poster help text exceeds footer space")
	}
	end := obj.block(data.Copy.Fr.Help, "regular", 14, 42, 686, 528)
	if end > 711 {
		return fmt.Errorf("poster help text exceeds footer space")
	}
	website := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(data.URL, "https://"), "http://"), "/")
	if obj.block(website, "bold", 25, 42, 714, 440) > 750 {
		return fmt.Errorf("website address exceeds footer space")
	}
	if obj.block(data.Email, "regular", 12, 42, 757, 440) > 780 {
		return fmt.Errorf("email address exceeds footer space")
	}
	if obj.err != nil {
		return obj.err
	}
	// Inset the frame by a quarter inch to accommodate printer margins.
	pdf.SetLineWidth(0.75)
	pdf.Line(18, 18, 594, 18)
	pdf.Line(594, 18, 594, 774)
	pdf.Line(594, 774, 18, 774)
	pdf.Line(18, 774, 18, 18)
	if err := pdf.WritePdf(output); err != nil {
		return err
	}
	return pdf.writePreview(preview)
}

func drawPlate(pdf *surface, x, y float64) {
	pdf.SetLineWidth(1.8)
	pdf.Oval(x+18, y+17, x+48, y+47)
	pdf.Oval(x+23, y+22, x+43, y+42)
	pdf.Line(x+10, y+17, x+10, y+27)
	pdf.Curve(x+10, y+27, x+10, y+32, x+17, y+32, x+17, y+27, "D")
	pdf.Line(x+17, y+27, x+17, y+17)
	pdf.Line(x+13.5, y+17, x+13.5, y+48)
	pdf.Line(x+54, y+48, x+54, y+17)
	pdf.Curve(x+54, y+17, x+48, y+20, x+48, y+34, x+54, y+34, "D")
}

func main() {
	input := flag.String("input", "public/poster/index.json", "Hugo poster JSON export")
	output := flag.String("output", "public/poster.pdf", "destination PDF")
	preview := flag.String("preview", "public/poster.svg", "destination SVG preview")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Usage: poster [options]")
		flag.VisitAll(func(option *flag.Flag) {
			fmt.Fprintf(flag.CommandLine.Output(), "  --%s PATH\n\t%s (default %q)\n", option.Name, option.Usage, option.DefValue)
		})
		fmt.Fprintln(flag.CommandLine.Output(), "  -h, --help\n\tShow help")
	}
	// The standard flag package accepts single-dash long options; require
	// double dashes here. Skip option values so paths are not checked as flags.
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "--" {
			break
		}
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 2 {
			fmt.Fprintf(os.Stderr, "poster: use --%s instead of %s\n", arg[1:], arg)
			os.Exit(2)
		}
		name, _, hasValue := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		if flag.Lookup(name) != nil && !hasValue {
			i++
		}
	}
	flag.Parse()
	err := func() error {
		if flag.NArg() != 0 {
			return fmt.Errorf("unexpected positional arguments: %s", strings.Join(flag.Args(), " "))
		}
		contents, err := os.ReadFile(*input)
		if err != nil {
			return err
		}
		var data posterData
		if err := json.Unmarshal(contents, &data); err != nil {
			return err
		}
		return render(data, *output, *preview)
	}()
	if err != nil {
		fmt.Fprintln(os.Stderr, "poster:", err)
		os.Exit(1)
	}
}
