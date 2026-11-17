package croakcross

import (
	"bytes"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/oto/v3"
)

// Audio format for the synthesized blips: CD-quality mono, 16-bit signed.
const (
	sampleRate = 44100
	channels   = 1

	// deviceBufferSize caps how much audio the output device keeps queued ahead.
	// Smaller means more immediate feedback - but large enough old hardware doesn't glitch.
	deviceBufferSize = 30 * time.Millisecond
)

// soundPlayer renders short retro square-wave blips for game events. It owns a
// single oto context and a pool of in-flight players. If the audio device can't
// be opened the player simply disables itself and every Play call is a no-op, so
// the game runs fine on machines with no sound.
type soundPlayer struct {
	ctx     *oto.Context
	enabled bool
	muted   atomic.Bool // when set, every Play is a no-op
	pcm     map[Sound][]byte

	mu     sync.Mutex
	active []*oto.Player
}

// SetMuted silences (or unsilences) the player. Safe to call from any goroutine.
func (sp *soundPlayer) SetMuted(muted bool) {
	if sp != nil {
		sp.muted.Store(muted)
	}
}

// Muted reports whether the player is currently silenced.
func (sp *soundPlayer) Muted() bool {
	return sp != nil && sp.muted.Load()
}

// newSoundPlayer opens the audio device and pre-renders every effect. It returns
// a usable (possibly disabled) player and never an error — silence is an
// acceptable degraded mode.
func newSoundPlayer() *soundPlayer {
	sp := &soundPlayer{pcm: buildSounds()}

	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: channels,
		Format:       oto.FormatSignedInt16LE,
		BufferSize:   deviceBufferSize,
	})
	if err != nil {
		return sp // disabled: no audio device
	}
	sp.ctx = ctx

	// Wait briefly for the device to come up; if it stalls, give up on sound
	// rather than holding the whole game's startup hostage.
	select {
	case <-ready:
		sp.enabled = true
	case <-time.After(2 * time.Second):
	}
	return sp
}

// Play renders the requested effect. Safe to call from any goroutine.
func (sp *soundPlayer) Play(s Sound) {
	if sp == nil || !sp.enabled || sp.muted.Load() {
		return
	}
	buf, ok := sp.pcm[s]
	if !ok {
		return
	}

	sp.mu.Lock()
	defer sp.mu.Unlock()

	// Drop players that have finished so the pool can't grow without bound.
	live := sp.active[:0]
	for _, p := range sp.active {
		if p.IsPlaying() {
			live = append(live, p)
		} else {
			_ = p.Close()
		}
	}
	sp.active = live

	p := sp.ctx.NewPlayer(bytes.NewReader(buf))
	p.Play()
	sp.active = append(sp.active, p)
}

// --- synthesis ---

// segment is one freq-swept square-wave tone.
type segment struct {
	fStart, fEnd float64 // hertz, linearly swept across the segment
	dur          time.Duration
	vol          float64 // 0..1 peak amplitude
}

// buildSounds pre-renders the PCM for every game event.
func buildSounds() map[Sound][]byte {
	return map[Sound][]byte{
		// A quick little "boing" for every hop.
		SoundHop: render(segment{520, 760, 45 * time.Millisecond, 0.28}),
		// A cheerful rising pair when a frog makes it home.
		SoundHome: render(
			segment{660, 660, 80 * time.Millisecond, 0.35},
			segment{880, 880, 160 * time.Millisecond, 0.35},
		),
		// A buzzy chirp for catching the fly.
		SoundFly: render(
			segment{1200, 900, 40 * time.Millisecond, 0.30},
			segment{1200, 900, 40 * time.Millisecond, 0.30},
			segment{1200, 1500, 80 * time.Millisecond, 0.30},
		),
		// A flat thud when the frog is squashed.
		SoundSquash: render(segment{180, 60, 260 * time.Millisecond, 0.45}),
		// A gurgling fall into the water.
		SoundSplash: render(
			segment{600, 200, 120 * time.Millisecond, 0.35},
			segment{420, 120, 240 * time.Millisecond, 0.30},
		),
		// An urgent double tick as the clock runs low.
		SoundTimeWarning: render(
			segment{990, 990, 60 * time.Millisecond, 0.30},
			segment{0, 0, 70 * time.Millisecond, 0},
			segment{990, 990, 60 * time.Millisecond, 0.30},
		),
		// A sparkling run for an extra frog.
		SoundExtraLife: render(
			segment{784, 784, 70 * time.Millisecond, 0.35},
			segment{988, 988, 70 * time.Millisecond, 0.35},
			segment{1175, 1175, 70 * time.Millisecond, 0.35},
			segment{1568, 1568, 200 * time.Millisecond, 0.35},
		),
		// A bright rising triad as a level is cleared.
		SoundLevelUp: render(
			segment{523, 523, 90 * time.Millisecond, 0.35},
			segment{659, 659, 90 * time.Millisecond, 0.35},
			segment{784, 784, 140 * time.Millisecond, 0.35},
		),
		// A short descending arpeggio for game over.
		SoundGameOver: render(
			segment{330, 330, 160 * time.Millisecond, 0.40},
			segment{262, 262, 160 * time.Millisecond, 0.40},
			segment{196, 196, 160 * time.Millisecond, 0.40},
			segment{131, 131, 320 * time.Millisecond, 0.40},
		),
		// A triumphant ascending fanfare for winning the whole game.
		SoundWin: render(
			segment{523, 523, 130 * time.Millisecond, 0.40},
			segment{659, 659, 130 * time.Millisecond, 0.40},
			segment{784, 784, 130 * time.Millisecond, 0.40},
			segment{1047, 1047, 360 * time.Millisecond, 0.40},
		),
	}
}

// render synthesizes one or more segments into 16-bit little-endian PCM bytes.
func render(segs ...segment) []byte {
	var out []byte
	for _, s := range segs {
		n := int(float64(sampleRate) * s.dur.Seconds())
		attack := sampleRate / 200 // ~5ms fade-in kills the click
		release := sampleRate / 50 // ~20ms fade-out
		phase := 0.0
		for i := 0; i < n; i++ {
			t := float64(i) / float64(n)
			freq := s.fStart + (s.fEnd-s.fStart)*t
			phase += freq / sampleRate

			// Square wave.
			sample := 1.0
			if math.Mod(phase, 1.0) >= 0.5 {
				sample = -1.0
			}

			// Linear attack/release envelope.
			env := 1.0
			if i < attack {
				env = float64(i) / float64(attack)
			} else if i > n-release {
				env = float64(n-i) / float64(release)
			}

			v := int16(sample * env * s.vol * math.MaxInt16)
			out = append(out, byte(v), byte(v>>8))
		}
	}
	return out
}
