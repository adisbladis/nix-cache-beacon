{
  pkgs ? import <nixpkgs> { },
}:
pkgs.mkShell {
  env.CGO_ENABLED = "0";
  packages = [
    pkgs.go
  ];
}
