flake:
{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.programs.tmux-sessions;
  system = pkgs.stdenv.hostPlatform.system;
  toml = pkgs.formats.toml { };
in
{
  options.programs.tmux-sessions = {
    enable = lib.mkEnableOption "tmux-sessions";

    package = lib.mkOption {
      type = lib.types.package;
      default = flake.packages.${system}.default;
      description = "The tmux-sessions package to install.";
    };

    projectDirectories = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      description = "Directories to scan for git repositories.";
    };
  };

  config = lib.mkIf cfg.enable {
    home.packages = [ cfg.package ];

    xdg.configFile."tmux-sessions/config.toml".source = toml.generate "config.toml" {
      projectDirectories = cfg.projectDirectories;
    };
  };
}
