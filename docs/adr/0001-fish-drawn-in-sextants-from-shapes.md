# The Fish is drawn in sextant pixels, from shapes

Visuals are what this project is for, so the Fish is pixel art at the finest resolution a terminal cell reliably gives: sextant characters (U+1FB00 block, 2×3 pixels per cell). Each Species is stored as shapes (ellipses and polygons) and drawn into sextants every frame. Drawing shapes live is what gives the tail its smooth swish and lets the eye show each Mood and blinks without a separate frame for each. A cell holds only a foreground and a background colour, so where a third colour falls in a cell it is replaced by the nearer of the two most common. Fonts without sextants show boxes in their place; we accepted that until ADR 0004, which adds a Coarse Detail for them.

## Considered Options

- **Half-blocks (`▀▄`, 1×2):** work in every font, but too coarse for eyes and markings.
- **Quadrants (2×2):** safe in fonts, half the vertical detail of sextants.
- **Braille (2×4):** finest, but drawn as dots with gaps, so the Fish looks stippled.
- **Glyph eyes** (a `◕` character on the body colour) instead of pixel eyes: Mood is easier to read, but the character doesn't match the pixels around it.
- **Hand-made pixel grids:** can be touched up pixel by pixel, but need a separate frame for every Mood and tail position.
