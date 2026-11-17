package main

import (
	"image"
	"image/color"
	"path/filepath"
	"testing"
)

var gray = color.NRGBA{R: 162, G: 162, B: 162, A: 255}

// sheet returns a w×h gray image with a colored pixel at each given point.
func sheet(w, h int, marks map[image.Point]color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, gray)
		}
	}
	for p, c := range marks {
		img.SetNRGBA(p.X, p.Y, c)
	}
	return img
}

func TestBuildAtlasKeysBackgroundAndStacksInputs(t *testing.T) {
	red := color.NRGBA{R: 200, A: 255}
	blue := color.NRGBA{B: 200, A: 255}
	near := color.NRGBA{R: 170, G: 160, B: 165, A: 255}
	// Origin 2,0 skips a label column; cells are 4×4.
	first := sheet(14, 8, map[image.Point]color.NRGBA{
		{0, 1}: red,               // label area, must be cropped
		{3, 1}: red, {7, 5}: near, // row 0 frame 0; row 1 frame 1 (keyed)
	})
	second := sheet(14, 4, map[image.Point]color.NRGBA{{11, 2}: blue}) // row 0 frame 2

	atlas, _, err := buildAtlas([]image.Image{first, second}, options{
		cellWidth: 4, cellHeight: 4, originX: 2, tolerance: 12,
		counts: []int{1, 2, 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := atlas.Bounds().Size(); got != image.Pt(12, 12) {
		t.Fatalf("atlas size = %v, want 12x12", got)
	}
	if got := atlas.NRGBAAt(1, 1); got != red {
		t.Errorf("first frame pixel = %v, want %v", got, red)
	}
	if got := atlas.NRGBAAt(9, 10); got != blue {
		t.Errorf("second input pixel = %v, want %v", got, blue)
	}
	for y := 0; y < 12; y++ {
		for x := 0; x < 12; x++ {
			if p := image.Pt(x, y); p != image.Pt(1, 1) && p != image.Pt(9, 10) && atlas.NRGBAAt(x, y).A != 0 {
				t.Errorf("pixel %v = %v, want transparent", p, atlas.NRGBAAt(x, y))
			}
		}
	}
}

func TestBuildAtlasRejectsMismatchedCounts(t *testing.T) {
	_, _, err := buildAtlas([]image.Image{sheet(8, 8, nil)}, options{
		cellWidth: 4, cellHeight: 4, counts: []int{2},
	})
	if err == nil {
		t.Fatal("expected error for 1 count with 2 rows")
	}
}

func TestBuildMetadataNamesAndLocatesFrames(t *testing.T) {
	red := color.NRGBA{R: 200, A: 255}
	opts := options{cellWidth: 4, cellHeight: 4, tolerance: 12}
	atlas, counts, err := buildAtlas([]image.Image{
		sheet(8, 8, map[image.Point]color.NRGBA{{5, 6}: red, {6, 7}: red}),
	}, opts)
	if err != nil {
		t.Fatal(err)
	}
	md, err := buildMetadata(atlas, []string{"atlas.png"}, opts, counts, []string{"idle", "walk"})
	if err != nil {
		t.Fatal(err)
	}
	if len(md.Frames) != 4 || md.Animations[1].Name != "walk" || md.Animations[1].Frames[1] != "walk_1" {
		t.Fatalf("unexpected metadata: %+v", md)
	}
	walk1 := md.Frames[3]
	if walk1.Name != "walk_1" || walk1.rect != (rect{X: 4, Y: 4, W: 4, H: 4}) {
		t.Errorf("walk_1 = %+v, want cell at 4,4 size 4x4", walk1)
	}
	if walk1.Trim == nil || *walk1.Trim != (rect{X: 1, Y: 2, W: 2, H: 2}) {
		t.Errorf("walk_1 trim = %+v, want 1,2 2x2", walk1.Trim)
	}
	if md.Frames[0].Trim != nil {
		t.Errorf("empty frame trim = %+v, want nil", md.Frames[0].Trim)
	}

	if _, err := buildMetadata(atlas, []string{"atlas.png"}, opts, counts, []string{"idle"}); err == nil {
		t.Error("expected error for too few names")
	}
	if _, err := buildMetadata(atlas, []string{"atlas.png"}, opts, counts, []string{"a", "a"}); err == nil {
		t.Error("expected error for duplicate names")
	}
}

func TestOutputBase(t *testing.T) {
	dir := t.TempDir()
	sep := string(filepath.Separator)
	for _, tc := range []struct{ in, want string }{
		{dir, filepath.Join(dir, filepath.Base(dir))},
		{filepath.Join("out", "hero") + "/", filepath.Join("out", "hero", "hero")},
		{filepath.Join("out", "hero") + sep, filepath.Join("out", "hero", "hero")},
		{filepath.Join("out", "atlas.png"), filepath.Join("out", "atlas")},
		{filepath.Join("out", "atlas.WEBP"), filepath.Join("out", "atlas")},
		{filepath.Join("out", "atlas"), filepath.Join("out", "atlas")},
	} {
		if got := outputBase(tc.in); got != tc.want {
			t.Errorf("outputBase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestBuildAtlasWithoutKeyingKeepsBackground(t *testing.T) {
	atlas, _, err := buildAtlas([]image.Image{sheet(4, 4, nil)}, options{
		cellWidth: 4, cellHeight: 4, bgMode: backgroundNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := atlas.NRGBAAt(0, 0); got != gray {
		t.Fatalf("pixel = %v, want %v", got, gray)
	}
}
