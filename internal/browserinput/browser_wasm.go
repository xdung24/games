//go:build js && wasm

package browserinput

import "syscall/js"

func ControlsHeight() float32 {
	controls := js.Global().Get("gameMobileControls")
	if controls.IsUndefined() {
		return 0
	}
	return float32(controls.Call("height").Float())
}

func Show(game string) {
	controls := js.Global().Get("gameMobileControls")
	if !controls.IsUndefined() {
		controls.Call("show", game)
	}
}
