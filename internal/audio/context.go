package audio

import (
	"fmt"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

// Context opens one CD-quality mono device shared by all games.
var Context = sync.OnceValues(func() (*oto.Context, error) {
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   44100,
		ChannelCount: 1,
		Format:       oto.FormatSignedInt16LE,
		BufferSize:   30 * time.Millisecond,
	})
	if err != nil {
		return nil, err
	}
	select {
	case <-ready:
		return ctx, nil
	case <-time.After(2 * time.Second):
		return nil, fmt.Errorf("audio device was not ready within 2 seconds")
	}
})
