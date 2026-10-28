# Control Center API

The plugin listens on `127.0.0.1:8060` and accepts JSON-RPC 2.0 requests at
`POST /jsonrpc`. Requests may be single calls or batches. A call without an
`id` is a notification and does not receive a JSON-RPC response.

```json
{"jsonrpc":"2.0","method":"screen.tap","params":{"x":300,"y":400},"id":1}
```

`GET /screenshot` returns the current screen as a PNG. Its query string is
passed directly to the screenshot provider.

## Methods

| Method | Parameters | Result | Description |
| --- | --- | --- | --- |
| `ping` | none | `"pong"` | Verifies that the server is reachable. |
| `ui.find_widget` | `id` (string) | UI-tree node or `null` | Builds the visible UI tree and recursively searches it from top to bottom for a matching `test_id`. |
| `plugin.status` | `name` (string) | plugin details or `null` | Returns the entry for `name` from KOReader's loaded plugin details. |
| `menu.open` | none | `null` | Opens the main menu. |
| `menu.activate` | `path` (string) | `{ activated, path }` | Activates the entry at an exact numerical path. It selects the tab, enters intermediate submenus, and performs the final entry's normal tap action. Fails if the path is invalid, missing, or disabled. |
| `screen.tap` | `x`, `y` (number) | `null` | Sends a tap gesture at the screen coordinates. |
| `screen.hold` | `x`, `y` (number) | `null` | Sends a hold gesture at the screen coordinates. |
| `screen.swipe` | `x`, `y`, `direction`, `distance` | `true` | Sends a swipe starting at the screen coordinates. `direction` is `north`, `east`, `south`, or `west`; `distance` is in pixels. |
| `screen.size` | none | `{ width, height }` | Returns screen dimensions in pixels. |
| `ui.window_stack` | none | array | Returns visible window-stack entries with widget metadata, position, and dimensions when available. |
| `ui.touch_zones` | none | array | Returns touch zones for the topmost visible widget. Each entry contains `id`, gesture type (`ges`), and its `range` when available. |
| `system.event` | `name` (string) | `true` | Schedules a KOReader event with the supplied name. |
| `system.version` | none | string | Returns the current KOReader revision. |
| `system.suspend_for` | `seconds` (number) | `true` | Requests suspend, then schedules KOReader's resume handler after `seconds`. |
| `system.restart` | none | `true` | Schedules a `Restart` event. |
| `system.exit` | none | `true` | Schedules an `Exit` event. |

### Widget details

`ui.find_widget` returns the matching serialized UI-tree node, including its
widget metadata, `depth`, and recursive `children`. A root node also includes
its window-stack `pos`. It returns `null` when no visible widget matches.

## Menu paths

A menu path is a dot-separated sequence of one-based numerical indexes such as
`"4.2.3"`. The first number selects a tab in KOReader's main menu. Each
remaining number selects an item in that tab or in the submenu selected by the
previous number. In this example, `4` is the fourth menu tab, `2` is its
second item, and `3` is the third item inside that item's submenu.

The final component identifies the entry to activate. Kontroller selects the
path's tab, activates each intermediate submenu entry, and then calls the final
entry's normal tap action. Depending on that entry, this may toggle a setting,
run a command, open a dialog, or enter another submenu.

```json
{"jsonrpc":"2.0","method":"menu.activate","params":{"path":"4.2.3"},"id":1}
```

The result is an object containing `activated: true` and the requested `path`.
For example, the request above returns
`{"activated":true,"path":"4.2.3"}`. Numerical paths reflect the current menu
layout, so they may change across KOReader versions or when settings and
installed plugins alter the menu.

## Errors

The server follows JSON-RPC error codes: `-32700` for parse errors, `-32600`
for invalid requests, `-32601` for unknown methods, and `-32603` when a method
raises an error. Malformed HTTP requests return `400`; POST requests without a
valid `Content-Length` return `411`, and bodies over 1 MiB return `413`.
