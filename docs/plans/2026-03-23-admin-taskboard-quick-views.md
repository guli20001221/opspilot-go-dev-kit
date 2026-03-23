# 2026-03-23 Admin Taskboard Quick Views

## Goal

Add a minimal operator shortcut layer to the embedded task board so common monitoring slices can be reached without repeatedly re-entering the same filters.

## Scope

- keep the current backend contract unchanged
- reuse the existing filter form and URL sync model
- provide only a few high-frequency operator presets
- keep the presets transparent by mapping back into visible form fields

## Implementation notes

- add `Quick views` buttons near the board summary
- start with `All tasks`, `Needs approval`, `Failed`, and `Running`
- map each preset into the current filter form values, then call the existing board load path
- visually mark the active preset without replacing the underlying filters

## Validation

- add a failing admin page HTML test for the quick-view controls
- confirm the targeted page test passes after implementation
- rebuild the embedded API page in the local Compose stack
- create a tenant with both approval and report tasks, then verify `Needs approval` narrows the board and `All tasks` restores the full slice
