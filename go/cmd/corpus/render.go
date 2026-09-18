package main

// What the port draws, page by page, as migration/oracle/JavaRender.java writes
// it for PDFBox, and the comparison of the two.
//
// Each page is rendered at 72 dpi as RGB and written as a row: its size, a digest
// of its pixels, and a 16 by 16 grid of the mean brightness of each cell. The
// digest says whether two renderings agree to the last bit, which two rasterisers
// rarely do at the edges of what they draw. The grid says how far apart they are
// where they do not: a cell is a mean over a sixteenth of the page each way, so
// antialiasing washes out of it and a missing glyph, a wrong colour or a shifted
// image does not.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering/raster"
)

const (
	renderMaxPages = 1000
	renderGrid     = 16
)

var renderGroup = group{
	child:  "-onerender",
	header: "file\tpage\tstatus\twidth\theight\texact\tgrid",
	fail: func(why string) []string {
		return []string{"0\t" + why + "\t0\t0\t-\t-"}
	},
}

// renderPageRow renders one page and answers its row after the file name.
func renderPageRow(doc *pdmodel.PDDocument, index int) (row string) {
	failed := strconv.Itoa(index+1) + "\terror\t0\t0\t-\t-"
	defer func() {
		if recover() != nil {
			row = failed
		}
	}()
	img, err := raster.RenderPageWithDPI(doc, index, 72, rendering.RGB, false)
	if err != nil || img == nil {
		return failed
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	sum := sha256.New()
	var sums, counts [renderGrid * renderGrid]int64
	rgb := make([]byte, width*3)
	for y := 0; y < height; y++ {
		gy := y * renderGrid / height
		for x := 0; x < width; x++ {
			var r, g, b int
			switch m := img.(type) {
			case *image.RGBA:
				o := m.PixOffset(bounds.Min.X+x, bounds.Min.Y+y)
				r, g, b = int(m.Pix[o]), int(m.Pix[o+1]), int(m.Pix[o+2])
			case *image.NRGBA:
				o := m.PixOffset(bounds.Min.X+x, bounds.Min.Y+y)
				r, g, b = int(m.Pix[o]), int(m.Pix[o+1]), int(m.Pix[o+2])
			default:
				c := color.NRGBAModel.Convert(img.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.NRGBA)
				r, g, b = int(c.R), int(c.G), int(c.B)
			}
			rgb[x*3], rgb[x*3+1], rgb[x*3+2] = byte(r), byte(g), byte(b)
			gx := x * renderGrid / width
			sums[gy*renderGrid+gx] += int64((299*r + 587*g + 114*b + 500) / 1000)
			counts[gy*renderGrid+gx]++
		}
		sum.Write(rgb)
	}
	var grid strings.Builder
	for i := range sums {
		mean := int64(0)
		if counts[i] > 0 {
			mean = sums[i] / counts[i]
		}
		fmt.Fprintf(&grid, "%02x", mean)
	}
	return fmt.Sprintf("%d\tok\t%d\t%d\t%s\t%s", index+1, width, height, hex.EncodeToString(sum.Sum(nil)[:8]), grid.String())
}

// renderNow renders one way of opening one file in this process.
func renderNow(j job) []string {
	document, err := func() (d *pdmodel.PDDocument, err error) {
		defer func() {
			if p := recover(); p != nil {
				err = fmt.Errorf("panic: %v", p)
			}
		}()
		return openDocument(j)
	}()
	if err != nil {
		return []string{"0\t" + short(err) + "\t0\t0\t-\t-"}
	}
	defer document.Close()
	pages := document.NumberOfPages()
	if pages > renderMaxPages {
		pages = renderMaxPages
	}
	rows := make([]string, 0, pages)
	for i := 0; i < pages; i++ {
		rows = append(rows, renderPageRow(document, i))
	}
	return rows
}

// renderRow is one row of a render table read back.
type renderRow struct {
	status        string
	width, height string
	exact, grid   string
}

func readRender(path string) (map[string]map[int]renderRow, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	files := map[string]map[int]renderRow{}
	for i, line := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		cells := strings.Split(line, "\t")
		if len(cells) < 7 {
			continue
		}
		page, _ := strconv.Atoi(cells[1])
		if files[cells[0]] == nil {
			files[cells[0]] = map[int]renderRow{}
		}
		files[cells[0]][page] = renderRow{status: cells[2], width: cells[3], height: cells[4], exact: cells[5], grid: cells[6]}
	}
	return files, nil
}

// gridDistance answers the mean and the largest absolute difference of two
// grids' cells, or false when either is not a grid.
func gridDistance(a, b string) (float64, int, bool) {
	if len(a) != renderGrid*renderGrid*2 || len(b) != len(a) {
		return 0, 0, false
	}
	x, errA := hex.DecodeString(a)
	y, errB := hex.DecodeString(b)
	if errA != nil || errB != nil {
		return 0, 0, false
	}
	total, largest := 0, 0
	for i := range x {
		d := int(x[i]) - int(y[i])
		if d < 0 {
			d = -d
		}
		total += d
		if d > largest {
			largest = d
		}
	}
	return float64(total) / float64(len(x)), largest, true
}

// compareRender joins two render tables, PDFBox's first, and reports how far
// apart each page is. It answers the number of pages that disagree beyond the
// last bit: a different size, one side failing, a page one table does not have,
// or a grid more than a level apart on average.
func compareRender(javaPath, goPath string) (int, error) {
	java, err := readRender(javaPath)
	if err != nil {
		return 0, err
	}
	mine, err := readRender(goPath)
	if err != nil {
		return 0, err
	}
	names := make([]string, 0, len(java)+len(mine))
	for name := range java {
		names = append(names, name)
	}
	for name := range mine {
		if _, known := java[name]; !known {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	type far struct {
		name   string
		page   int
		mean   float64
		larges int
	}
	var worst []far
	var pages, identical, within1, within4, within16, beyond, sized, statuses, missing, neither, openDiffer int
	for _, name := range names {
		them, now := java[name], mine[name]
		if them == nil || now == nil {
			missing++
			side := "PDFBox's"
			if them == nil {
				side = "this program's"
			}
			fmt.Printf("  MISSING %s: in %s table only\n", name, side)
			continue
		}
		// Both drivers write a file they could not open as one row for page 0.
		_, javaFailed := them[0]
		_, goFailed := now[0]
		javaOpen, goOpen := !javaFailed, !goFailed
		if !javaOpen || !goOpen {
			if javaOpen != goOpen {
				openDiffer++
				fmt.Printf("  OPEN   %s: go %s, java %s\n", name, openCell(now), openCell(them))
			}
			continue
		}
		numbers := map[int]bool{}
		for page := range them {
			numbers[page] = true
		}
		for page := range now {
			numbers[page] = true
		}
		ordered := make([]int, 0, len(numbers))
		for page := range numbers {
			ordered = append(ordered, page)
		}
		sort.Ints(ordered)
		for _, page := range ordered {
			j, inJava := them[page]
			g, inGo := now[page]
			if !inJava || !inGo {
				missing++
				fmt.Printf("  PAGE   %s page %d: in one table only\n", name, page)
				continue
			}
			pages++
			switch {
			case j.status != "ok" && g.status != "ok":
				neither++
			case j.status != g.status:
				statuses++
				fmt.Printf("  STATUS %s page %d: go %s, java %s\n", name, page, g.status, j.status)
			case j.width != g.width || j.height != g.height:
				sized++
				fmt.Printf("  SIZE   %s page %d: go %sx%s, java %sx%s\n", name, page, g.width, g.height, j.width, j.height)
			case j.exact == g.exact:
				identical++
			default:
				mean, largest, ok := gridDistance(j.grid, g.grid)
				switch {
				case !ok:
					beyond++
				case mean <= 1:
					within1++
				case mean <= 4:
					within4++
				case mean <= 16:
					within16++
				default:
					beyond++
				}
				if ok {
					worst = append(worst, far{name, page, mean, largest})
				}
			}
		}
	}

	sort.Slice(worst, func(a, b int) bool { return worst[a].mean > worst[b].mean })
	if len(worst) > 40 {
		worst = worst[:40]
	}
	for _, w := range worst {
		if w.mean <= 1 {
			break
		}
		fmt.Printf("  FAR    %s page %d: mean %.1f levels a cell, largest %d\n", w.name, w.page, w.mean, w.larges)
	}
	fmt.Printf("\n%d pages compared, %d in one table only, %d files one side did not open\n", pages, missing, openDiffer)
	fmt.Printf("  identical to the last bit   %d\n", identical)
	fmt.Printf("  within 1 level a cell       %d\n", within1)
	fmt.Printf("  within 4 levels a cell      %d\n", within4)
	fmt.Printf("  within 16 levels a cell     %d\n", within16)
	fmt.Printf("  further apart               %d\n", beyond)
	fmt.Printf("  a different size            %d\n", sized)
	fmt.Printf("  one side failed             %d\n", statuses)
	fmt.Printf("  both sides failed           %d\n", neither)
	return within4 + within16 + beyond + sized + statuses + missing + openDiffer, nil
}

func openCell(rows map[int]renderRow) string {
	if row, failed := rows[0]; failed {
		return row.status
	}
	return fmt.Sprintf("ok, %d pages", len(rows))
}
