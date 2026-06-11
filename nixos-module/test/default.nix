{
  pkgs ? import <nixpkgs> { },
}:
let
  inherit (pkgs) lib;

  testDrv = pkgs.hello;
  storePrefix = lib.head (lib.split "-" (lib.removePrefix (builtins.storeDir + "/") testDrv.outPath));
in
pkgs.testers.nixosTest {
  name = "nix-cache-beacon-test";

  nodes = {
    machine = {
      imports = [
        ../.
      ];

      nix.settings.trusted-public-keys = [
        (builtins.readFile ./cache.pub)
      ];

      services.nix-cache-beacon = {
        cache.enable = true;
        advert = {
          enable = true;
          port = 5000;
        };
      };

      services.harmonia.cache = {
        signKeyPaths = [ ./cache.secret ];
        enable = true;
      };

      networking.firewall.enable = false;
      system.extraDependencies = [
        testDrv
      ];
    };
  };

  testScript = ''
    start_all()

    machine.wait_for_unit("harmonia.socket")
    machine.wait_for_unit("network.target")

    machine.wait_for_unit("nix-cache-beacon-advert.service")

    # Wait for client cache ready
    machine.wait_until_succeeds("curl --fail localhost:5028/nix-cache-info")

    # Wait for harmonia to be ready
    machine.wait_until_succeeds("curl --fail localhost:5000/${storePrefix}.narinfo")

    # Fetch from client cache
    machine.wait_until_succeeds("curl --fail localhost:5028/${storePrefix}.narinfo")
  '';
}
