{
  description = "qlimaster - keyboard-driven pub-quiz score manager";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    flake-checks.url = "github:kradalby/flake-checks";
    flake-checks.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, flake-utils, flake-checks }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        fc = flake-checks.lib;

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

        common = {
          inherit pkgs;
          root = ./.;
          pname = "qlimaster";
          inherit version;
          # vendorHash is the sha256 of the fetched Go module cache. Bump
          # this after changing go.sum (`nix build` will print the new
          # hash in the error output).
          vendorHash = "sha256-wOrV1/uPP3Nu3J5g7i+7D7oHNzNgPtP8aiZpPJDKjKs=";
          goPkg = go;
        };
      in
      {
        packages = {
          default = fc.goBuild common;
        };

        apps.default = flake-utils.lib.mkApp {
          drv = fc.goBuild common;
        };

        formatter = fc.formatter common;

        checks = {
          build = fc.goBuild common;
          gotest = fc.goTest (common // { goRace = false; });
          gotest-race = fc.goTest (common // { goRace = true; });
          golangci-lint = fc.goLint common;
          formatting = fc.goFormat common;
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            go-tools # staticcheck
            golangci-lint
            delve
            gotestsum
            git
            gh
            prek
            goreleaser
          ];
          shellHook = ''
            echo "qlimaster dev shell"
            echo "  go:            $(go version | awk '{print $3}')"
            echo "  golangci-lint: $(golangci-lint --version 2>/dev/null | head -1)"
          '';
        };
      });
}
