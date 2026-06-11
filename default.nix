{ buildGoModule, lib }:
buildGoModule {
  pname = "nix-cache-beacon";
  version = "0.1";
  src = lib.cleanSource ./.;
  vendorHash = "sha256-5Sf1DkeQJf+sesfoI/gKswVAnz7LBcuBXzspd9sPTZo=";
  env.CGO_ENABLED = "0";
  ldflags = [
    "-s"
    "-w"
  ];
  meta.mainProgram = "nix-cache-beacon";
}
