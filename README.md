# sing-box XOR builds

This branch is the build controller for XOR-enabled sing-box releases.

It does not carry the upstream sing-box source tree. The release workflow checks
out an upstream SagerNet/sing-box tag, replays the XOR commits from a configured
`xor/v*` branch, builds binaries, and publishes releases with an `xor-` prefix.

XOR source branches are configured in `.github/xor-release.json`.
