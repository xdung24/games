(() => {
	"use strict";

	const touch = navigator.maxTouchPoints > 0 || matchMedia("(any-pointer: coarse)").matches;
	const bar = document.createElement("nav");
	bar.id = "mobile-controls";
	bar.setAttribute("aria-label", "Star Breaker controls");
	bar.hidden = true;
	document.body.appendChild(bar);

	const pointers = new Map();
	const held = new Map();
	const keys = [
		["Left", "ArrowLeft", 37], ["Right", "ArrowRight", 39],
		["Thrust", "ArrowUp", 38], ["Fire / Play", " ", 32],
		["Jump", "h", 72], ["Pause", "p", 80],
		["Restart", "r", 82], ["Mute", "m", 77],
	];

	function send(type, key) {
		// The Fyne GLFW web driver reads keyCode for physical key events.
		document.dispatchEvent(new KeyboardEvent(type, {
			key: key[1], keyCode: key[2], which: key[2],
			bubbles: true, cancelable: true,
		}));
	}

	function release(id) {
		const button = pointers.get(id);
		if (!button) return;
		pointers.delete(id);
		const count = held.get(button) - 1;
		if (count > 0) {
			held.set(button, count);
			return;
		}
		held.delete(button);
		button.setAttribute("aria-pressed", "false");
		send("keyup", keys[Number(button.dataset.index)]);
	}

	function releaseAll() {
		for (const id of pointers.keys()) release(id);
	}

	for (const [index, key] of keys.entries()) {
		const button = document.createElement("button");
		button.type = "button";
		button.textContent = key[0];
		button.dataset.index = index;
		button.setAttribute("aria-pressed", "false");
		button.addEventListener("pointerdown", (event) => {
			if (event.button !== 0 || pointers.has(event.pointerId)) return;
			event.preventDefault();
			button.setPointerCapture(event.pointerId);
			pointers.set(event.pointerId, button);
			const count = held.get(button) || 0;
			held.set(button, count + 1);
			button.setAttribute("aria-pressed", "true");
			if (count === 0) send("keydown", key);
		});
		for (const type of ["pointerup", "pointercancel", "lostpointercapture"]) {
			button.addEventListener(type, (event) => {
				if (event.cancelable) event.preventDefault();
				release(event.pointerId);
			});
		}
		button.addEventListener("click", (event) => event.preventDefault());
		bar.appendChild(button);
	}

	// Do not let the game canvas receive compatibility mouse/touch events
	// from the controls, including when several fingers are held at once.
	for (const type of ["touchstart", "touchmove", "touchend", "touchcancel", "mousedown", "mouseup"]) {
		bar.addEventListener(type, (event) => {
			if (event.cancelable) event.preventDefault();
			event.stopPropagation();
		}, { passive: false });
	}
	document.addEventListener("gesturestart", (event) => event.preventDefault(), { passive: false });
	document.addEventListener("touchmove", (event) => {
		if (event.touches.length > 1) event.preventDefault();
	}, { passive: false });
	document.addEventListener("contextmenu", (event) => event.preventDefault());
	window.addEventListener("blur", releaseAll);
	document.addEventListener("visibilitychange", () => {
		if (document.hidden) releaseAll();
	});

	window.gameMobileControls = {
		height: () => touch ? 112 : 0,
		show(game) {
			releaseAll();
			bar.hidden = !touch || game !== "starbreaker";
		},
	};
})();
