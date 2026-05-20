{ buildGoModule, lib }:
buildGoModule {
  pname = "nix-cache-beacon";
  version = "0.1";
  src = lib.cleanSource ./.;
  vendorHash = "sha256-tRTgE521qoGuG4v5eBRUrgXPWMAO7x1jmqvAtlAgico=";
  env.CGO_ENABLED = "0";
  ldflags = [
    "-s"
    "-w"
  ];
  meta.mainProgram = "nix-cache-beacon";
}
