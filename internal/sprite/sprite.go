// Package sprite loads sprite sheets produced by cmd/spritesheet (an image
// plus a JSON file naming and locating every frame) and draws their frames.
package sprite

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"io/fs"
	"math"
	"path"

	_ "golang.org/x/image/webp"
)

// Rect is a rectangle in sheet-image pixels.
type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

func (r Rect) bounds() image.Rectangle { return image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H) }

// Frame is one named sprite cell.
type Frame struct {
	Name      string `json:"name"`
	Animation string `json:"animation"`
	Index     int    `json:"index"`
	Rect
	// Trim is the visible area relative to the cell; nil for empty frames.
	Trim *Rect `json:"trim,omitempty"`
}

// Bounds returns the frame's cell in the sheet image.
func (f Frame) Bounds() image.Rectangle { return f.Rect.bounds() }

// Visible returns the frame's non-transparent area in the sheet image, or an
// empty rectangle when the frame has no visible pixels.
func (f Frame) Visible() image.Rectangle {
	if f.Trim == nil {
		return image.Rectangle{}
	}
	return f.Trim.bounds().Add(image.Pt(f.X, f.Y))
}

// BottomPadding is the number of transparent rows below the visible pixels.
// Use it to stand sprites on a floor line regardless of cell padding.
func (f Frame) BottomPadding() int {
	if f.Trim == nil {
		return 0
	}
	return f.H - f.Trim.Y - f.Trim.H
}

type metadata struct {
	Image      string   `json:"image"`
	Images     []string `json:"images"`
	Animations []struct {
		Name   string   `json:"name"`
		Frames []string `json:"frames"`
	} `json:"animations"`
	Frames []Frame `json:"frames"`
}

// Sheet is a decoded sprite image with its frame metadata.
type Sheet struct {
	img        *image.NRGBA
	frames     map[string]Frame
	animations map[string][]Frame
}

// Load reads the metadata JSON at metaPath from fsys and decodes the first of
// its listed images that exists in fsys, so a sheet can ship with only the
// WebP or only the PNG.
func Load(fsys fs.FS, metaPath string) (*Sheet, error) {
	data, err := fs.ReadFile(fsys, metaPath)
	if err != nil {
		return nil, err
	}
	var md metadata
	if err := json.Unmarshal(data, &md); err != nil {
		return nil, fmt.Errorf("parse %s: %w", metaPath, err)
	}
	refs := md.Images
	if len(refs) == 0 && md.Image != "" {
		refs = []string{md.Image}
	}
	img, err := decodeFirst(fsys, path.Dir(metaPath), refs)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", metaPath, err)
	}

	s := &Sheet{img: img, frames: map[string]Frame{}, animations: map[string][]Frame{}}
	for _, f := range md.Frames {
		if !f.Bounds().In(img.Bounds()) {
			return nil, fmt.Errorf("%s: frame %q %v is outside the %v image", metaPath, f.Name, f.Bounds(), img.Bounds())
		}
		s.frames[f.Name] = f
	}
	for _, a := range md.Animations {
		frames := make([]Frame, len(a.Frames))
		for i, name := range a.Frames {
			f, ok := s.frames[name]
			if !ok {
				return nil, fmt.Errorf("%s: animation %q references unknown frame %q", metaPath, a.Name, name)
			}
			frames[i] = f
		}
		s.animations[a.Name] = frames
	}
	return s, nil
}

func decodeFirst(fsys fs.FS, dir string, refs []string) (*image.NRGBA, error) {
	for _, ref := range refs {
		f, err := fsys.Open(path.Join(dir, ref))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		src, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", ref, err)
		}
		if img, ok := src.(*image.NRGBA); ok {
			return img, nil
		}
		img := image.NewNRGBA(src.Bounds())
		draw.Draw(img, img.Bounds(), src, src.Bounds().Min, draw.Src)
		return img, nil
	}
	return nil, fmt.Errorf("none of the listed images %q exist", refs)
}

// Image returns the decoded sheet image.
func (s *Sheet) Image() image.Image { return s.img }

// Frame returns the frame with the given name, e.g. "walk_3".
func (s *Sheet) Frame(name string) (Frame, bool) {
	f, ok := s.frames[name]
	return f, ok
}

// Animation returns the ordered frames of the named animation.
func (s *Sheet) Animation(name string) []Frame { return s.animations[name] }

// Clip is a timed sequence of frames.
type Clip struct {
	Frames    []Frame
	FrameTime float32 // seconds per frame
	Loop      bool
}

// Clip builds a clip from count frames of the named animation starting at
// first. A count of 0 takes every frame from first to the end.
func (s *Sheet) Clip(animation string, first, count int, frameTime float32, loop bool) (Clip, error) {
	frames, ok := s.animations[animation]
	if !ok {
		return Clip{}, fmt.Errorf("unknown animation %q", animation)
	}
	if count == 0 {
		count = len(frames) - first
	}
	if first < 0 || count <= 0 || first+count > len(frames) {
		return Clip{}, fmt.Errorf("animation %q has %d frames; cannot take %d from %d", animation, len(frames), count, first)
	}
	if frameTime <= 0 {
		return Clip{}, fmt.Errorf("animation %q: frame time must be positive", animation)
	}
	return Clip{Frames: frames[first : first+count], FrameTime: frameTime, Loop: loop}, nil
}

// Index returns the frame index shown after elapsed seconds. Looping clips
// wrap around; others hold their last frame.
func (c Clip) Index(elapsed float32) int {
	i := max(0, int(elapsed/c.FrameTime))
	if c.Loop {
		return i % len(c.Frames)
	}
	return min(i, len(c.Frames)-1)
}

// At returns the frame shown after elapsed seconds.
func (c Clip) At(elapsed float32) Frame { return c.Frames[c.Index(elapsed)] }

// Draw alpha-blends frame f, scaled with nearest-neighbour sampling, into the
// destination rectangle at (x, y) of size w×h in dst pixels. flipX mirrors it
// horizontally, e.g. for a left-facing character.
func (s *Sheet) Draw(dst *image.RGBA, f Frame, x, y, w, h float32, flipX bool) {
	left := int(math.Floor(float64(x)))
	top := int(math.Floor(float64(y)))
	right := int(math.Ceil(float64(x + w)))
	bottom := int(math.Ceil(float64(y + h)))
	target := image.Rect(left, top, right, bottom).Intersect(dst.Bounds())
	if target.Empty() || f.W <= 0 || f.H <= 0 {
		return
	}
	src := f.Bounds()
	for py := target.Min.Y; py < target.Max.Y; py++ {
		sy := src.Min.Y + (py-top)*src.Dy()/(bottom-top)
		for px := target.Min.X; px < target.Max.X; px++ {
			offset := ((px-left)*2 + 1) * src.Dx() / (2 * (right - left))
			if flipX {
				offset = src.Dx() - 1 - offset
			}
			c := s.img.NRGBAAt(src.Min.X+offset, sy)
			if c.A == 0 {
				continue
			}
			dst.SetRGBA(px, py, blend(dst.RGBAAt(px, py), c))
		}
	}
}

func blend(bg color.RGBA, c color.NRGBA) color.RGBA {
	a := uint32(c.A)
	inv := 255 - a
	return color.RGBA{
		R: uint8((uint32(c.R)*a + uint32(bg.R)*inv) / 255),
		G: uint8((uint32(c.G)*a + uint32(bg.G)*inv) / 255),
		B: uint8((uint32(c.B)*a + uint32(bg.B)*inv) / 255),
		A: uint8(a + uint32(bg.A)*inv/255),
	}
}
