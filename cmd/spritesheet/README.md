# spritesheet

Slices grid-based sprite sheets (WebP, PNG or JPEG) and packs the frames into
one transparent atlas, written as PNG and lossless WebP, plus a JSON metadata
file. Every source row becomes an atlas row, in input order, and the
background color is keyed to transparency.

```sh
go run ./cmd/spritesheet -o OUTPUT -cell WIDTHxHEIGHT [flags] input.webp...
```

`OUTPUT` is either:

- a directory (existing, or ending in `/`): files are named after it, so
  `-o out/hero/` writes `out/hero/hero.png`, `hero.webp` and `hero.json`;
- a base path such as `out/atlas` (a `.png`, `.webp` or `.json` extension is
  dropped): writes `out/atlas.png`, `out/atlas.webp` and `out/atlas.json`.

Missing directories are created.

| Flag | Default | Meaning |
|---|---|---|
| `-o` | (required) | Output directory or base path (see above) |
| `-cell` | (required) | Frame cell size, e.g. `192x225` |
| `-origin` | `0,0` | Top-left of the first cell in each input (skips labels/margins) |
| `-rows` | as many as fit | Rows read from each input |
| `-columns` | as many as fit | Cells read from each source row |
| `-counts` | `-columns` | Comma-separated frame count per output row; unused cells stay empty |
| `-bg` | `auto` | Background to remove: `auto` (top-left pixel), `none`, or `#RRGGBB` |
| `-tolerance` | `12` | Max per-channel difference still treated as background (helps with lossy WebP) |
| `-names` | `row0,row1,...` | Comma-separated animation name per output row; must be unique |
| `-meta` | `auto` | Metadata JSON path: `auto` (`<base>.json`), `none`, or a file path |
| `-formats` | `png,webp` | Image formats to write; WebP output is lossless and keeps transparency |

The atlas is as wide as the longest row (`max(counts) × cell width`) and
`total rows × cell height` tall.

## Metadata

Alongside the images, the tool writes a JSON file that names and locates every
sprite. Frames are named `<animation>_<index>` (0-based). `x`, `y`, `w`, `h`
give the frame's cell in the atlas, in pixels. `trim` is the visible
(non-transparent) area relative to the cell's top-left corner and is omitted
for empty frames. `images` lists every written image relative to the JSON
file, and `image` is the first of them.

```json
{
  "image": "warrior-atlas.png",
  "images": ["warrior-atlas.png", "warrior-atlas.webp"],
  "width": 1536,
  "height": 2250,
  "cellWidth": 192,
  "cellHeight": 225,
  "animations": [
    { "name": "walk", "row": 2, "frames": ["walk_0", "walk_1", "..."] }
  ],
  "frames": [
    {
      "name": "walk_3", "animation": "walk", "index": 3,
      "x": 576, "y": 450, "w": 192, "h": 225,
      "trim": { "x": 72, "y": 47, "w": 54, "h": 159 }
    }
  ]
}
```

## Example: Anime Warrior sheets

The two 1800×1200 sheets in `resources/Anime-Warrior-Man-Character-Pixel-Sprite/`
have five rows each, with labels in the left 234 px:

```sh
go run ./cmd/spritesheet -o resources/Anime-Warrior-Man-Character-Pixel-Sprite/ -cell 192x225 -origin 234,0 -rows 5 -counts 6,3,8,8,5,8,5,3,3,5 -names idle,idle2,walk,run,run_attack,jump,attack1,attack2,hurt,dead resources/Anime-Warrior-Man-Character-Pixel-Sprite/Anime-Warrior-Man-Character-Pixel-Sprite-Pack1.webp resources/Anime-Warrior-Man-Character-Pixel-Sprite/Anime-Warrior-Man-Character-Pixel-Sprite-Pack2.webp
```

This writes, into that directory, a 1536×2250 atlas (8 columns × 10 rows of
192×225 cells) with the same row order as
`kungfumaster/assets/warrior-atlas.png`:

- `Anime-Warrior-Man-Character-Pixel-Sprite.png`
- `Anime-Warrior-Man-Character-Pixel-Sprite.webp`
- `Anime-Warrior-Man-Character-Pixel-Sprite.json` (10 animations, 54 frames)
