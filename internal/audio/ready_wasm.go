//go:build js && wasm

package audio

func waitReady(<-chan struct{}) error {
	return nil
}
