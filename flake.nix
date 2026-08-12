{
  description = "sqls - an implementation of the Language Server Protocol for SQL";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      inherit (nixpkgs) lib;

      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      forAllSystems = f: lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});

      # Stamped into the binary as `sqls -v` output.
      revision = self.shortRev or self.dirtyShortRev or "unknown";

      mkSqls =
        pkgs:
        pkgs.buildGoModule {
          pname = "sqls";
          # Keep in sync with the `version` const in main.go.
          version = "0.2.48";

          src = lib.cleanSource ./.;

          vendorHash = "sha256-p8s0uLE1bIjL/P52umLIer85BarpWXluppzmnA8WSwc=";

          # The oracle (godror) and sqlite3 drivers are cgo-only.
          env.CGO_ENABLED = "1";

          ldflags = [
            "-s"
            "-w"
            "-X main.revision=${revision}"
          ];

          # The config package resolves its default path from $HOME at init time.
          preCheck = ''
            export HOME="$TMPDIR"
          '';

          meta = {
            description = "Implementation of the Language Server Protocol for SQL";
            homepage = "https://github.com/sqls-server/sqls";
            license = lib.licenses.mit;
            mainProgram = "sqls";
            platforms = systems;
          };
        };
    in
    {
      overlays.default = _final: prev: {
        sqls = mkSqls prev;
      };

      packages = forAllSystems (pkgs: {
        sqls = mkSqls pkgs;
        default = mkSqls pkgs;
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          inputsFrom = [ (mkSqls pkgs) ];

          # Azure AD connections authenticate through `az login`, but the CLI is
          # deliberately left out: it is not a build tool and pulls in a large
          # python closure. Add pkgs.azure-cli if `az` is not already on PATH.
          packages = [
            pkgs.go
            pkgs.gopls
            pkgs.gotools # goimports
            pkgs.go-tools # staticcheck
            pkgs.golangci-lint
            pkgs.delve
            pkgs.git
            pkgs.gnumake
          ];
        };
      });

      formatter = forAllSystems (pkgs: pkgs.nixfmt);
    };
}
