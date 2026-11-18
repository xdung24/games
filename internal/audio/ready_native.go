//go:build !js || !wasm

package audio

import (
	"fmt"
	"time"
)

func waitReady(ready <-chan struct{}) error {
	select {
	case <-ready:
		return nil
	case <-time.After(2 * time.Second):
		return fmt.Errorf("audio device was not ready within 2 seconds")
	}
}
