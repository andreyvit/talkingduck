# Menu bar images

The desktop app and scratch command share this renderer. Agent motifs are
simplified pixel silhouettes: Claude rays, Codex paired lobes, Grok slashed arc.
Motifs are cropped inside the right border, preserving the solid or dotted
outline for each status. Filled attention cells use the same silhouette with inverted polarity.

Inspect both standard and compact output with `cmd/scratch-imagegen
-agent-backdrops` (add `-compact`). The 1× preview uses logical pixels; the 4×
preview enlarges that raster without smoothing. Neither simulates every AppKit
display setting; check the live menu bar too.
