{
    description = "Python development environment";

    inputs = {
        nixpkgs.url = "github:Nixos/nixpkgs/nixos-unstable";
    };

    outputs =
        { self, nixpkgs }:
        let
            system = "x86_64-linux";
            pkgs = nixpkgs.legacyPackages.${system};
        in
        {
            devShells.${system}.default = pkgs.mkShell {
                buildInputs = with pkgs; [
                    python3
                    python3Packages.pip
                    python3Packages.virtualenv
                ];

                shellHook = ''
                    	      export NIX_SHELL_NAME='cpm-executor'

                              if [ ! -d ".venv" ]; then
                                python -m venv .venv
                              fi

                              source .venv/bin/activate

                              pip install --upgrade pip
                '';
            };
        };
}
