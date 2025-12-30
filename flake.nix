{
  description = "Waller - A GTK-based Wallpaper Manager for Wayland";

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
          pname = "waller";
          version = "0.1.0";
          src = ./.;

          vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";

          nativeBuildInputs = with pkgs; [
            pkg-config
            wrapGAppsHook3
          ];

          buildInputs = with pkgs; [
            # GTK & Layer Shell Dependencies
            gtk3
            gtk-layer-shell
            glib
            cairo
            pango
            gobject-introspection
            zlib
            gsettings-desktop-schemas
            hicolor-icon-theme
          ];

          tags = [ "gtk_3_24" ];

          postInstall = ''
            mkdir -p $out/share/icons/hicolor/512x512/apps
            cp icon.png $out/share/icons/hicolor/512x512/apps/waller.png
            
            mkdir -p $out/share/applications
            cat <<EOF > $out/share/applications/waller.desktop
            [Desktop Entry]
            Name=Waller
            Comment=Wallpaper Manager
            Exec=$out/bin/waller
            Icon=waller
            Terminal=false
            Type=Application
            Categories=Utility;
            EOF
          '';
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools
            pkg-config
            wrapGAppsHook3

            # GTK & Layer Shell Dependencies
            gtk3
            gtk-layer-shell
            glib
            cairo
            pango
            gobject-introspection
            zlib
            gsettings-desktop-schemas
            hicolor-icon-theme
          ];

          shellHook = ''
            echo "Welcome to the Waller dev environment!"
            echo "GTK3 and Layer Shell dependencies are loaded."
            
            # Manually set XDG_DATA_DIRS to include schemas for the dev shell
            export XDG_DATA_DIRS=${pkgs.gsettings-desktop-schemas}/share/gsettings-schemas/${pkgs.gsettings-desktop-schemas.name}:${pkgs.gtk3}/share/gsettings-schemas/${pkgs.gtk3.name}:$XDG_DATA_DIRS
          '';
        };


        apps.default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/waller";
        };
      }
    ) // {
      overlays.default = final: prev: {
        waller = self.packages.${prev.stdenv.hostPlatform.system}.default;
      };
    };
}
