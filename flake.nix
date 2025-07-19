{
  description = "A development shell for go";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    treefmt-nix.url = "github:numtide/treefmt-nix";
    treefmt-nix.inputs.nixpkgs.follows = "nixpkgs";
  };
  outputs = {
    nixpkgs,
    treefmt-nix,
    ...
  }: let
    supportedSystems = [
      "x86_64-linux"
      "x86_64-darwin"
      "aarch64-linux"
      "aarch64-darwin"
    ];
    forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
  in {
    devShells = forAllSystems (system: let
      pkgs = import nixpkgs {
        inherit system;
      };

      rooted = exec:
        builtins.concatStringsSep "\n"
        [
          ''
            REPO_ROOT="$(git rev-parse --show-toplevel)"
          ''
          exec
        ];

      setup = exec:
        builtins.concatStringsSep "\n"
        [
          ''
            go mod tidy
            go mod download
            templ generate
          ''
          exec
        ];

      scripts = {
        dx = {
          exec = ''$EDITOR "$REPO_ROOT"/flake.nix'';
          description = "Edit flake.nix";
        };
        gx = {
          exec = ''$EDITOR "$REPO_ROOT"/go.mod'';
          description = "Edit go.mod";
        };
        lint = {
          exec = rooted (setup ''
            #!/usr/bin/env bash

            # Exit on undefined variables and pipe failures (except for the linting commands)
            set -u
            set -o pipefail

            # Get the repository root (adjust this based on your needs)
            REPO_ROOT="''${REPO_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}"

            # Initialize array to store exit codes
            declare -a exit_codes=()
            declare -a command_names=()

            # Function to run a command and store its exit code
            run_lint_step() {
                local cmd_name="$1"
                shift
                local cmd="$@"

                echo "Running: $cmd_name"
                echo "Command: $cmd"

                # Run the command and capture exit code
                $cmd
                local exit_code=$?

                # Store exit code and command name
                exit_codes+=($exit_code)
                command_names+=("$cmd_name")

                if [ $exit_code -eq 0 ]; then
                    echo "✓ $cmd_name passed"
                else
                    echo "✗ $cmd_name failed with exit code $exit_code"
                fi
                echo
            }

            # Run all linting steps
            run_lint_step "basedpyright hooks" basedpyright "$REPO_ROOT"/.claude/hooks/*.py
            run_lint_step "templ" templ generate "$REPO_ROOT"
            run_lint_step "golangci-lint" golangci-lint run --fix "$REPO_ROOT"/...
            run_lint_step "statix" statix check "$REPO_ROOT"/flake.nix
            run_lint_step "deadnix" deadnix "$REPO_ROOT"/flake.nix

            # Calculate final exit code
            final_exit_code=0
            failed_commands=()

            echo "=== Summary ==="
            for i in "''${!exit_codes[@]}"; do
                if [ "''${exit_codes[$i]}" -ne 0 ]; then
                    final_exit_code=1
                    failed_commands+=("''${command_names[$i]}")
                    echo "✗ ''${command_names[$i]}: exit code ''${exit_codes[$i]}"
                else
                    echo "✓ ''${command_names[$i]}: success"
                fi
            done

            echo
            if [ $final_exit_code -eq 0 ]; then
                echo "All linting steps passed! 🎉"
            else
                echo "The following linting steps failed:"
                printf ' - %s\n' "''${failed_commands[@]}"
                echo
                echo "Please fix the issues and try again."
            fi

            exit $final_exit_code
          '');
          type = "script";
          deps = with pkgs; [golangci-lint git statix deadnix templ];
          description = "Run golangci-lint, statix, and deadnix";
        };
      };

      scriptPackages =
        pkgs.lib.mapAttrs
        (
          name: script: let
            scriptType = script.type or "app";
          in
            if scriptType == "script"
            then pkgs.writeShellScriptBin name script.exec
            else
              pkgs.writeShellApplication {
                inherit name;
                bashOptions = scripts.baseOptions or ["errexit" "pipefail" "nounset"];
                text = script.exec;
                runtimeInputs = script.deps or [];
              }
        )
        scripts;

      buildWithSpecificGo = pkg: pkg.override {buildGoModule = pkgs.buildGo124Module;};
    in {
      default = pkgs.mkShell {
        name = "dev";

        # Available packages on https://search.nixos.org/packages
        packages = with pkgs;
          [
            alejandra # Nix
            nixd
            statix
            deadnix

            go_1_24 # Go Tools
            air
            golangci-lint
            gopls
            (buildWithSpecificGo revive)
            (buildWithSpecificGo golines)
            (buildWithSpecificGo golangci-lint-langserver)
            (buildWithSpecificGo gomarkdoc)
            (buildWithSpecificGo gotests)
            (buildWithSpecificGo gotools)
            (buildWithSpecificGo reftools)
            pprof
            graphviz
            goreleaser
            cobra-cli
            templ
          ]
          ++ builtins.attrValues scriptPackages;

        shellHook = ''
          export REPO_ROOT=$(git rev-parse --show-toplevel)
        '';
      };
    });

    packages = forAllSystems (system: let
      pkgs = import nixpkgs {
        inherit system;
      };
    in {
      # default = pkgs.buildGoModule {
      #   pname = "my-go-project";
      #   version = "0.0.1";
      #   src = ./.;
      #   vendorHash = "";
      #   doCheck = false;
      #   meta = with pkgs.lib; {
      #     description = "My Go project";
      #     homepage = "https://github.com/conneroisu/my-go-project";
      #     license = licenses.asl20;
      #     maintainers = with maintainers; [connerohnesorge];
      #   };
      # };
    });

    formatter = forAllSystems (system: let
      pkgs = nixpkgs.legacyPackages.${system};
      treefmtModule = {
        projectRootFile = "flake.nix";
        programs = {
          alejandra.enable = true; # Nix formatter
        };
      };
    in
      treefmt-nix.lib.mkWrapper pkgs treefmtModule);
  };
}
