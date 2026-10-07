self:
{ config, lib, pkgs, ... }:
let
  cfg = config.programs.kubectl-unitconv;
in
{
  options.programs.kubectl-unitconv = {
    enable = lib.mkEnableOption "kubectl-unitconv, a kubectl plugin converting resource.Quantity units";

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
      defaultText = lib.literalExpression "kubectl-unitconv.packages.\${system}.default";
      description = "The kubectl-unitconv package to install.";
    };

    k9sPlugins.enable = lib.mkOption {
      type = lib.types.bool;
      default = config.programs.k9s.enable;
      defaultText = lib.literalExpression "config.programs.k9s.enable";
      description = ''
        Add k9s pod-view shortcuts (Shift-M memory in Gi, Shift-C cpu in
        cores) to programs.k9s.plugins. Requires `less` on PATH.
      '';
    };
  };

  config = lib.mkIf cfg.enable {
    home.packages = [ cfg.package ];

    programs.k9s.plugins = lib.mkIf cfg.k9sPlugins.enable (import ./k9s-plugins.nix);
  };
}
