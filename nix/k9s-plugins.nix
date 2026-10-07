# k9s plugin definitions. examples/k9s/plugins.yaml mirrors these for
# non-Nix users; `nix flake check` fails if the two drift apart.
let
  # Arguments arrive as sh positional parameters rather than being spliced
  # into the script, so names are quoted safely whatever they contain.
  pagedUnitconv = family: target: ''
    out=$(kubectl unitconv --from pod/"$1" --family ${family} ${target} --output table --color always --context "$2" ''${3:+--namespace "$3"} ''${4:+--kubeconfig "$4"} 2>&1)
    printf '%s\n' "$out" | less -R
  '';

  podPlugin = { shortCut, description, family, target }: {
    inherit shortCut description;
    scopes = [ "pods" ];
    background = false;
    confirm = false;
    command = "sh";
    args = [ "-c" (pagedUnitconv family target) "_" "$NAME" "$CONTEXT" "$NAMESPACE" "$KUBECONFIG" ];
  };
in
{
  unitconv-memory = podPlugin {
    shortCut = "Shift-M";
    description = "Memory requests/limits (Gi)";
    family = "memory";
    target = "Gi";
  };

  unitconv-cpu = podPlugin {
    shortCut = "Shift-C";
    description = "CPU requests/limits (cores)";
    family = "cpu";
    target = "cores";
  };
}
