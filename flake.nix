{
  description = "SRO development flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
  };

  outputs = inputs @ {
    self,
    nixpkgs,
    flake-parts,
    ...
  }:
    flake-parts.lib.mkFlake {inherit inputs;} {
      systems = ["x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin"];

      perSystem = {
        config,
        self',
        inputs',
        pkgs,
        system,
        ...
      }: let
        version =
          if (self ? dirtyShortRev)
          then "${self.dirtyShortRev}-dirty"
          else if (self ? shortRev)
          then self.shortRev
          else "dev";
        go = pkgs.go_1_26;
        buildGoModule = pkgs.buildGoModule.override {inherit go;};
      in {
        packages.default = buildGoModule {
          pname = "sro";
          inherit version;
          src = ./.;

          vendorHash = "sha256-FmrCSNInn0fmtN2pwQ5gmFcXAGTWZ0t7KxEkiXqSTJI=";

          env.CGO_ENABLED = "0";

          ldflags = [
            "-s"
            "-w"
            "-X github.com/infraflakes/sro/cmd.version=${version}"
          ];

          nativeBuildInputs = [pkgs.installShellFiles pkgs.git];

          postInstall = ''
            installShellCompletion --cmd sro \
              --bash completions/sro.bash \
              --fish completions/sro.fish \
              --zsh completions/sro.zsh
          '';
        };

        devShells.default = pkgs.mkShell {
          packages = [
            go
            pkgs.golangci-lint
            pkgs.cmake
            pkgs.goreleaser
          ];
          shellHook = ''
            export GOPATH="$TMPDIR/.go"
            export PATH="$GOPATH/bin:$PATH"
          '';
        };
      };
    };
}
