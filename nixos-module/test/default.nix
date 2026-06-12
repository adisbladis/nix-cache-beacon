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
    server = {
      imports = [
        ../.
      ];

      services.nix-cache-beacon.advert = {
        enable = true;
        port = 5000;
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

    client = {
      imports = [
        ../.
      ];

      # Resolve .local hostnames over mDNS
      services.avahi = {
        enable = true;
        nssmdns4 = true;
        ipv4 = true;
        ipv6 = true;
      };

      nix.settings.trusted-public-keys = [
        (builtins.readFile ./cache.pub)
      ];

      services.nix-cache-beacon.cache.enable = true;

      networking.firewall.enable = false;
    };
  };

  testScript = ''
    start_all()

    server.wait_for_unit("harmonia.socket")
    server.wait_for_unit("nix-cache-beacon-advert.service")

    client.wait_for_unit("nix-cache-beacon-cache.service")

    # Wait for client cache ready
    client.wait_until_succeeds("curl --fail localhost:5028/nix-cache-info")

    # Wait for harmonia to be ready
    server.wait_until_succeeds("curl --fail localhost:5000/${storePrefix}.narinfo")

    # Cross-host mDNS hostname resolution
    client.wait_until_succeeds("getent hosts server.local")

    # Fetch narinfo from the server's cache, discovered via mDNS
    client.wait_until_succeeds("curl --fail localhost:5028/${storePrefix}.narinfo")
  '';
}
