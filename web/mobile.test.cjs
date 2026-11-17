"use strict";

const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const { join } = require("node:path");
const { test } = require("node:test");
const { runInNewContext } = require("node:vm");

const source = readFileSync(join(__dirname, "mobile.js"), "utf8");

class Element extends EventTarget {
	constructor(tag) {
		super();
		this.tag = tag;
		this.children = [];
		this.dataset = {};
		this.attributes = new Map();
	}
	appendChild(child) { this.children.push(child); }
	setAttribute(name, value) { this.attributes.set(name, String(value)); }
	getAttribute(name) { return this.attributes.get(name) ?? null; }
	setPointerCapture() {}
}

function setup() {
	const document = new Element("document");
	document.body = new Element("body");
	document.createElement = (tag) => new Element(tag);
	document.hidden = false;
	const window = new EventTarget();
	class KeyboardEvent extends Event {
		constructor(type, options) {
			super(type, options);
			this.key = options.key;
		}
	}
	runInNewContext(source, {
		document, window, KeyboardEvent,
		navigator: { maxTouchPoints: 5 },
		matchMedia: () => ({ matches: false }),
	});
	const elements = (parent) => [parent, ...parent.children.flatMap(elements)];
	const buttons = elements(document.body).filter((element) => element.tag === "button");
	const events = [];
	for (const type of ["keydown", "keyup"]) {
		document.addEventListener(type, (event) => events.push([event.type, event.key]));
	}
	const pointer = (button, type, id) => {
		const event = new Event(type, { cancelable: true });
		event.button = 0;
		event.pointerId = id;
		button.dispatchEvent(event);
	};
	window.gameMobileControls.show("starbreaker");
	return { document, window, buttons, events, pointer };
}

test("either Thrust button keeps the shared action held until both are released", () => {
	const { buttons, events, pointer } = setup();
	const [left, right] = buttons.filter((button) => button.getAttribute("aria-label") === "Thrust");
	pointer(left, "pointerdown", 1);
	pointer(right, "pointerdown", 2);
	assert.deepEqual(events, [["keydown", "ArrowUp"]]);
	pointer(left, "pointerup", 1);
	assert.equal(left.getAttribute("aria-pressed"), "false");
	assert.equal(right.getAttribute("aria-pressed"), "true");
	assert.deepEqual(events, [["keydown", "ArrowUp"]]);
	pointer(right, "pointerup", 2);
	assert.deepEqual(events, [["keydown", "ArrowUp"], ["keyup", "ArrowUp"]]);
});

test("multiple fingers, cancellation and lost capture release only their own holds", () => {
	const { window, buttons, events, pointer } = setup();
	const [thrust, otherThrust] = buttons.filter((button) => button.getAttribute("aria-label") === "Thrust");
	const fire = buttons.find((button) => button.getAttribute("aria-label") === "Fire / Play");
	pointer(thrust, "pointerdown", 1);
	pointer(thrust, "pointerdown", 2);
	pointer(otherThrust, "pointerdown", 3);
	pointer(fire, "pointerdown", 4);
	pointer(thrust, "pointercancel", 1);
	assert.equal(thrust.getAttribute("aria-pressed"), "true");
	pointer(thrust, "lostpointercapture", 2);
	assert.equal(thrust.getAttribute("aria-pressed"), "false");
	assert.deepEqual(events, [["keydown", "ArrowUp"], ["keydown", " "]]);
	pointer(otherThrust, "pointercancel", 3);
	assert.equal(fire.getAttribute("aria-pressed"), "true");
	window.dispatchEvent(new Event("blur"));
	assert.deepEqual(events, [
		["keydown", "ArrowUp"], ["keydown", " "],
		["keyup", "ArrowUp"], ["keyup", " "],
	]);
	pointer(fire, "lostpointercapture", 4);
	assert.equal(events.length, 4);
});

test("navigation and visibility loss clear shared action counts for the next session", () => {
	for (const reason of ["navigation", "visibility"]) {
		const { document, window, buttons, events, pointer } = setup();
		const thrusts = buttons.filter((button) => button.getAttribute("aria-label") === "Thrust");
		thrusts.forEach((button, index) => pointer(button, "pointerdown", index + 1));
		if (reason === "navigation") {
			window.gameMobileControls.show("");
		} else {
			document.hidden = true;
			document.dispatchEvent(new Event("visibilitychange"));
		}
		assert.deepEqual(events, [["keydown", "ArrowUp"], ["keyup", "ArrowUp"]], reason);
		assert.ok(thrusts.every((button) => button.getAttribute("aria-pressed") === "false"), reason);
		document.hidden = false;
		window.gameMobileControls.show("starbreaker");
		pointer(thrusts[0], "pointerdown", 3);
		assert.deepEqual(events.at(-1), ["keydown", "ArrowUp"], reason);
	}
});
