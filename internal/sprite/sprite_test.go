package sprite

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
	"testing/fstest"
)

var red = color.NRGBA{R: 255, A: 255}

// testSheet is a 2-frame "walk" animation in 4×4 cells. walk_1 has a single
// visible pixel at (1,2), i.e. one transparent row below it.
func testSheet(t *testing.T) *Sheet {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 8, 4))
	img.SetNRGBA(0, 0, red)
	img.SetNRGBA(5, 2, red)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{
		"art/hero.json": {Data: []byte(`{
			"image": "hero.webp", "images": ["hero.webp", "hero.png"],
			"animations": [{"name": "walk", "row": 0, "frames": ["walk_0", "walk_1"]}],
			"frames": [
				{"name": "walk_0", "animation": "walk", "index": 0, "x": 0, "y": 0, "w": 4, "h": 4, "trim": {"x": 0, "y": 0, "w": 1, "h": 1}},
				{"name": "walk_1", "animation": "walk", "index": 1, "x": 4, "y": 0, "w": 4, "h": 4, "trim": {"x": 1, "y": 2, "w": 1, "h": 1}}
			]}`)},
		"art/hero.png": {Data: buf.Bytes()},
	}
	sheet, err := Load(fsys, "art/hero.json")
	if err != nil {
		t.Fatal(err)
	}
	return sheet
}

func TestLoadFallsBackToAvailableImageAndIndexesFrames(t *testing.T) {
	sheet := testSheet(t)
	f, ok := sheet.Frame("walk_1")
	if !ok || f.Bounds() != image.Rect(4, 0, 8, 4) || f.Visible() != image.Rect(5, 2, 6, 3) || f.BottomPadding() != 1 {
		t.Fatalf("walk_1 = %+v (ok=%v)", f, ok)
	}
	if got := sheet.Animation("walk"); len(got) != 2 || got[1].Name != "walk_1" {
		t.Fatalf("walk animation = %+v", got)
	}
}

func TestLoadRejectsUnknownFrameReference(t *testing.T) {
	fsys := fstest.MapFS{
		"a.json": {Data: []byte(`{"images": ["a.png"], "animations": [{"name": "x", "frames": ["missing"]}], "frames": []}`)},
		"a.png":  {Data: encodePNG(t, image.NewNRGBA(image.Rect(0, 0, 1, 1)))},
	}
	if _, err := Load(fsys, "a.json"); err == nil {
		t.Fatal("expected error for unknown frame")
	}
}

func TestClipLoopsOrHoldsLastFrame(t *testing.T) {
	sheet := testSheet(t)
	loop, err := sheet.Clip("walk", 0, 0, 0.1, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := loop.At(0.25).Name; got != "walk_0" {
		t.Errorf("looping clip at 0.25s = %s, want walk_0", got)
	}
	once, _ := sheet.Clip("walk", 0, 2, 0.1, false)
	if got := once.At(5).Name; got != "walk_1" {
		t.Errorf("one-shot clip at 5s = %s, want walk_1", got)
	}
	if _, err := sheet.Clip("walk", 1, 2, 0.1, false); err == nil {
		t.Error("expected error for clip past the end of the animation")
	}
	if _, err := sheet.Clip("run", 0, 0, 0.1, false); err == nil {
		t.Error("expected error for unknown animation")
	}
}

func TestDrawScalesAndMirrors(t *testing.T) {
	sheet := testSheet(t)
	f, _ := sheet.Frame("walk_1")
	right := image.NewRGBA(image.Rect(0, 0, 8, 8))
	left := image.NewRGBA(image.Rect(0, 0, 8, 8))
	sheet.Draw(right, f, 0, 0, 8, 8, false)
	sheet.Draw(left, f, 0, 0, 8, 8, true)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			want := y >= 4 && y < 6 && x >= 2 && x < 4
			if got := right.RGBAAt(x, y).A != 0; got != want {
				t.Errorf("right (%d,%d) drawn = %v, want %v", x, y, got, want)
			}
			if left.RGBAAt(x, y) != right.RGBAAt(7-x, y) {
				t.Errorf("left (%d,%d) is not a mirror of right", x, y)
			}
		}
	}
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
