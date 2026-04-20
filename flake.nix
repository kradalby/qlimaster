{
  description = "qlimaster - keyboard-driven pub-quiz score manager";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };

        # Require Go 1.26. Fail loudly if nixpkgs does not provide it.
        go = pkgs.go_1_26 or (throw ''
          qlimaster requires Go 1.26.
          pkgs.go_1_26 is not available in the pinned nixpkgs.
          Bump nixpkgs to a revision that includes Go 1.26 (nixos-unstable).
        '');

        version =
          if self ? rev then builtins.substring 0 7 self.rev
          else if self ? dirtyRev then (builtins.substring 0 7 self.dirtyRev) + "-dirty"
          else "dev";

        # Use buildGoModule pinned to Go 1.26 so the flake's toolchain
        # matches go.mod regardless of the default Go in nixpkgs.
        buildGoModule126 =
          if pkgs ? buildGo126Module
          then pkgs.buildGo126Module
          else pkgs.buildGoModule.override { inherit go; };

        qlimaster = buildGoModule126 {
          pname = "qlimaster";
          inherit version;
          src = ./.;
          # vendorHash is the sha256 of the fetched Go module cache. Bump
          # this after changing go.sum (`nix build` will print the new
          # hash in the error output).
          vendorHash = "sha256-BClHLKlC/fn3cwcD6MNtteDvmAOiWJ58gGrpqwdAtiM=";
          subPackages = [ "cmd/qlimaster" ];
          env.CGO_ENABLED = "0";
          ldflags = [ "-s" "-w" "-X main.version=${version}" ];
          meta = with pkgs.lib; {
            description = "Keyboard-driven pub-quiz score manager TUI";
            mainProgram = "qlimaster";
            license = licenses.bsd3;
            platforms = platforms.unix;
          };
        };
      in {
        packages = {
          default = qlimaster;
          qlimaster = qlimaster;
        };

        apps.default = flake-utils.lib.mkApp {
          drv = qlimaster;
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            go-tools          # staticcheck
            golangci-lint
            delve
            gotestsum
            git
            gh
            prek
          ];
          shellHook = ''
            echo "qlimaster dev shell"
            echo "  go:            $(go version | awk '{print $3}')"
            echo "  golangci-lint: $(golangci-lint --version 2>/dev/null | head -1)"
          '';
        };

        checks.default = qlimaster;
      });
}
