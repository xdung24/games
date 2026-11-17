//go:build !js || !wasm

package browserinput

func ControlsHeight() float32 { return 0 }

func Show(string) {}
