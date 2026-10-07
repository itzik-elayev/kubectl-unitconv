# k9s integration

Adds two shortcuts on the pod view:

- `Shift-M` — the selected pod's memory requests/limits, in Gi.
- `Shift-C` — the selected pod's CPU requests/limits, in cores.

## Setup

Requires `kubectl-unitconv` and `less` on `PATH`, and k9s itself.

Merge `plugins.yaml` into your k9s plugin config (`~/.config/k9s/plugins.yaml`
on Linux, `~/Library/Application Support/k9s/plugins.yaml` on macOS). If you
already have plugins defined, copy the two entries under `unitconv-memory`
and `unitconv-cpu` into your existing `plugins:` map instead of overwriting
the file.

## How it works

Each shortcut runs `kubectl unitconv --from pod/<selected-pod> ...` using the
context, namespace, and kubeconfig k9s itself resolved (`$CONTEXT`,
`$NAMESPACE`, `$KUBECONFIG`) — including kubeconfig merging, since k9s
expands `$KUBECONFIG` to the single merged path it's actually using, not your
raw `KUBECONFIG` environment variable. Pod/context/namespace values are
passed as shell positional parameters rather than interpolated directly into
the command string, so they're quoted safely even if a name ever contained
unusual characters.

Output renders with `--output table --color always` (preserving the styled
table through the pipe) and is shown via `less -R` (which interprets color
codes instead of printing them raw). This temporarily opens an external
terminal view; press `q` to exit the pager and return to k9s. It does not
modify k9s's native columns or views — it's a standalone lookup, not a
persistent UI change.

If the underlying command fails (e.g. the pod has no quantity for the
requested resource type), the error is captured and shown in the pager too,
rather than failing silently.
