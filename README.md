# kubectl-unitconv

A kubectl plugin that converts Kubernetes `resource.Quantity` values between
unit representations — binary SI (`Ki`/`Mi`/`Gi`), decimal SI (`k`/`M`/`G`),
plain bytes/cores, and CPU millicores/nanocores — using the same
`k8s.io/apimachinery` parsing and arithmetic Kubernetes itself uses.

## Install

```sh
go build -o kubectl-unitconv .
sudo mv kubectl-unitconv /usr/local/bin/
```

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

Show every sensible unit for a value (no target unit given):

```sh
kubectl unitconv 500Mi
bytes                524288000
kilobytes (10^3)     524288k
kibibytes (2^10)     512000Ki
megabytes (10^6)     524.288M
mebibytes (2^20)     500Mi
gigabytes (10^9)     0.524288G
gibibytes (2^30)     0.488281Gi
...
```

### Flags

- `--family {auto|cpu|memory}` — force which unit family to use when the
  input is ambiguous (default `auto`; a bare number like `2` defaults to
  memory).
- `--precision int` — decimal places in output (default `6`).

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

# pvc and node have one obvious quantity each
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
  container (default: all containers, including init containers).
- `--requirement {requests|limits}` — with a default-path lookup, limit to
  one requirement (default: both).
