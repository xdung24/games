//go:build !js || !wasm

package touchcontrols

import "fyne.io/fyne/v2"

func Enabled() bool { return fyne.CurrentDevice().IsMobile() }

func attach(*Panel) func() { return func() {} }
