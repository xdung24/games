(() => {
	"use strict";

	document.addEventListener("gesturestart", (event) => event.preventDefault(), { passive: false });
	document.addEventListener("touchmove", (event) => {
		if (event.touches.length > 1) event.preventDefault();
	}, { passive: false });
	document.addEventListener("contextmenu", (event) => event.preventDefault());
})();
