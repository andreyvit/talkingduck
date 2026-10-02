# Menu bar image experiment

Standalone, standard-library Go renderer; no Wails, cgo, font installation, or
changes to the inventory collector. Run from the repository root:

```sh
CGO_ENABLED=0 go run ./cmd/scratch-imagegen
CGO_ENABLED=0 go run ./cmd/scratch-imagegen -input /path/to/cells.json -out .tasks/menubar-images
```

Without input it renders a synthetic style sample. Input is a JSON array:

```json
[{"number":1,"stage":"L","state":"running"},
 {"number":2,"stage":"Q","state":"attention"}]
```

Stages: `-`, `d`, `H`, `L`, `E`, `F`, `Q`, `R`, `P`, `?`.
States: `idle` (dim dotted border), `running` (solid outline), `attention`
(filled with transparent reversed text), `unknown` (corner marks).

The renderer lays out cells in two rows, preserving input order. Ten cells use
123 x 23 logical pixels. The 5x7 pixel font is deliberately aligned to logical
pixels. Outputs include transparent 2x PNG icons and composited light/dark
previews at 1x and 4x. The enlarged preview uses nearest-neighbor scaling.
The integration must assign the icon its logical size; @2x naming alone is not
sufficient when passing image bytes to a native API.

This is a drawing experiment, not an inventory parser or a folder-state policy.
Personal sample inputs and generated images belong in ignored `.tasks/`.
In the initial supplied-data preview, the first listed task represents each
folder; completed stage marks and explicit questions map to attention. Unknown
progress remains unknown. These are provisional visualization choices, not a
claim that a completed stage means the whole task is complete.

Check changes with `CGO_ENABLED=0 go vet ./cmd/scratch-imagegen`, run the sample,
and inspect both preview themes. No app integration has been tested.
