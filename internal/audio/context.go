package audio

import (
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

// Context opens one CD-quality mono device shared by all games. On the web,
// Oto becomes ready after the browser receives a user gesture.
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
	if err := waitReady(ready); err != nil {
		return nil, err
	}
	return ctx, nil
})
