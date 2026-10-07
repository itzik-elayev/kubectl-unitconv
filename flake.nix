{
  description = "kubectl plugin that converts Kubernetes resource.Quantity values between units";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "kubectl-unitconv";
          version = "0.1.0";

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

        devShells.default = pkgs.mkShell {
          packages = [ pkgs.go pkgs.golangci-lint ];
        };
      });
}
