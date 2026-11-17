//go:build js && wasm

package touchcontrols

import (
	"sync"
	"syscall/js"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/mobile"
)

func Enabled() bool {
	global := js.Global()
	navigator := global.Get("navigator")
	points := navigator.Get("maxTouchPoints")
	return (points.Type() == js.TypeNumber && points.Int() > 0) ||
		global.Call("matchMedia", "(any-pointer: coarse)").Get("matches").Bool()
}

// Fyne's web driver currently exposes a single mouse stream. Translate browser
// pointers into Go widget touches without DOM controls or keyboard emulation.
func attach(panel *Panel) func() {
	global := js.Global()
	document := global.Get("document")
	element := document.Call("querySelector", "canvas")
	if element.IsNull() {
		fyne.LogError("Cannot attach game touch controls: browser canvas is missing", nil)
		return func() {}
	}
	var mu sync.Mutex
	active := make(map[int]*Button)
	attached := true
	queue := func(fn func()) {
		fyne.Do(func() {
			mu.Lock()
			valid := attached
			mu.Unlock()
			if valid {
				fn()
			}
		})
	}
	hit := func(event js.Value) *Button {
		rect := element.Call("getBoundingClientRect")
		return panel.hit(
			float32(event.Get("clientX").Float()-rect.Get("left").Float()),
			float32(event.Get("clientY").Float()-rect.Get("top").Float()),
			float32(rect.Get("width").Float()), float32(rect.Get("height").Float()),
		)
	}
	stop := func(event js.Value) {
		event.Call("preventDefault")
		event.Call("stopImmediatePropagation")
	}
	down := js.FuncOf(func(_ js.Value, args []js.Value) any {
		event := args[0]
		if event.Get("pointerType").String() == "mouse" {
			return nil
		}
		button := hit(event)
		if button == nil {
			return nil
		}
		id := event.Get("pointerId").Int()
		mu.Lock()
		active[id] = button
		mu.Unlock()
		stop(event)
		element.Call("setPointerCapture", id)
		queue(func() { button.TouchDown(&mobile.TouchEvent{ID: id}) })
		return nil
	})
	up := js.FuncOf(func(_ js.Value, args []js.Value) any {
		event := args[0]
		id := event.Get("pointerId").Int()
		mu.Lock()
		button := active[id]
		delete(active, id)
		mu.Unlock()
		if button != nil {
			stop(event)
			queue(func() { button.TouchUp(&mobile.TouchEvent{ID: id}) })
		}
		return nil
	})
	click := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if hit(args[0]) != nil {
			stop(args[0])
		}
		return nil
	})
	reset := js.FuncOf(func(_ js.Value, _ []js.Value) any {
		mu.Lock()
		clear(active)
		mu.Unlock()
		queue(panel.Reset)
		return nil
	})
	visibility := js.FuncOf(func(_ js.Value, _ []js.Value) any {
		if document.Get("hidden").Bool() {
			mu.Lock()
			clear(active)
			mu.Unlock()
			queue(panel.Reset)
		}
		return nil
	})
	listeners := []struct {
		target js.Value
		name   string
		fn     js.Func
	}{
		{document, "pointerdown", down},
		{document, "pointerup", up},
		{document, "pointercancel", up},
		{document, "lostpointercapture", up},
		{document, "click", click},
		{global, "blur", reset},
		{document, "visibilitychange", visibility},
	}
	for _, l := range listeners {
		l.target.Call("addEventListener", l.name, l.fn, true)
	}
	return func() {
		mu.Lock()
		attached = false
		clear(active)
		mu.Unlock()
		for _, l := range listeners {
			l.target.Call("removeEventListener", l.name, l.fn, true)
		}
		for _, fn := range []js.Func{down, up, click, reset, visibility} {
			fn.Release()
		}
	}
}
