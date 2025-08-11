{
  description =
    "Minecraft App Manager - Go service for managing Minecraft servers";

  inputs = { nixpkgs.url = "nixpkgs/nixos-unstable"; };

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
    in {
      packages.${system}.minecraft-app-manager = pkgs.buildGoModule {
        pname = "minecraft-app-manager";
        version = "1.0.0";
        src = ./.;
        vendorHash = null;

        nativeBuildInputs = [ pkgs.makeWrapper ];
        buildInputs = [ pkgs.bash pkgs.busybox pkgs.openjdk17 ];

        postInstall = ''
          for bin in $out/bin/*; do
            wrapProgram "$bin" \
              --prefix PATH : ${
                pkgs.lib.makeBinPath [ pkgs.bash pkgs.busybox pkgs.openjdk17 ]
              }
          done
        '';
      };

      defaultPackage.${system} = self.packages.${system}.minecraft-app-manager;

      devShells.${system}.default = pkgs.mkShell {
        buildInputs = with pkgs; [ go gopls bash busybox openjdk17 ];
      };
    };
}
