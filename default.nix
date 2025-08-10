{ pkgs ? import <nixpkgs> {} }:

pkgs.buildGoModule {
  pname = "minecraft-app-manager";
  version = "1.0.0";
  src = ./.;
  vendorSha256 = null;
  meta = {
    description = "Minecraft server management app";
    license = pkgs.lib.licenses.mit;
    maintainers = [ ];
  };
}
