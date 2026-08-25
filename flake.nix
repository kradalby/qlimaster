{
  description = "qlimaster - keyboard-driven pub-quiz score manager";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    flake-checks.url = "github:kradalby/flake-checks";
    flake-checks.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      flake-checks,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        # Every Go tool in this repo must agree on one toolchain. Bare
        # `pkgs.go` still resolves to 1.26 in unstable, so go_latest /
        # buildGoLatestModule are named explicitly and the tools are rebuilt
        # against them. goimports (gotools) ships wrapped with a `go` on
        # PATH; that `go` must be at least the go.mod directive or
        # GOTOOLCHAIN=auto tries to fetch a toolchain from inside the
        # network-less treefmt sandbox.
        goOverlay = _: prev: {
          gotools = prev.gotools.override {
            buildGoModule = prev.buildGoLatestModule;
            go = prev.go_latest;
          };
          gotestsum = prev.gotestsum.override {
            buildGoModule = prev.buildGoLatestModule;
          };
        };

        pkgs = import nixpkgs {
          inherit system;
          overlays = [ goOverlay ];
        };
        fc = flake-checks.lib;

        # Track the newest Go nixpkgs ships rather than pinning a version, so
        # this keeps working across bumps. Fail loudly if it ever regresses
        # below what go.mod requires.
        go =
          let
            g = pkgs.go_latest or (throw "pkgs.go_latest is not available in the pinned nixpkgs.");
          in
          if builtins.compareVersions g.version "1.27.0" >= 0 then
            g
          else
            throw ''
              qlimaster requires Go 1.27 or newer.
              pkgs.go_latest is ${g.version}.
              Bump nixpkgs to a revision that includes Go 1.27 (nixos-unstable).
            '';

        version =
          if self ? rev then
            builtins.substring 0 7 self.rev
          else if self ? dirtyRev then
            (builtins.substring 0 7 self.dirtyRev) + "-dirty"
          else
            "dev";

        common = {
          inherit pkgs;
          root = ./.;
          pname = "qlimaster";
          inherit version;
          # vendorHash is the sha256 of the fetched Go module cache. Bump
          # this after changing go.sum (`nix build` will print the new
          # hash in the error output).
          vendorHash = "sha256-CQBKNvUBJS5aJ39ao5jMtmhiyRlxGZDYI0g4r6zKygs=";
          goPkg = go;
        };
      in
      {
        packages = {
          default = fc.goBuild common;
        };

        apps = {
          default = flake-utils.lib.mkApp { drv = fc.goBuild common; };
          test = flake-utils.lib.mkApp {
            drv = pkgs.writeShellApplication {
              name = "qlimaster-test";
              runtimeInputs = [
                go
                pkgs.gcc
              ]; # -race needs cgo + a C compiler
              text = ''
                export CGO_ENABLED=1
                exec go test -race -cover ./...
              '';
            };
          };
          lint = flake-utils.lib.mkApp {
            drv = pkgs.writeShellApplication {
              name = "qlimaster-lint";
              runtimeInputs = [ pkgs.golangci-lint ];
              text = ''
                export CGO_ENABLED=0
                exec golangci-lint run --timeout=5m ./...
              '';
            };
          };
          fuzz = flake-utils.lib.mkApp {
            drv = pkgs.writeShellApplication {
              name = "qlimaster-fuzz";
              runtimeInputs = [ go ];
              # -fuzz must match exactly one target; pass a name + duration:
              #   nix run .#fuzz -- FuzzParse 60s
              text = ''
                export CGO_ENABLED=0
                exec go test -fuzz="''${1:-FuzzParse}" -fuzztime="''${2:-30s}" ./score
              '';
            };
          };
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
      }
    );
}
