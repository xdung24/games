package asteroidshooter

import "fyne.io/fyne/v2/canvas"

// The neon background is a single full-field canvas.Shader. It paints a drifting
// magenta/cyan/violet nebula from layered value-noise, a hashed twinkling
// starfield, and a soft vignette — the animated backdrop the vector ships and
// rocks glow against. The game loop advances "time" only while the scene moves.
//
// The fragment body is shared verbatim between the desktop (OpenGL core,
// #version 110) and the mobile/web (OpenGL ES, #version 100) targets; only the
// version/precision header differs. It follows Fyne's vector-shader contract:
// frame, bounds (x1,y1,x2,y2) and time are supplied by the painter.
const neonFragBody = `
uniform vec2 frame;
uniform vec4 bounds;
uniform float time;

float hash21(vec2 p) {
    p = fract(p * vec2(123.34, 456.21));
    p += dot(p, p + 45.32);
    return fract(p.x * p.y);
}

vec2 hash22(vec2 p) {
    float n = hash21(p);
    return vec2(n, hash21(p + n));
}

float vnoise(vec2 p) {
    vec2 i = floor(p);
    vec2 f = fract(p);
    f = f * f * (3.0 - 2.0 * f);
    float a = hash21(i);
    float b = hash21(i + vec2(1.0, 0.0));
    float c = hash21(i + vec2(0.0, 1.0));
    float d = hash21(i + vec2(1.0, 1.0));
    return mix(mix(a, b, f.x), mix(c, d, f.x), f.y);
}

float fbm(vec2 p) {
    float v = 0.0;
    float amp = 0.5;
    for (int i = 0; i < 5; i++) {
        v += amp * vnoise(p);
        p *= 2.0;
        amp *= 0.5;
    }
    return v;
}

void main() {
    // Object-local pixel coords with a top-left origin (+y down), matching the
    // built-in vector shaders, then normalised to 0..1 across the object.
    vec2 fragXY = vec2(gl_FragCoord.x, frame.y - gl_FragCoord.y) - bounds.xy;
    vec2 size = vec2(bounds.z - bounds.x, bounds.w - bounds.y);
    vec2 uv = fragXY / max(size, vec2(1.0));
    float aspect = size.x / max(size.y, 1.0);
    vec2 p = vec2(uv.x * aspect, uv.y);

    float t = time * 0.04;
    vec2 q = p * 3.0;
    float n1 = fbm(q + vec2(t, t * 0.5));
    float n2 = fbm(q * 1.7 - vec2(t * 0.7, t));
    float n3 = fbm(q * 0.8 + vec2(-t, t * 0.3));

    vec3 magenta = vec3(0.85, 0.12, 0.62);
    vec3 cyan = vec3(0.10, 0.75, 0.90);
    vec3 violet = vec3(0.35, 0.10, 0.65);

    vec3 col = vec3(0.015, 0.01, 0.04);
    col += violet * pow(n1, 2.2) * 0.55;
    col += magenta * pow(n2, 3.0) * 0.45;
    col += cyan * pow(n3, 3.0) * 0.35;

    // Vignette so the field edges fall into depth.
    float vig = smoothstep(1.25, 0.30, length(uv - 0.5));
    col *= vig;

    // Starfield: one candidate star per grid cell, a few bright ones twinkle.
    vec2 grid = p * 60.0;
    vec2 cell = floor(grid);
    vec2 gv = fract(grid) - 0.5;
    vec2 star = hash22(cell) - 0.5;
    float d = length(gv - star * 0.7);
    float bright = hash21(cell + 3.1);
    if (bright > 0.82) {
        float tw = 0.6 + 0.4 * sin(time * 2.0 + bright * 30.0);
        col += vec3(smoothstep(0.08, 0.0, d) * tw);
    }

    gl_FragColor = vec4(col, 1.0);
}
`

// neonFragCore is the desktop OpenGL (core profile) fragment shader.
const neonFragCore = "#version 110\n" + neonFragBody

// neonFragES is the OpenGL ES / mobile / web fragment shader.
const neonFragES = `#version 100
#ifdef GL_ES
# ifdef GL_FRAGMENT_PRECISION_HIGH
precision highp float;
# else
precision mediump float;
# endif
precision mediump int;
#endif
` + neonFragBody

// newNeonBackground builds the nebula shader, driven by the game loop's clock.
func newNeonBackground() *canvas.Shader {
	s := canvas.NewShader("asteroidshooter-neon", []byte(neonFragCore), []byte(neonFragES))
	s.Uniforms = map[string]float32{"time": 0}
	return s
}
