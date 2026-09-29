# plugin-cua

The `cua:` computer-use verb and the `kind: cua` entity for OpenCharly — drive a
live desktop through [Cua Driver](https://github.com/trycua/cua) from a candy/box
plan, served out-of-process (`verb:cua`, `kind:cua`).

The plugin is a standalone Go module the charly loader host-builds and serves
over go-plugin gRPC, so the Cua wire lives here, out of charly's core check
surface. Every call is routed through the venue's executor, so the driver is
driven **where it lives** (a desktop VM or pod), never on the host running
charly.

## What it provides

| Capability | Surface |
|---|---|
| `verb:cua` | the declarative `cua:` check/control step |
| `kind:cua` | the `cua:` device entity — connection + permission defaults a step inherits via `device:` |

## The verb

Cua Driver exposes its tools over MCP-over-stdio **and** a CLI
(`cua-driver call <tool> '<json>'`, `cua-driver status`, …). This plugin uses the
CLI: one process per call, no long-lived client state.

Read-only: `status`, `doctor`, `version`, `list-tools`, `list-apps`,
`list-windows`, `get-window-state`, `get-desktop-state`, `get-screen-size`,
`screenshot`.

Mutating (need `allow_control: true`): `click`, `double-click`, `right-click`,
`drag`, `type`, `press-key`, `hotkey`, `scroll`, `move-cursor`, `launch-app`,
`kill-app`, `bring-to-front`, `set-window-frame`, `set-value`, `recording-start`,
`recording-stop`, `recording-status`, `recording-render`, `call`.

`delivery: foreground` escalates an input action to Cua's foreground route; the
default is background, and a structured refusal (`background_unavailable`) is
never a silent success.

## READ-ONLY BY DEFAULT — the desktop-safety gate

A live desktop is not a throwaway. Every **mutating** method requires
`allow_control: true`; without it the step reports a documented skip naming the
gate rather than acting. The classification is an allowlist, so a method added
later is read-only by default. Cua Driver's own permission mode (`standard` /
`bounded` / `unrestricted`), declared on the `kind: cua` entity, remains the
authority the plugin passes through.

## How to use it

Compose the plugin candy in a box or check bed's `candy:` list:

```yaml
- '@github.com/opencharly/plugin-cua/candy/plugin-cua:<tag>'
```

Then author the verb in a plan:

```yaml
- check: the Cua driver is running in the desktop session
  context: [runtime]
  cua: status
  stdout:
    - contains: running
```

## The `kind: cua` entity

A `kind: cua` entity carries the connection/permission defaults a step inherits
when it names the entity with `device:` — the driver path, host, permission mode,
and a default session `env`. Authored step fields win over the entity's.

```yaml
my-desktop:
    cua:
        description: the Cua Omarchy desktop
        driver: /usr/local/bin/cua-driver
        permission_mode: standard
        env:
            XDG_RUNTIME_DIR: /run/user/1000
            WAYLAND_DISPLAY: wayland-1
```

## Layout

- `candy/plugin-cua/` — the plugin module: `plugin.go`, `provider.go`,
  `catalog.go` (one wrapper per method), `client.go` (the `cua-driver` CLI
  client), `kind.go` (the `kind: cua` entity decode + device resolution),
  `schema/cua.cue` (the self-contained `#CuaInput` / `#CuaMethod` /
  `#CuaDeviceInput`), `params/cue_types_gen.go`, `plugin_test.go`,
  `cmd/serve/main.go`.
- `charly.yml` — the root project manifest (`discover: candy`) plus the
  `cua-verb-probe` / `cua-entity-probe` disposable local beds.
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.
- `.github/workflows/ci.yml` — the Go gate (gofmt, golangci-lint, `go vet`,
  `go test`) for the module's unit tests.
- `LICENSE` — MIT.

## Related

- Owning skill: `/charly-check:cua` — the `cua:` verb + `kind: cua` reference.
  This candy carries no `skill:` entity of its own; the gap is tracked in
  [opencharly/opencharly#291](https://github.com/opencharly/opencharly/issues/291).
- `/charly-check:check` — the declarative check-step surface the verb is
  authored through.
- [`opencharly/charly`](https://github.com/opencharly/charly) — the charly CLI.
