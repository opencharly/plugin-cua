# plugin-cua

The OpenCharly plugin serving the **`cua`** computer-use verb and the **`kind: cua`**
entity — driving a live desktop through [Cua Driver](https://github.com/trycua/cua) from
a charly candy/box plan, out-of-process.

## What it is

`cua:` is a DECLARATIVE check/control verb — authored as `cua: <method>` inside a
candy/box plan `check:`/`run:` step. It is NOT a host `charly check` subcommand: the
verb's implementation lives in this out-of-tree module, and at check time the host
dispatches `cua:` through the provider registry to this plugin (the same path `jetkvm:` /
`wl:` / `record:` take).

Cua Driver is the background computer-use driver that exposes its tools over MCP-over-stdio
**and** a CLI (`cua-driver call <tool> '<json>'`, `cua-driver status`, `cua-driver
list-tools`, …). This plugin uses the CLI: one process per call, no long-lived client
state — exactly what an out-of-process check step wants. Every call is routed through the
venue's executor, so the driver is driven **where it lives** (a desktop VM or pod), never on
the host running charly.

## Authoring a `cua:` step

The method name is the scalar value for a bare-method step (`cua: status`), or the
`method:` key of the `cua:` map when the step carries cua-exclusive fields — those live
INSIDE the `cua:` map. Only the shared matchers (`stdout:`, `stderr:`, `exit_status:`) and
`context:`/`id:`/`timeout:` stay siblings.

```yaml
- check: the Cua driver is running in the desktop session
  context: [runtime]
  cua: status
  stdout:
    - contains: running

- check: a fresh desktop frame is captured as a real PNG
  context: [runtime]
  cua:
    method: get-desktop-state
    artifact: /tmp/desktop.png
    artifact_min_bytes: 10000
    artifact_not_uniform: true
```

## READ-ONLY BY DEFAULT — the desktop-safety gate

A live desktop is not a throwaway. Every **mutating** method (input, app/window control,
recording, the raw `call` escape hatch) requires `allow_control: true`; without it the step
reports a documented skip naming the gate rather than acting. The classification is an
allowlist, so a method added later is read-only by default.

Cua Driver's own permission mode (`standard` / `bounded` / `unrestricted`), declared on the
`kind: cua` entity, remains the authority the plugin passes through.

## Methods

Read-only: `status`, `doctor`, `version`, `list-tools`, `list-apps`, `list-windows`,
`get-window-state`, `get-desktop-state`, `get-screen-size`, `screenshot`.

Mutating (need `allow_control: true`): `click`, `double-click`, `right-click`, `drag`,
`type`, `press-key`, `hotkey`, `scroll`, `move-cursor`, `launch-app`, `kill-app`,
`bring-to-front`, `set-window-frame`, `set-value`, `recording-start`, `recording-stop`,
`recording-status`, `recording-render`, `call`.

`delivery: foreground` escalates an input action to Cua's foreground route; the default is
background, and a structured refusal (`background_unavailable`) is never a silent success.

## `kind: cua`

A `kind: cua` entity carries the connection/permission defaults a step inherits when it
names the entity with `device:` — the driver path, host, permission mode, and a default
session `env`. Authored step fields win over the entity's.

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

## Verifying

```bash
charly check live <image> --filter cua      # against a live deployment with the driver
charly check run cua-verb-probe             # the registry/verdict smoke bed
```

The full contract is exercised by the `check-cua-*` beds in `opencharly/distro-omarchy`.

## License

MIT — see `LICENSE`. The plugin itself carries no Cua code; it drives the `cua-driver`
binary installed by the `layer-cua` candies.
