package flappygophy

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

var (
	gopherRest = gopherResource(false)
	gopherFlap = gopherResource(true)
	cloud      = fyne.NewStaticResource("gophy-cloud.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="96" height="72" viewBox="0 0 96 72">
<g transform="translate(48 36)" fill="#f5faff" fill-opacity="0.862745">
<circle r="32"/><circle cx="-21.333" cy="1.6" r="21.333"/>
<circle cx="21.333" cy="1.6" r="21.333"/>
<circle cx="-10.667" cy="18.667" r="12.8"/><circle cx="10.667" cy="18.667" r="12.8"/>
</g></svg>`))
)

type gameScene struct {
	widget.BaseWidget
	game    *Game
	sky     [2]*canvas.LinearGradient
	ground  *canvas.Raster
	clouds  [numClouds]*canvas.Image
	pipes   [3][4]pipeSprite
	gopher  [2]*canvas.Image
	objects []fyne.CanvasObject
}

type pipeSprite struct {
	edge *canvas.Rectangle
	fill *canvas.LinearGradient
}

func (p pipeSprite) place(x, y, w, h, border float32) {
	placeScene(p.edge, x, y, w, h)
	placeScene(p.fill, x+border, y, w-2*border, h)
}

func newGameScene(g *Game) *gameScene {
	s := &gameScene{game: g}
	s.ExtendBaseWidget(s)
	s.sky[0] = canvas.NewLinearGradient(
		color.RGBA{R: 185, G: 225, B: 250, A: 255},
		color.RGBA{R: 135, G: 206, B: 235, A: 255}, 0)
	s.sky[1] = canvas.NewLinearGradient(
		color.RGBA{R: 135, G: 206, B: 235, A: 255},
		color.RGBA{R: 80, G: 160, B: 210, A: 255}, 0)
	s.objects = append(s.objects, s.sky[0], s.sky[1])
	for i := range s.clouds {
		s.clouds[i] = canvas.NewImageFromResource(cloud)
		s.objects = append(s.objects, s.clouds[i])
	}
	s.ground = canvas.NewRaster(func(w, h int) image.Image {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		drawGround(img, 0, w, h)
		return img
	})
	s.objects = append(s.objects, s.ground)
	for i := range s.pipes {
		for j := range s.pipes[i] {
			edge := color.RGBA{R: 50, G: 140, B: 45, A: 255}
			fill := color.RGBA{R: 80, G: 195, B: 78, A: 255}
			if j%2 == 1 {
				edge = color.RGBA{R: 40, G: 125, B: 38, A: 255}
				fill = color.RGBA{R: 65, G: 178, B: 63, A: 255}
			}
			part := pipeSprite{
				edge: canvas.NewRectangle(edge),
				fill: canvas.NewLinearGradient(color.RGBA{R: 165, G: 235, B: 135, A: 255}, fill, 270),
			}
			part.edge.Hide()
			part.fill.Hide()
			s.pipes[i][j] = part
			s.objects = append(s.objects, part.edge, part.fill)
		}
	}
	for i, resource := range []fyne.Resource{gopherRest, gopherFlap} {
		s.gopher[i] = canvas.NewImageFromResource(resource)
		s.objects = append(s.objects, s.gopher[i])
	}
	return s
}

func (s *gameScene) CreateRenderer() fyne.WidgetRenderer {
	return &sceneRenderer{scene: s}
}

func placeScene(o fyne.CanvasObject, x, y, w, h float32) {
	o.Move(fyne.NewPos(x, y))
	o.Resize(fyne.NewSize(w, h))
}

func (s *gameScene) refresh(now time.Time) {
	size := s.Size()
	if size.Width <= 0 || size.Height <= 0 {
		return
	}
	sx, sy := size.Width/gameW, size.Height/gameH
	scale := min(sx, sy)
	groundY := (gameH - groundH) * sy
	placeScene(s.sky[0], 0, 0, size.Width, groundY*0.15)
	placeScene(s.sky[1], 0, groundY*0.15, size.Width, groundY*0.85)
	placeScene(s.ground, 0, groundY, size.Width, size.Height-groundY)

	const cloudMargin = 150
	const span = gameW + 2*cloudMargin
	seconds := float64(now.UnixMilli()) / 1000
	for i, spec := range cloudDefs {
		x := spec.baseX - float32(math.Mod(seconds*float64(spec.speed), span))
		if x < -cloudMargin {
			x += span
		}
		radius := spec.radius * scale
		placeScene(s.clouds[i], x*sx-radius*1.5,
			(spec.baseY+gameH*0.08)*sy-radius*1.125, radius*3, radius*2.25)
	}

	g := s.game
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, parts := range s.pipes {
		if i >= len(g.pipes) {
			for _, part := range parts {
				part.edge.Hide()
				part.fill.Hide()
			}
			continue
		}
		p := g.pipes[i]
		x := float32(p.x) * sx
		top := float32(p.gapY-pipeGap/2) * sy
		bottom := float32(p.gapY+pipeGap/2) * sy
		capHeight := 14 * sy
		parts[0].place(x, 0, pipeW*sx, top-capHeight, sx)
		parts[1].place(x-5*sx, top-capHeight, (pipeW+10)*sx, capHeight, sx)
		parts[2].place(x, bottom+capHeight, pipeW*sx, groundY-bottom-capHeight, sx)
		parts[3].place(x-5*sx, bottom, (pipeW+10)*sx, capHeight, sx)
		for _, part := range parts {
			part.edge.Show()
			part.fill.Show()
		}
	}

	x, y, radius := float32(birdX)*sx, float32(g.birdY)*sy, float32(birdRadius)*scale
	if g.state == StateStart {
		x, y, radius = size.Width/2, size.Height*0.30, 38*scale
	}
	flapping := g.flapTimer > 0 && g.state != StateStart
	for i, pose := range s.gopher {
		placeScene(pose, x-radius*1.6, y-radius*1.6, radius*3.2, radius*3.2)
		if (i == 1) == flapping {
			pose.Show()
		} else {
			pose.Hide()
		}
	}
}

type sceneRenderer struct {
	scene *gameScene
}

func (r *sceneRenderer) Layout(fyne.Size)             { r.scene.refresh(time.Now()) }
func (r *sceneRenderer) MinSize() fyne.Size           { return fyne.NewSize(1, 1) }
func (r *sceneRenderer) Objects() []fyne.CanvasObject { return r.scene.objects }
func (r *sceneRenderer) Refresh()                     { r.scene.refresh(time.Now()) }
func (r *sceneRenderer) Destroy()                     {}

func gopherResource(flapping bool) fyne.Resource {
	frontY, backY, pupilOffset, direction := 15.0, 12.0, 0.0, 1
	if flapping {
		frontY, backY, pupilOffset, direction = -15, -12, -3, -1
	}
	paw := func(x, y float64) string {
		return fmt.Sprintf(`<g fill="#486984" transform="translate(%g %g) scale(1 %d)">
<circle r="6.667"/><circle cx="-3.333" cy="5.556" r="2.222"/>
<circle cy="6.667" r="2.222"/><circle cx="3.333" cy="5.556" r="2.222"/></g>`, x, y, direction)
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64"><g transform="translate(32 32)">
%s<circle r="20" fill="#6388a8"/>%s
<circle cx="12" cy="3.333" r="8" fill="#b9d2dc"/>
<circle cx="14.667" cy="0.667" r="2.857" fill="#0f0f0f"/>
<g fill="white"><rect x="7" y="7.333" width="4" height="8"/>
<rect x="13" y="7.333" width="4" height="8"/><circle cx="5" cy="-5" r="6.667"/></g>
<circle cx="7.5" cy="%g" r="4" fill="#0f0f0f"/>
<circle cx="%g" cy="%g" r="2.5" fill="white"/></g></svg>`,
		paw(6.667, frontY), paw(-13.333, backY), -5+pupilOffset/2, 10.5+pupilOffset, -5+pupilOffset)
	return fyne.NewStaticResource(fmt.Sprintf("gophy-flap-%t.svg", flapping), []byte(svg))
}
