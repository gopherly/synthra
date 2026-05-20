{
  description = "Synthra — Go configuration synthesis library (dev shell)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    git-hooks = {
      url = "github:cachix/git-hooks.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      git-hooks,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };

        devTools = with pkgs; [
          go
          gopls
          gotools
          golangci-lint
          markdownlint-cli
          delve
          git
        ];

        mkApp =
          {
            name,
            description,
            script,
          }:
          {
            type = "app";
            program = toString (pkgs.writeShellScript name script);
            meta = {
              mainProgram = name;
              inherit description;
            };
          };

        mkTaggedRaceTest =
          {
            name,
            description,
            tags,
            coverProfile,
            # With -tags=integration only the module root has *_test.go; listing ./... still
            # selects child packages with no tests and triggers Nix Go covdata failures under -coverpkg.
            integrationTestsAtModuleRoot ? false,
          }:
          let
            goListCmd =
              if integrationTestsAtModuleRoot then
                ''"$go" list -tags=${tags} .''
              else
                ''"$go" list -tags=${tags} ./...'';
          in
          mkApp {
            inherit name description;
            script = ''
              # Nix Go only (go#75031). Example mains under examples/ are not test packages; including
              # them in one -coverpkg=./... run triggers "no such tool covdata" in CI.
              export GOTOOLCHAIN=local
              go="${pkgs.go}/bin/go"
              mapfile -t testpkgs < <(${goListCmd} | grep -vE '/examples(/|$)' || true)
              if [ ''${#testpkgs[@]} -eq 0 ]; then
                echo "go list: no test packages after filters (tags=${tags})" >&2
                exit 1
              fi
              exec "$go" test -tags=${tags} -race -shuffle=on -covermode=atomic \
                -coverpkg=./... -coverprofile=${coverProfile} -timeout 10m "''${testpkgs[@]}"
            '';
          };

        pre-commit-check = git-hooks.lib.${system}.run {
          src = ./.;
          hooks = {
            # git-hooks' default env omits `go` on PATH; golangci-lint needs it.
            golangci-lint = {
              enable = true;
              extraPackages = [ pkgs.go ];
            };
            markdownlint = {
              enable = true;
              excludes = [ "node_modules" ];
              settings.configuration = builtins.fromJSON (builtins.readFile ./.markdownlint.json);
            };
            go-mod-tidy = {
              enable = true;
              name = "go-mod-tidy";
              entry = "${pkgs.go}/bin/go mod tidy";
              files = "(\\.go|go\\.mod|go\\.sum)$";
              pass_filenames = false;
            };
            nixfmt.enable = true;
          };
        };
      in
      {
        formatter = pkgs.nixfmt-tree;

        checks = {
          pre-commit = pre-commit-check;
        };

        devShells.default = pkgs.mkShell {
          name = "synthra";
          packages = devTools ++ pre-commit-check.enabledPackages;
          env = {
            GO111MODULE = "on";
            CGO_ENABLED = "1";
          };
          shellHook = ''
            ${pre-commit-check.shellHook}
            export GOPATH="''${GOPATH:-$HOME/go}"
            export PATH="$GOPATH/bin:$PATH"
            echo "Synthra dev shell — $(go version)"
          '';
        };

        apps = {
          fmt = mkApp {
            name = "fmt";
            description = "Format Go files (gofumpt + gci via golangci-lint)";
            script = ''
              exec ${pkgs.golangci-lint}/bin/golangci-lint fmt ./...
            '';
          };

          tidy = mkApp {
            name = "tidy";
            description = "Run go mod tidy for the module";
            script = ''
              exec ${pkgs.go}/bin/go mod tidy
            '';
          };

          lint = mkApp {
            name = "lint";
            description = "Run golangci-lint";
            script = ''
              exec ${pkgs.golangci-lint}/bin/golangci-lint run ./...
            '';
          };

          lint-md = mkApp {
            name = "lint-md";
            description = "Lint Markdown files with markdownlint";
            script = ''
              exec ${pkgs.markdownlint-cli}/bin/markdownlint '**/*.md'
            '';
          };

          test-unit = mkTaggedRaceTest {
            name = "test-unit";
            description = "Run unit tests with race detector; write coverage-unit.out (build tag !integration)";
            tags = "!integration";
            coverProfile = "coverage-unit.out";
          };

          test-integration = mkTaggedRaceTest {
            name = "test-integration";
            description = "Run integration tests with race detector; write coverage-integration.out (build tag integration)";
            tags = "integration";
            coverProfile = "coverage-integration.out";
            integrationTestsAtModuleRoot = true;
          };
        };
      }
    );
}
