# sing-box XOR builds

This repository builds XOR-enabled sing-box releases.

The default branch is only a build controller. It does not carry the upstream
sing-box source tree. The release workflow checks out an upstream
`SagerNet/sing-box` tag, replays the XOR commits from a configured `xor/v*`
branch, builds binaries, and publishes releases with an `xor-` prefix.

XOR source branches are configured in `xor-release.json`.

## Branch model

- `xor-release-controller`: workflow and release manifest only.
- `xor/v1.13`: XOR code based on upstream `v1.13.13`.
- `xor/v1.14`: XOR code based on upstream `v1.14.0-alpha.30`.

The workflow selects the newest manifest entry whose version range contains the
upstream tag. Non-stable upstream tags such as `v1.14.0-alpha.31` are published
as GitHub pre-releases and are not marked as Latest.

## Client usage

Use an `xor` outbound as the lower transport for another protocol through
`detour`. For example, a WireGuard endpoint can send packets to an XOR-wrapped
server:

```json
{
  "endpoints": [
    {
      "type": "wireguard",
      "tag": "wg-client",
      "address": "192.168.1.1/32",
      "private_key": "CLIENT_PRIVATE_KEY",
      "detour": "xor-out",
      "peers": [
        {
          "address": "127.0.0.1",
          "port": 51822,
          "public_key": "SERVER_PUBLIC_KEY",
          "allowed_ips": "0.0.0.0/0",
          "persistent_keepalive_interval": 15
        }
      ]
    }
  ],
  "outbounds": [
    {
      "type": "xor",
      "tag": "xor-out",
      "xor-to": 123,
      "xor-length": 16
    },
    {
      "type": "direct",
      "tag": "direct"
    }
  ]
}
```

`xor-to` is required and is the byte value used for XOR. `xor-length` is
optional; when set, only the first N bytes of each packet are XORed.

## Server usage

For WireGuard endpoint server mode, use an XOR bind. The XOR bind owns the UDP
listen socket, so the endpoint does not need a separate `listen_port`.

```json
{
  "endpoints": [
    {
      "type": "wireguard",
      "tag": "wg-server",
      "address": "192.0.2.1/24",
      "private_key": "SERVER_PRIVATE_KEY",
      "bind": {
        "type": "xor",
        "listen": "0.0.0.0",
        "listen_port": 51822,
        "xor-to": 123,
        "xor-length": 16
      },
      "peers": [
        {
          "public_key": "CLIENT_PUBLIC_KEY",
          "allowed_ips": [
            "192.168.1.1/32"
          ]
        }
      ]
    }
  ],
  "outbounds": [
    {
      "type": "direct",
      "tag": "direct"
    }
  ],
  "route": {
    "final": "direct"
  }
}
```

Do not set `persistent_keepalive_interval` on a server peer unless the peer also
has a fixed endpoint address. Without a known endpoint, WireGuard cannot
actively send keepalive packets before the client contacts the server.

## Transparent UDP XOR

To transparently XOR only one UDP destination through TUN, route only that IP
into the TUN inbound and match the target port in sing-box routing:

```json
{
  "inbounds": [
    {
      "type": "tun",
      "tag": "tun-in",
      "interface_name": "sing-box-xor",
      "address": [
        "198.18.0.1/30"
      ],
      "auto_route": true,
      "strict_route": true,
      "route_address": [
        "203.0.113.10/32"
      ],
      "stack": "system"
    }
  ],
  "outbounds": [
    {
      "type": "xor",
      "tag": "xor-out",
      "xor-to": 123,
      "xor-length": 16
    },
    {
      "type": "direct",
      "tag": "direct"
    }
  ],
  "route": {
    "rules": [
      {
        "inbound": "tun-in",
        "network": "udp",
        "ip_cidr": "203.0.113.10/32",
        "port": 12345,
        "outbound": "xor-out"
      },
      {
        "inbound": "tun-in",
        "outbound": "direct"
      }
    ],
    "final": "direct"
  }
}
```

System routing can only capture by IP prefix, not by port. Therefore other
ports on the same IP may enter sing-box, but the second rule sends them direct
without XOR.
