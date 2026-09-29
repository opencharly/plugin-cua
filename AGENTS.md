# AGENTS.md — plugin-cua

Standalone out-of-tree plugin repo serving the `cua` computer-use verb
(`verb:cua`) and the `cua` device entity (`kind:cua`). The plugin is a Go module
at `candy/plugin-cua/` (module path
`github.com/opencharly/plugin-cua/candy/plugin-cua`); the root `charly.yml` only
declares `discover: candy` so the repo is a project and its candy is scanned.

Canonical files:

- `candy/plugin-cua/charly.yml` — the `plugin-cua:` candy entity (`plugin:`
  block, `plan:` checks).
- `candy/plugin-cua/plugin.go` / `provider.go` — the verb provider, `NewMeta()`,
  and the `Invoke` verdict path.
- `candy/plugin-cua/catalog.go` — one typed wrapper per method; `client.go` —
  the `cua-driver` CLI client.
- `candy/plugin-cua/kind.go` — the `kind: cua` entity decode + device resolution.
- `candy/plugin-cua/schema/cua.cue` — the self-contained `#CuaInput` /
  `#CuaMethod` / `#CuaDeviceInput` (single source for `params/cue_types_gen.go`).
- `charly.yml` — the root project manifest (`discover: candy`) plus the
  disposable `cua-verb-probe` / `cua-entity-probe` local beds.
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge;
  `.github/workflows/ci.yml` — the module's Go gate.
- `README.md` — user overview only; never agent guidance.

## Load these skills first (R0)

- `/charly-check:cua` — the `cua:` verb + `kind: cua` reference. Load before
  changing the verb surface or the device entity. This candy carries no
  `skill:` entity of its own; the gap is tracked in
  [opencharly/opencharly#291](https://github.com/opencharly/opencharly/issues/291).
- `/charly-internals:plugin` — the plugin authoring reference: the `plugin:`
  block, the `verb` + `kind` provider classes, the per-plugin CUE-schema
  contract.
- `/charly-internals:git-workflow` — before any git/PR action.

## Build / validate / test

- `go build ./...` in `candy/plugin-cua/` — compile the plugin module.
- `go test ./...` in `candy/plugin-cua/` — the plugin's Go tests.
- `charly box validate` at the repo root — the structural check (the candy +
  `plugin:` block, CUE schema).
- The merge gate is the **org-wide** `charly/pr-validator` (required check
  `validate / validate`, defined in `opencharly/.github`); this repo has **no**
  per-repo candy gate. The `ci.yml` Go gate is this repo's own module test run.
- The R10 consumers are the `check-cua-*` beds in `opencharly/distro-omarchy`,
  which compose this plugin against a live Cua Fleet desktop.

## Modify this repo

- Edit the `plugin-cua:` candy entity, the Go source, and `schema/cua.cue`
  **together** — the schema is the single source for the verb's `params/` struct;
  regenerate `params/cue_types_gen.go` from it.
- The read-only / mutating split is an ALLOWLIST (`catalog.go`'s
  `mutatingMethods`): a new method is read-only by default until deliberately
  classified. Keep the `allow_control` gate on every mutating path.

## Landing

- PR-only. Every change lands through a pull request; the org-required
  `charly/pr-validator` validates the diff and body and arms native auto-merge on
  PASS. Direct pushes to `main` are blocked.
- History lives in `CHANGELOG/` (written by `tag-on-merge` at merge time); the PR
  body IS the changelog.
- The authoritative rulebook is the umbrella `AGENTS.md` in
  `opencharly/opencharly` and `charly/AGENTS.md` in the charly repo. Do not
  restate its rules here.
