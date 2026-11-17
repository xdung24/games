package asteroidshooter

import (
	"bytes"
	"math"
	"math/rand"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"github.com/ebitengine/oto/v3"
	"github.com/xdung24/games/internal/audio"
)

// Audio format for the synthesized blips: CD-quality mono, 16-bit signed.
const (
	sampleRate = 44100
)

// soundPlayer renders short retro effects for game events. It shares the app's
// oto context and owns a pool of in-flight players. If the audio device can't be opened
// the player disables itself and every Play call is a no-op, so the game runs
// fine on machines with no sound.
type soundPlayer struct {
	ctx     *oto.Context
	enabled bool
	muted   bool // when true, Play is a no-op (toggled by the player, persisted)
	pcm     map[Sound][]byte

	mu     sync.Mutex
	active []*oto.Player
}

// SetMuted silences (or unsilences) all effects.
func (sp *soundPlayer) SetMuted(m bool) {
	if sp != nil {
		sp.muted = m
	}
}

// ToggleMuted flips the mute state and reports the new value.
func (sp *soundPlayer) ToggleMuted() bool {
	if sp == nil {
		return false
	}
	sp.muted = !sp.muted
	return sp.muted
}

// IsMuted reports whether effects are currently silenced.
func (sp *soundPlayer) IsMuted() bool { return sp != nil && sp.muted }

// newSoundPlayer opens the audio device and pre-renders every effect. It returns
// a usable (possibly disabled) player and never an error — silence is an
// acceptable degraded mode.
func newSoundPlayer() *soundPlayer {
	sp := &soundPlayer{pcm: buildSounds()}

	ctx, err := audio.Context()
	if err != nil {
		fyne.LogError("Asteroid Shooter audio is unavailable", err)
		return sp // disabled: no audio device
	}
	sp.ctx = ctx
	sp.enabled = true
	return sp
}

// Play renders the requested effect. Safe to call from any goroutine.
func (sp *soundPlayer) Play(s Sound) {
	if sp == nil || !sp.enabled || sp.muted {
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

// --- synthesis --------------------------------------------------------------

// wave selects the timbre of a tone segment.
type wave int

const (
	waveSquare wave = iota // bright retro tone
	waveNoise              // hissy burst, used for thrust and explosions
)

// segment is one synthesized sound stage: a freq-swept tone or a noise burst.
type segment struct {
	kind         wave
	fStart, fEnd float64 // hertz, linearly swept (ignored for noise)
	dur          time.Duration
	vol          float64 // 0..1 peak amplitude
}

func tone(f0, f1 float64, d time.Duration, vol float64) segment {
	return segment{kind: waveSquare, fStart: f0, fEnd: f1, dur: d, vol: vol}
}

func noise(d time.Duration, vol float64) segment {
	return segment{kind: waveNoise, dur: d, vol: vol}
}

// buildSounds pre-renders the PCM for every game event.
func buildSounds() map[Sound][]byte {
	return map[Sound][]byte{
		// A short bright zap as a bullet leaves the ship.
		SoundFire: render(tone(900, 540, 90*time.Millisecond, 0.28)),
		// A breathy low rumble for the engine.
		SoundThrust: render(noise(120*time.Millisecond, 0.18)),
		// Explosions: noise bursts, deeper and longer for bigger rocks.
		SoundBangLarge:  render(noise(380*time.Millisecond, 0.42)),
		SoundBangMedium: render(noise(260*time.Millisecond, 0.36)),
		SoundBangSmall:  render(noise(170*time.Millisecond, 0.30)),
		// Ship death: a noisy boom tailing into a falling tone.
		SoundShipDeath: render(
			noise(220*time.Millisecond, 0.45),
			tone(300, 90, 420*time.Millisecond, 0.40),
		),
		// A bright rising triad for a bonus ship.
		SoundExtraLife: render(
			tone(523, 523, 90*time.Millisecond, 0.32),
			tone(659, 659, 90*time.Millisecond, 0.32),
			tone(784, 784, 150*time.Millisecond, 0.32),
		),
		// The saucer's two-tone warble (one cycle; it loops via repeated spawns).
		SoundSaucer: render(
			tone(360, 360, 110*time.Millisecond, 0.26),
			tone(280, 280, 110*time.Millisecond, 0.26),
		),
		// A thin descending zap for the saucer's gun.
		SoundSaucerFire: render(tone(620, 360, 110*time.Millisecond, 0.24)),
		// A rising sweep announcing the next wave.
		SoundNewWave: render(tone(330, 660, 260*time.Millisecond, 0.30)),
	}
}

// render synthesizes one or more segments into 16-bit little-endian PCM bytes.
func render(segs ...segment) []byte {
	var out []byte
	for _, s := range segs {
		n := int(float64(sampleRate) * s.dur.Seconds())
		attack := sampleRate / 200 // ~5ms fade-in kills the click
		release := sampleRate / 40 // ~25ms fade-out
		phase := 0.0
		for i := 0; i < n; i++ {
			t := float64(i) / float64(n)

			var sample float64
			switch s.kind {
			case waveNoise:
				sample = rand.Float64()*2 - 1
			default: // waveSquare
				freq := s.fStart + (s.fEnd-s.fStart)*t
				phase += freq / sampleRate
				sample = 1.0
				if math.Mod(phase, 1.0) >= 0.5 {
					sample = -1.0
				}
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
