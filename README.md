# nix-cache-beacon - mDNS discovery for Nix binary caches

_Status_: Alpha.

`nix-cache-beacon` is a binary cache that uses mDNS service discovery to announce & find caches on the local network & races gets against discovered caches, turning your entire network of Nix nodes into a distributed binary cache.

## Security

`nix-cache-beacon` doesn't change the security model of package substitutions. Packages still need to be signed by a trusted key.
During cache racing package signatures are checked before package metadata is returned.

Traffic is _unencrypted_ & has potential privacy implications. meaning that nodes on the network can monitor for what you try to substitute.

If using the NixOS module the same cryptographic keys as your NixOS system will automatically be used.

## Usage

The primary way to use `nix-cache-beacon` is via it's NixOS module.

```nix
{ ... }:
{
  services.nix-cache-beacon = {
    # Announce cache to the local network
    advert = {
      enable = true;
      port = 5000; # Harmonia port
    };

    # Enable local binary cache using discovered caches on the local network
    cache.enable = true;
  };

  # Make Nix aware of our local network cache
  nix.settings.substituters = [ "http://localhost:5028" ];

  # Local binary cache using Harmonia
  # nix-cache-beacon can be used with any cache implementation
  services.harmonia.cache.enable = true; # Serve up local Nix store
  networking.firewall.allowedTCPPorts = [ 5000 ]; # Open firewall port for Harmonia
}
```

## License

- The application is licensed under `GPL-3.0-or-later`
- Nix expressions are licensed under `MIT`
