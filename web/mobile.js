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
	const buttonHolds = new Map();
	const keys = [
		["Left", "ArrowLeft", 37, "left"], ["Right", "ArrowRight", 39, "right"],
		["Thrust", "ArrowUp", 38, "thrust"], ["Fire / Play", " ", 32, "fire"],
		["Jump", "h", 72, "jump"], ["Pause", "p", 80, "pause"],
		["Restart", "r", 82, "restart"], ["Mute", "m", 77, "mute"],
	];
	const icons = {
		left: '<path d="M19 12H5m7-7-7 7 7 7"/>',
		right: '<path d="M5 12h14m-7-7 7 7-7 7"/>',
		thrust: '<path d="M12 3c1 4-2 5-2 8 0 1 1 2 2 2 2 0 3-2 3-4 3 3 4 5 3 8-1 3-4 5-7 4-4-1-6-5-5-8 1-3 3-5 6-10Z"/>',
		fire: '<circle cx="12" cy="12" r="6"/><path d="M12 2v4m0 12v4M2 12h4m12 0h4"/><circle cx="12" cy="12" r="1"/>',
		jump: '<path d="m13 2-9 12h7l-1 8L20 9h-7Z"/>',
		pause: '<path d="M9 5v14M15 5v14"/>',
		restart: '<path d="M4 10a8 8 0 1 1 1.8 7M4 4v6h6"/>',
		mute: '<path d="m11 5-5 4H3v6h3l5 4V5ZM16 9l5 6m0-6-5 6"/>',
	};

	function group(parent, label, className) {
		const element = document.createElement("div");
		element.className = className;
		element.setAttribute("role", "group");
		element.setAttribute("aria-label", label);
		parent.appendChild(element);
		return element;
	}

	const system = group(bar, "System controls", "system-controls");
	const gameplay = document.createElement("div");
	gameplay.className = "gameplay-controls";
	bar.appendChild(gameplay);
	const left = group(gameplay, "Left-hand gameplay controls", "hand-controls");
	const center = group(gameplay, "Center gameplay controls", "center-controls");
	const right = group(gameplay, "Right-hand gameplay controls", "hand-controls");

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
		const buttonCount = buttonHolds.get(button) - 1;
		if (buttonCount > 0) {
			buttonHolds.set(button, buttonCount);
		} else {
			buttonHolds.delete(button);
			button.setAttribute("aria-pressed", "false");
		}
		const key = keys[Number(button.dataset.index)];
		const count = held.get(key) - 1;
		if (count > 0) {
			held.set(key, count);
			return;
		}
		held.delete(key);
		send("keyup", key);
	}

	function releaseAll() {
		for (const id of pointers.keys()) release(id);
	}

	function addButton(parent, index) {
		const key = keys[index];
		const button = document.createElement("button");
		button.type = "button";
		button.setAttribute("aria-label", key[0]);
		button.title = key[0];
		button.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">${icons[key[3]]}</svg>`;
		button.dataset.index = index;
		button.setAttribute("aria-pressed", "false");
		button.addEventListener("pointerdown", (event) => {
			if (event.button !== 0 || pointers.has(event.pointerId)) return;
			event.preventDefault();
			button.setPointerCapture(event.pointerId);
			pointers.set(event.pointerId, button);
			buttonHolds.set(button, (buttonHolds.get(button) || 0) + 1);
			const count = held.get(key) || 0;
			held.set(key, count + 1);
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
		parent.appendChild(button);
	}

	for (const [parent, indices] of [
		[left, [0, 1]],
		[center, [4]],
		[right, [2, 3]],
		[system, [5, 6, 7]],
	]) {
		for (const index of indices) addButton(parent, index);
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
