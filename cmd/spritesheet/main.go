// Command spritesheet slices grid-based sprite sheets (WebP, PNG, or JPEG)
// and packs their frames into a single transparent atlas (PNG and lossless
// WebP) plus a JSON file naming and locating every frame.
//
// Each input is read as a grid of rows; every row becomes one atlas row, in
// input order. Background pixels are keyed to transparency.
//
//	go run ./cmd/spritesheet -o out/warrior/ -cell 192x225 -origin 234,0 -rows 5 \
//	  -counts 6,3,8,8,5,8,5,3,3,5 pack1.webp pack2.webp
//
// See README.md for all flags.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/HugoSmits86/nativewebp"
	_ "golang.org/x/image/webp"
)

type background int

const (
	backgroundAuto background = iota
	backgroundNone
	backgroundFixed
)

type options struct {
	cellWidth, cellHeight int
	originX, originY      int
	columns, rows         int
	counts                []int
	bgMode                background
	bgColor               color.NRGBA
	tolerance             int
}

func main() {
	if err := run(os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "spritesheet:", err)
		os.Exit(1)
	}
}

func run(args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("spritesheet", flag.ContinueOnError)
	fs.SetOutput(stderr)
	output := fs.String("o", "", "output directory (files are named after it) or base path such as out/atlas (required)")
	cell := fs.String("cell", "", "frame cell size as WIDTHxHEIGHT (required)")
	origin := fs.String("origin", "0,0", "top-left of the first frame in each input as X,Y")
	columns := fs.Int("columns", 0, "frames per source row (default: as many as fit)")
	rows := fs.Int("rows", 0, "rows to read from each input (default: as many as fit)")
	counts := fs.String("counts", "", "comma-separated frame count for each output row (default: -columns)")
	bg := fs.String("bg", "auto", "background color to make transparent: auto (top-left pixel), none, or #RRGGBB")
	tolerance := fs.Int("tolerance", 12, "maximum per-channel difference from the background color")
	names := fs.String("names", "", "comma-separated animation name for each output row (default: row0, row1, ...)")
	meta := fs.String("meta", "auto", "metadata JSON path: auto (<base>.json), none, or a file path")
	formatList := fs.String("formats", "png,webp", "comma-separated image formats to write: png, webp")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: spritesheet -o OUTPUT -cell WxH [flags] input.webp...")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *output == "" || *cell == "" || fs.NArg() == 0 {
		fs.Usage()
		return errors.New("-o, -cell, and at least one input are required")
	}

	var opts options
	var err error
	if opts.cellWidth, opts.cellHeight, err = parsePair(*cell, "x"); err != nil {
		return fmt.Errorf("-cell: %w", err)
	}
	if opts.cellWidth <= 0 || opts.cellHeight <= 0 {
		return errors.New("-cell dimensions must be positive")
	}
	if opts.originX, opts.originY, err = parsePair(*origin, ","); err != nil {
		return fmt.Errorf("-origin: %w", err)
	}
	if *columns < 0 || *rows < 0 || *tolerance < 0 {
		return errors.New("-columns, -rows, and -tolerance must not be negative")
	}
	opts.columns, opts.rows, opts.tolerance = *columns, *rows, *tolerance
	if *counts != "" {
		if opts.counts, err = parseCounts(*counts); err != nil {
			return fmt.Errorf("-counts: %w", err)
		}
	}
	if opts.bgMode, opts.bgColor, err = parseBackground(*bg); err != nil {
		return fmt.Errorf("-bg: %w", err)
	}
	var rowNames []string
	if *names != "" {
		if rowNames, err = parseNames(*names); err != nil {
			return fmt.Errorf("-names: %w", err)
		}
	}
	formats, err := parseFormats(*formatList)
	if err != nil {
		return fmt.Errorf("-formats: %w", err)
	}
	base := outputBase(*output)

	sources := make([]image.Image, 0, fs.NArg())
	for _, path := range fs.Args() {
		img, err := decodeFile(path)
		if err != nil {
			return err
		}
		sources = append(sources, img)
	}
	atlas, frameCounts, err := buildAtlas(sources, opts)
	if err != nil {
		return err
	}
	imagePaths := make([]string, len(formats))
	for i, format := range formats {
		imagePaths[i] = base + "." + format
	}
	metaPath := *meta
	if metaPath == "auto" {
		metaPath = base + ".json"
	}
	var md metadata
	if metaPath != "none" {
		refs := make([]string, len(imagePaths))
		for i, path := range imagePaths {
			ref, err := filepath.Rel(filepath.Dir(metaPath), path)
			if err != nil {
				ref = path
			}
			refs[i] = filepath.ToSlash(ref)
		}
		if md, err = buildMetadata(atlas, refs, opts, frameCounts, rowNames); err != nil {
			return err
		}
	}
	if dir := filepath.Dir(base); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	b := atlas.Bounds()
	for _, path := range imagePaths {
		if err := writeImage(path, atlas); err != nil {
			return err
		}
		fmt.Fprintf(stderr, "wrote %s: %dx%d, %d rows of %dx%d cells\n",
			path, b.Dx(), b.Dy(), b.Dy()/opts.cellHeight, opts.cellWidth, opts.cellHeight)
	}
	if metaPath != "none" {
		if err := writeJSON(metaPath, md); err != nil {
			return err
		}
		fmt.Fprintf(stderr, "wrote %s: %d animations, %d frames\n", metaPath, len(md.Animations), len(md.Frames))
	}
	return nil
}

// outputBase turns -o into a path without extension. A directory (existing,
// or written with a trailing separator) yields <dir>/<dir name>; a path
// ending in .png, .webp, or .json has that extension removed.
func outputBase(output string) string {
	trimmed := strings.TrimRight(output, `/\`)
	if info, err := os.Stat(output); trimmed != output || (err == nil && info.IsDir()) {
		if trimmed == "" {
			trimmed = string(filepath.Separator)
		}
		return filepath.Join(trimmed, filepath.Base(filepath.Clean(trimmed)))
	}
	switch strings.ToLower(filepath.Ext(output)) {
	case ".png", ".webp", ".json":
		return strings.TrimSuffix(output, filepath.Ext(output))
	}
	return output
}

func parseFormats(value string) ([]string, error) {
	var formats []string
	seen := map[string]bool{}
	for _, part := range strings.Split(value, ",") {
		format := strings.ToLower(strings.TrimSpace(part))
		if format != "png" && format != "webp" {
			return nil, fmt.Errorf("unsupported format %q (want png or webp)", part)
		}
		if !seen[format] {
			seen[format] = true
			formats = append(formats, format)
		}
	}
	return formats, nil
}

type rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type frameMeta struct {
	Name      string `json:"name"`
	Animation string `json:"animation"`
	Index     int    `json:"index"`
	rect
	// Trim is the visible (non-transparent) area relative to the frame's
	// top-left corner; nil when the frame is empty.
	Trim *rect `json:"trim,omitempty"`
}

type animationMeta struct {
	Name   string   `json:"name"`
	Row    int      `json:"row"`
	Frames []string `json:"frames"`
}

type metadata struct {
	Image      string          `json:"image"`
	Images     []string        `json:"images"`
	Width      int             `json:"width"`
	Height     int             `json:"height"`
	CellWidth  int             `json:"cellWidth"`
	CellHeight int             `json:"cellHeight"`
	Animations []animationMeta `json:"animations"`
	Frames     []frameMeta     `json:"frames"`
}

// buildMetadata names every frame "<animation>_<index>" and records its
// location in the atlas. Rows without a name in names become "row<N>".
func buildMetadata(atlas *image.NRGBA, imageRefs []string, opts options, counts []int, names []string) (metadata, error) {
	if names != nil && len(names) != len(counts) {
		return metadata{}, fmt.Errorf("-names has %d entries but the atlas has %d rows", len(names), len(counts))
	}
	b := atlas.Bounds()
	md := metadata{
		Image: imageRefs[0], Images: imageRefs, Width: b.Dx(), Height: b.Dy(),
		CellWidth: opts.cellWidth, CellHeight: opts.cellHeight,
		Animations: []animationMeta{}, Frames: []frameMeta{},
	}
	seen := map[string]bool{}
	for row, count := range counts {
		name := fmt.Sprintf("row%d", row)
		if names != nil {
			name = names[row]
		}
		if seen[name] {
			return metadata{}, fmt.Errorf("-names: duplicate name %q", name)
		}
		seen[name] = true
		anim := animationMeta{Name: name, Row: row, Frames: []string{}}
		for i := 0; i < count; i++ {
			cell := rect{X: i * opts.cellWidth, Y: row * opts.cellHeight, W: opts.cellWidth, H: opts.cellHeight}
			frame := frameMeta{Name: fmt.Sprintf("%s_%d", name, i), Animation: name, Index: i, rect: cell}
			if v := visibleBounds(atlas, image.Rect(cell.X, cell.Y, cell.X+cell.W, cell.Y+cell.H)); !v.Empty() {
				frame.Trim = &rect{X: v.Min.X - cell.X, Y: v.Min.Y - cell.Y, W: v.Dx(), H: v.Dy()}
			}
			anim.Frames = append(anim.Frames, frame.Name)
			md.Frames = append(md.Frames, frame)
		}
		md.Animations = append(md.Animations, anim)
	}
	return md, nil
}

func visibleBounds(img *image.NRGBA, area image.Rectangle) image.Rectangle {
	var v image.Rectangle
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			if img.NRGBAAt(x, y).A != 0 {
				v = v.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return v
}

func parseNames(value string) ([]string, error) {
	parts := strings.Split(value, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
		if parts[i] == "" {
			return nil, errors.New("names must not be empty")
		}
	}
	return parts, nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

type sourceRow struct {
	img     image.Image
	y       int
	columns int
	keyed   bool
	bg      color.NRGBA
}

func buildAtlas(sources []image.Image, opts options) (*image.NRGBA, []int, error) {
	var rows []sourceRow
	for i, src := range sources {
		b := src.Bounds()
		columns := opts.columns
		if columns == 0 {
			columns = (b.Dx() - opts.originX) / opts.cellWidth
		}
		rowCount := opts.rows
		if rowCount == 0 {
			rowCount = (b.Dy() - opts.originY) / opts.cellHeight
		}
		if columns <= 0 || rowCount <= 0 {
			return nil, nil, fmt.Errorf("input %d (%dx%d) has no complete %dx%d cell after origin %d,%d",
				i+1, b.Dx(), b.Dy(), opts.cellWidth, opts.cellHeight, opts.originX, opts.originY)
		}
		row := sourceRow{img: src, columns: columns, keyed: opts.bgMode != backgroundNone, bg: opts.bgColor}
		if opts.bgMode == backgroundAuto {
			row.bg = color.NRGBAModel.Convert(src.At(b.Min.X, b.Min.Y)).(color.NRGBA)
		}
		for r := 0; r < rowCount; r++ {
			row.y = opts.originY + r*opts.cellHeight
			rows = append(rows, row)
		}
	}

	counts := opts.counts
	if counts == nil {
		counts = make([]int, len(rows))
		for i, row := range rows {
			counts[i] = row.columns
		}
	}
	if len(counts) != len(rows) {
		return nil, nil, fmt.Errorf("-counts has %d entries but the inputs provide %d rows", len(counts), len(rows))
	}
	atlasColumns := 0
	for _, count := range counts {
		atlasColumns = max(atlasColumns, count)
	}
	if atlasColumns == 0 {
		return nil, nil, errors.New("every row has zero frames")
	}

	atlas := image.NewNRGBA(image.Rect(0, 0, atlasColumns*opts.cellWidth, len(rows)*opts.cellHeight))
	for r, row := range rows {
		b := row.img.Bounds()
		for frame := 0; frame < counts[r]; frame++ {
			srcX := b.Min.X + opts.originX + frame*opts.cellWidth
			srcY := b.Min.Y + row.y
			for y := 0; y < opts.cellHeight; y++ {
				for x := 0; x < opts.cellWidth; x++ {
					p := image.Pt(srcX+x, srcY+y)
					if !p.In(b) {
						continue
					}
					c := color.NRGBAModel.Convert(row.img.At(p.X, p.Y)).(color.NRGBA)
					if c.A == 0 || (row.keyed && matches(c, row.bg, opts.tolerance)) {
						continue
					}
					atlas.SetNRGBA(frame*opts.cellWidth+x, r*opts.cellHeight+y, c)
				}
			}
		}
	}
	return atlas, counts, nil
}

func matches(c, bg color.NRGBA, tolerance int) bool {
	return absDiff(c.R, bg.R) <= tolerance && absDiff(c.G, bg.G) <= tolerance && absDiff(c.B, bg.B) <= tolerance
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

func parsePair(value, separator string) (int, int, error) {
	first, second, ok := strings.Cut(strings.ToLower(value), separator)
	if !ok {
		return 0, 0, fmt.Errorf("expected two numbers separated by %q, got %q", separator, value)
	}
	a, err := strconv.Atoi(strings.TrimSpace(first))
	if err != nil {
		return 0, 0, err
	}
	b, err := strconv.Atoi(strings.TrimSpace(second))
	if err != nil {
		return 0, 0, err
	}
	if a < 0 || b < 0 {
		return 0, 0, fmt.Errorf("values must not be negative, got %q", value)
	}
	return a, b, nil
}

func parseCounts(value string) ([]int, error) {
	parts := strings.Split(value, ",")
	counts := make([]int, len(parts))
	for i, part := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		if n < 0 {
			return nil, fmt.Errorf("count %d must not be negative", n)
		}
		counts[i] = n
	}
	return counts, nil
}

func parseBackground(value string) (background, color.NRGBA, error) {
	switch strings.ToLower(value) {
	case "auto":
		return backgroundAuto, color.NRGBA{}, nil
	case "none":
		return backgroundNone, color.NRGBA{}, nil
	}
	hex := strings.TrimPrefix(value, "#")
	n, err := strconv.ParseUint(hex, 16, 32)
	if len(hex) != 6 || err != nil {
		return 0, color.NRGBA{}, fmt.Errorf("expected auto, none, or #RRGGBB, got %q", value)
	}
	return backgroundFixed, color.NRGBA{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n), A: 255}, nil
}

func decodeFile(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return img, nil
}

// writeImage encodes img as PNG or lossless WebP based on the path extension.
func writeImage(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if strings.EqualFold(filepath.Ext(path), ".webp") {
		err = nativewebp.Encode(f, img, nil)
	} else {
		err = png.Encode(f, img)
	}
	if err != nil {
		f.Close()
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return f.Close()
}
