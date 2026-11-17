package acornrun

import (
	"bytes"
	"encoding/binary"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"github.com/ebitengine/oto/v3"
	"github.com/xdung24/games/internal/audio"
)

const soundSampleRate = 44100

type soundPlayer struct {
	ctx     *oto.Context
	muted   atomic.Bool
	samples map[Sound][]byte
	mu      sync.Mutex
	active  []*oto.Player
}

func newSoundPlayer() *soundPlayer {
	player := &soundPlayer{samples: buildSounds()}
	context, err := audio.Context()
	if err != nil {
		fyne.LogError("Acorn Run sound is unavailable", err)
		return player
	}
	player.ctx = context
	return player
}

func (s *soundPlayer) SetMuted(muted bool) {
	if s != nil {
		s.muted.Store(muted)
	}
}

func (s *soundPlayer) Muted() bool {
	return s != nil && s.muted.Load()
}

func (s *soundPlayer) Play(event Sound) {
	if s == nil || s.ctx == nil || s.muted.Load() {
		return
	}
	data, ok := s.samples[event]
	if !ok {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	live := s.active[:0]
	for _, player := range s.active {
		if player.IsPlaying() {
			live = append(live, player)
		} else if err := player.Close(); err != nil {
			fyne.LogError("Unable to close Acorn Run sound player", err)
		}
	}
	s.active = live

	player := s.ctx.NewPlayer(bytes.NewReader(data))
	player.Play()
	s.active = append(s.active, player)
}

type tone struct {
	start    float64
	end      float64
	duration time.Duration
	volume   float64
}

func buildSounds() map[Sound][]byte {
	return map[Sound][]byte{
		SoundJump:  renderTone(tone{start: 520, end: 760, duration: 75 * time.Millisecond, volume: 0.24}),
		SoundAcorn: renderTone(tone{start: 760, end: 1180, duration: 110 * time.Millisecond, volume: 0.25}),
		SoundStomp: renderTone(tone{start: 190, end: 90, duration: 110 * time.Millisecond, volume: 0.32}),
		SoundHit:   renderTone(tone{start: 260, end: 70, duration: 260 * time.Millisecond, volume: 0.35}),
		SoundGoal:  renderTone(tone{start: 520, end: 1040, duration: 330 * time.Millisecond, volume: 0.28}),
	}
}

func renderTone(t tone) []byte {
	count := int(float64(soundSampleRate) * t.duration.Seconds())
	out := make([]byte, count*2)
	phase := 0.0
	for i := 0; i < count; i++ {
		progress := float64(i) / float64(count)
		frequency := t.start + (t.end-t.start)*progress
		phase += 2 * math.Pi * frequency / soundSampleRate
		envelope := math.Min(float64(i)/220, float64(count-i)/880)
		if envelope > 1 {
			envelope = 1
		}
		sample := int16(math.Sin(phase) * envelope * t.volume * math.MaxInt16)
		binary.LittleEndian.PutUint16(out[i*2:], uint16(sample))
	}
	return out
}
