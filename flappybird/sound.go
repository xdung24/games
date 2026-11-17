package flappybird

import (
	"bytes"
	"encoding/binary"
	"math"

	"fyne.io/fyne/v2"
	"github.com/ebitengine/oto/v3"
	"github.com/xdung24/game/internal/audio"
)

type soundPlayer struct {
	ctx    *oto.Context
	muted  bool
	pcm    [3][]byte
	active []*oto.Player
}

func newSoundPlayer() *soundPlayer {
	s := &soundPlayer{}
	for i, pitch := range []float64{680, 1040, 160} {
		n := 4410
		if i == int(SoundHit) {
			n *= 2
		}
		s.pcm[i] = make([]byte, n*2)
		phase := 0.0
		for j := 0; j < n; j++ {
			t := float64(j) / float64(n)
			phase += pitch * (1 - 0.5*t) / 44100
			envelope := math.Min(float64(j)/220, 1) * (1 - t)
			sample := int16(math.Sin(phase*2*math.Pi) * envelope * 7000)
			binary.LittleEndian.PutUint16(s.pcm[i][j*2:], uint16(sample))
		}
	}
	ctx, err := audio.Context()
	if err != nil {
		fyne.LogError("Flappy Bird audio is unavailable", err)
		return s
	}
	s.ctx = ctx
	return s
}

func (s *soundPlayer) Play(sound Sound) {
	if s.ctx == nil || s.muted {
		return
	}
	live := s.active[:0]
	for _, p := range s.active {
		if p.IsPlaying() {
			live = append(live, p)
		} else if err := p.Close(); err != nil {
			fyne.LogError("Closing Flappy Bird sound", err)
		}
	}
	s.active = live
	p := s.ctx.NewPlayer(bytes.NewReader(s.pcm[sound]))
	p.Play()
	s.active = append(s.active, p)
}
