{
  description = "kubectl plugin that converts Kubernetes resource.Quantity values between units";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem
      (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          k9sPluginsYaml = (pkgs.formats.yaml { }).generate "plugins.yaml" {
            plugins = import ./nix/k9s-plugins.nix;
          };
        in
        {
          packages.default = pkgs.buildGoModule {
            pname = "kubectl-unitconv";
            version = "1.0.0"; # x-release-please-version

            src = ./.;

            # Keep in sync with go.sum; `nix build` reports the expected value
            # on mismatch.
            vendorHash = "sha256-LJ/uTn9ZXNobrLCqDpn9C4H4XIt23RjNrQh4uNZy9b0=";

            meta = with pkgs.lib; {
              description = "Convert Kubernetes resource.Quantity values between units";
              homepage = "https://github.com/itzik-elayev/kubectl-unitconv";
              license = licenses.asl20;
              mainProgram = "kubectl-unitconv";
            };
          };

          # A ready-to-merge k9s plugins.yaml, for non-home-manager setups.
          packages.k9s-plugins = k9sPluginsYaml;

          checks.k9s-plugins-in-sync = pkgs.runCommand "k9s-plugins-in-sync" { nativeBuildInputs = [ pkgs.yq-go ]; } ''
            normalize() { yq -o=json 'sort_keys(..)' "$1"; }
            if ! diff <(normalize ${./examples/k9s/plugins.yaml}) <(normalize ${k9sPluginsYaml}); then
              echo "examples/k9s/plugins.yaml drifted from nix/k9s-plugins.nix" >&2
              exit 1
            fi
            touch $out
          '';

          devShells.default = pkgs.mkShell {
            packages = [ pkgs.go pkgs.golangci-lint ];
          };
        })
    // {
      homeManagerModules.default = import ./nix/hm-module.nix self;
    };
}
