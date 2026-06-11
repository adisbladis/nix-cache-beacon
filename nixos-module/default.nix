{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.services.nix-cache-beacon;

  configFile = pkgs.writeText "nix-cache-beacon-config.json" (
    lib.generators.toJSON { } (
      {
        keys = config.nix.settings.trusted-public-keys;
      }
      // lib.optionalAttrs (cfg.cache.cacheInfo != { }) { inherit (cfg.cache) cacheInfo; }
      // lib.optionalAttrs (cfg.cache.timeout != null) { inherit (cfg.cache) timeout; }
    )
  );

  package = pkgs.callPackage ../. { };

in
{
  options.services.nix-cache-beacon = {
    package = lib.mkOption {
      type = lib.types.package;
      default = package;
      defaultText = lib.literalExpression "pkgs.nix-cache-beacon";
      description = ''
        nix-cache-beacon package to use.
      '';
    };

    cache = {
      enable = lib.mkEnableOption "nix-cache-beacon cache server";

      addSubstituter = lib.mkEnableOption "nix-cache-beacon cache server" // {
        default = true;
      };

      timeout = lib.mkOption {
        type = lib.types.nullOr lib.types.float;
        default = null;
        example = 3.0;
        description = "Request timeout in seconds.";
      };

      cacheInfo = lib.mkOption {
        type = lib.types.attrs;
        default = { };
        example = lib.literalExpression ''
          {
            StoreDir      = "/nix/store";
            WantMassQuery = 1;
            Priority      = 40;
          }
        '';
        description = "Overriden nix-cache-info attributes.";
      };

      listenAddresses = lib.mkOption {
        type = lib.types.listOf lib.types.str;
        default = [ "[::]:5028" ];
        description = ''
          Addresses for the cache server to listen on.
        '';
      };

      verbose = lib.mkOption {
        type = lib.types.bool;
        default = false;
        description = ''
          Enable debug logs.
        '';
      };
    };

    advert = {
      enable = lib.mkEnableOption "nix-cache-beacon advert service";

      port = lib.mkOption {
        type = lib.types.port;
        description = "Port number of the cache server to advertise.";
      };

      hostname = lib.mkOption {
        type = lib.types.nullOr lib.types.str;
        description = "Hostname of the cache server to advertise.";
        default = null;
      };
    };
  };

  config =
    let
      commonServiceConfig = {
        DynamicUser = true;
        Restart = "on-failure";
        RestartSec = "5s";
        NoNewPrivileges = true;
        PrivateTmp = true;
        PrivateDevices = true;
        PrivateUsers = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        ProtectProc = "invisible";
        ProtectHostname = true;
        ProtectClock = true;
        ProtectControlGroups = true;
        ProtectKernelLogs = true;
        ProtectKernelTunables = true;
        RestrictRealtime = true;
        CapabilityBoundingSet = "";
        RestrictAddressFamilies = [
          "AF_INET"
          "AF_INET6"
          "AF_NETLINK"
        ];
        RestrictNamespaces = true;
        LockPersonality = true;
        MemoryDenyWriteExecute = true;
        SystemCallFilter = [ "@system-service" ];
        SystemCallArchitectures = "native";
      };
    in
    lib.mkMerge [
      (lib.mkIf cfg.cache.enable {
        systemd.services.nix-cache-beacon-cache = {
          description = "nix-cache-beacon Nix binary cache server";
          wantedBy = [ "multi-user.target" ];
          after = [ "network.target" ];
          serviceConfig = commonServiceConfig // {
            ExecStart =
              let
                listenArgs = lib.concatMapStringsSep " " (
                  addr: "--listen ${lib.escapeShellArg addr}"
                ) cfg.cache.listenAddresses;
              in
              "${lib.getExe cfg.package} cache ${listenArgs} --config ${configFile} ${lib.optionalString cfg.cache.verbose "--verbose"}";
          };
        };
      })

      (lib.mkIf cfg.advert.enable {
        systemd.services.nix-cache-beacon-advert = {
          description = "nix-cache-beacon mDNS advertisement";
          wantedBy = [ "multi-user.target" ];
          after = [
            "network.target"
          ];
          serviceConfig = commonServiceConfig // {
            ExecStart = "${lib.getExe cfg.package} advert --port ${toString cfg.advert.port} ${
              lib.optionalString (cfg.advert.hostname != null) "--hostname ${cfg.advert.hostname}"
            }";
          };
        };
      })
    ];
}
