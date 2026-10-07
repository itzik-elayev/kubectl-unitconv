# kubectl-unitconv

A kubectl plugin that converts Kubernetes `resource.Quantity` values between
unit representations — binary SI (`Ki`/`Mi`/`Gi`), decimal SI (`k`/`M`/`G`),
plain bytes/cores, and CPU nanocores/microcores/millicores — using the same
`k8s.io/apimachinery` parsing and exact decimal arithmetic Kubernetes itself
uses (never float64, so large quantities never lose precision before the
final, intentional rounding).

## Install

### Go

```sh
go build -o kubectl-unitconv .
sudo mv kubectl-unitconv /usr/local/bin/
```

### Nix

```sh
nix build github.com/itzik-elayev/kubectl-unitconv
./result/bin/kubectl-unitconv --help

# or, from a checkout:
nix build .
nix develop   # dev shell with go + golangci-lint
```

#### home-manager

The flake exports a home-manager module that installs the plugin and, when
`programs.k9s` is enabled, adds the [k9s shortcuts](#k9s-integration) to
`programs.k9s.plugins`:

```nix
{
  inputs.kubectl-unitconv.url = "github:itzik-elayev/kubectl-unitconv";

  # in your home-manager configuration:
  imports = [ inputs.kubectl-unitconv.homeManagerModules.default ];
  programs.kubectl-unitconv.enable = true;
  # programs.kubectl-unitconv.k9sPlugins.enable = false;  # opt out of k9s shortcuts
}
```

Without home-manager, `nix build .#k9s-plugins` produces the same
`plugins.yaml` to merge by hand.

Once installed on `PATH`, invoke it as `kubectl unitconv ...`.

### krew

Once a tagged release publishes real binaries (`plugin.yaml` currently has
placeholder `uri`/`sha256` values):

```sh
kubectl krew install --manifest=plugin.yaml
```

## Usage

Convert a literal value to a target unit:

```sh
kubectl unitconv 500Mi Gi
# 0.488281Gi

kubectl unitconv 250m cores
# 0.25
```

An explicit target unit can resolve an otherwise-ambiguous bare number
without needing `--family`:

```sh
kubectl unitconv 2 m
# 2000m   (target unit "m" is cpu-only, so family is inferred as cpu)
```

`m`/`u`/`n` (milli/micro/nano) suffixes are the exception: apimachinery
allows them on *any* quantity, not just cpu, and real clusters do write
storage values this way. So on a literal input they're only a last-resort
guess — an explicit memory-only target unit (or `--family memory`) always
wins instead of erroring:

```sh
kubectl unitconv 2387966302991320m Ti
# 2.171843Ti   (not an error: "Ti" decides memory, overriding the weak "m" guess)
```

Show every sensible unit for a value (no target unit given):

```sh
kubectl unitconv 500Mi
mebibytes (2^20)     500Mi
megabytes (10^6)     524.288M
kibibytes (2^10)     512000Ki
...
```

### Flags

- `--family {auto|cpu|memory}` — force which unit family to use when the
  input is ambiguous (default `auto`; a bare number like `2` defaults to
  memory). A resource field's own name (e.g. `memory`) always takes
  precedence over guessing from the quantity's suffix — a memory quantity
  written as `400m` is 400 millibytes, not 400 millicores.
- `--precision int` — decimal places in output, `0`–`18` (default `6`).
- `--output {auto|table|plain|json}` — see [Output modes](#output-modes).
- `--color {auto|always|never}` — see [Color](#color).

`bytes`/`cores`/`millicores`/`nanocores`/`microcores` are accepted as target
unit aliases; `bytes` and `cores` stay semantically distinct even though both
are a family's unitless base value — a `cores` target is rejected for a
memory quantity and vice versa, and mismatched target/family combinations
(e.g. treating PVC storage as cpu) are rejected with an error rather than
silently misinterpreted.

### Precision and rounding

Conversion always uses exact decimal arithmetic (the quantity's own `AsDec`
representation), never a float64 intermediate — a `float64` only has
~15–17 significant decimal digits, which silently corrupts very large or
very precise integer quantities before they're even rounded. Only the final
display step rounds, to `--precision` decimal digits using **round-half-to-
even** ("banker's rounding": `0.125` → `0.12`, `0.375` → `0.38`), the same
default IEEE 754 uses, chosen because it has no systematic upward or
downward bias across many conversions.

**A rounded display value is not guaranteed to round-trip back to the
original quantity.** `kubectl unitconv 1Mi Gi --precision 2` prints `0Gi`;
re-parsing `"0Gi"` does not recover `1Mi`. Use a higher `--precision`, or the
unrounded `--output json` `converted` string at full precision, if an exact
value matters downstream.

## Output modes

- `auto` (default): an attractive table when stdout is a terminal, the
  plain-text format below when redirected (piped, or into a file).
- `table`: always renders the table, even when redirected — useful for
  forcing styled output through a pager (see [k9s](#k9s-integration)).
- `plain`: the stable, uncolored text format, regardless of terminal.
- `json`: always a valid JSON array, never ANSI codes or decorative text —
  see [JSON output](#json-output).

```sh
kubectl unitconv --from pod/my-app Gi --output table
╭───────────┬──────────┬─────────────┬──────────┬───────────╮
│ CONTAINER │ RESOURCE │ REQUIREMENT │ ORIGINAL │ CONVERTED │
├───────────┼──────────┼─────────────┼──────────┼───────────┤
│ app       │ memory   │ requests    │ 1536Mi   │ 1.5Gi     │
│ app       │ memory   │ limits      │ 2Gi      │ 2Gi       │
╰───────────┴──────────┴─────────────┴──────────┴───────────╯

kubectl unitconv 1536Mi Gi --output table
1536Mi (1.5Gi)

kubectl unitconv 500Mi --output table
500Mi
├─ mebibytes (2^20):    500Mi
├─ megabytes (10^6):    524.288M
├─ kibibytes (2^10):    512000Ki
├─ kilobytes (10^3):    524288k
└─ bytes:               524288000
```

Show-all mode (no target unit) omits every unit the quantity is less than
one whole of (e.g. 500Mi in gibibytes: `0.488281Gi`) — that unit is too
coarse to express it naturally. The check is exact, independent of
`--precision`. If the quantity is below one of every unit (e.g. `0`), only
the finest unit is shown. This applies to `plain` and `json` output too,
not just `table`.

## Color

`--color auto` (the default) colors output only when writing to an actual
terminal, and honors [`NO_COLOR`](https://no-color.org) (any non-empty
value disables color) — both checked against the real output destination,
so piping through e.g. `less -R` or redirecting to a file disables color
automatically. `--color always`/`--color never` override detection
unconditionally, in either direction. `--output plain` and `--output json`
never emit color regardless of `--color`.

## JSON output

```sh
kubectl unitconv --from pod/my-app Gi --output json
```

```json
[
  {
    "kind": "Pod",
    "name": "my-app",
    "namespace": "default",
    "container": "app",
    "resourceType": "memory",
    "requirement": "requests",
    "original": "1536Mi",
    "targetUnit": "Gi",
    "converted": "1.5Gi"
  }
]
```

Fields (all stable; new fields may be added, existing ones won't change
shape):

| Field             | Type    | Notes                                                                 |
|-------------------|---------|------------------------------------------------------------------------|
| `kind`            | string  | Omitted for a literal conversion (no `--from`).                        |
| `name`            | string  | Omitted for a literal conversion.                                      |
| `namespace`       | string  | Omitted for cluster-scoped kinds (e.g. `Node`) and literal conversions.|
| `container`       | string  | Omitted when not applicable (literal conversion, PVC, node).           |
| `isInitContainer` | bool    | Omitted (`false`) when not applicable.                                 |
| `resourceType`    | string  | `"cpu"`, `"memory"`, or `"storage"`.                                   |
| `requirement`     | string  | `"requests"` or `"limits"`; omitted when not applicable.               |
| `original`        | string  | The quantity exactly as written/fetched, e.g. `"1536Mi"`.              |
| `targetUnit`      | string  | The resolved apimachinery suffix (e.g. `"Gi"`); empty for a family's base unit (bytes for memory/storage, cores for cpu). |
| `converted`       | string  | The rounded value, suffixed (e.g. `"1.5Gi"`).                          |

`original` and `converted` are always JSON strings, never bare numbers —
large quantities (e.g. byte counts near int64's range) lose precision in a
JavaScript `number` (a float64), so exact text is used instead.

A show-all conversion (no target unit given) produces one array element per
unit in the family the quantity is at least one whole of (see [Output
modes](#output-modes)), sharing every field except `targetUnit`/`converted`.
A `--from` lookup against multiple containers/requirements produces one
element per reading.

The result-building and rendering functions are internal Go packages
(`pkg/result`, `pkg/render`), structured so a future batch/stdin mode, or a
Lens extension consuming this same JSON to render native UI components
instead of a terminal table, can reuse them without re-deriving conversion
context from scratch. Neither exists yet — out of scope for this iteration.

## Live cluster lookup

`kubectl unitconv --from TYPE/NAME[:field.path] [TARGET_UNIT]` reads a
quantity straight off a live resource via your current kubeconfig context
(standard `--kubeconfig`/`--context`/`-n` flags all work), converting it the
same way as a literal value.

Known kinds (`pod`, `deployment`, `statefulset`, `daemonset`, `replicaset`,
`job`, `pvc`, `node`) have a built-in default field path, so `TYPE/NAME` alone
is often enough:

```sh
# every container's requests+limits, in every unit
kubectl unitconv --from pod/my-app

# same, converted straight to Gi
kubectl unitconv --from pod/my-app Gi

# narrow to one container and/or one requirement
kubectl unitconv --from pod/my-app --container app --requirement limits Gi

# force cpu family instead of memory (which "auto" defaults to)
kubectl unitconv --from pod/my-app --family cpu

# pvc and node have one obvious quantity each; PVC storage can't be cpu
kubectl unitconv --from pvc/data Gi
kubectl unitconv --from node/worker-1 --family cpu
```

For a kind with no default, or to read a specific field, give an explicit
path:

```sh
kubectl unitconv --from pod/my-app:spec.containers[0].resources.requests.memory Gi
```

### `--from`-specific flags

- `--container string` — with a default-path lookup, limit to one named
  container (default: all containers, including init containers). Requires
  `--from` with no explicit `:field.path`.
- `--requirement {requests|limits}` — with a default-path lookup, limit to
  one requirement (default: both). Same restriction as `--container`.

## k9s integration

[`examples/k9s/`](examples/k9s/) adds two k9s pod-view shortcuts: memory
requests/limits in Gi, and CPU requests/limits in cores, for the currently
selected pod — using `--output table --color always` piped through
`less -R` so the styling survives the pipe. See that directory's README for
setup and exactly how it preserves k9s's resolved context/namespace/
kubeconfig. Requires `less` in addition to `kubectl-unitconv` itself.

## Contributing

Pull requests must pass CI (`lint`, `test`, `release-dry-run`) and get an
approving review before merging into `main`. PRs are squash-merged, so the
PR title becomes the commit message — it must be a
[Conventional Commit](https://www.conventionalcommits.org) (`fix: ...`,
`feat: ...`, `feat!: ...`), checked by the `pr-title` workflow.

## Releasing

Releases are automated with
[release-please](https://github.com/googleapis/release-please). Every merge
to `main` updates an open "release PR" that bumps the version from the
Conventional Commit titles since the last release (`fix` → patch, `feat` →
minor, `!`/`BREAKING CHANGE` → major) and writes
`CHANGELOG.md`. Merging that release PR tags the commit and runs goreleaser,
publishing archives and checksums to the GitHub release.

The release PR is opened with the workflow's own token, which GitHub doesn't
let trigger other workflows — so CI won't run on it and an admin has to
merge it past the required checks. It only touches the version and
changelog.
