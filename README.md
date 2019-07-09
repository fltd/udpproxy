# udpproxy

udpproxy is a Server Type plugin for Caddy [https://caddyserver.com](https://caddyserver.com), it is modified from [https://github.com/pieterlouw/caddy-net](https://github.com/pieterlouw/caddy-net) by @pieterlouw.

It proxies UDP traffic to a specified destination, and listen the reply on multiple addresses (defined via the `reply-addr-alias` directive in `Caddyfile`).

It helps in the situation when a service replies via a different interface (and probably with a different source IP address) than where it receives the request from, mostly in a multi-homed environment like what is described here [https://lists.zx2c4.com/pipermail/wireguard/2017-November/002016.html](https://lists.zx2c4.com/pipermail/wireguard/2017-November/002016.html)

`SO_REUSEADDR` and `SO_REUSEPORT` are used when creating connections. Multiple connections are created with all possible IP addresses defined in the `Caddyfile` which the service will use to reply. And these connections are all binded to the same local IP address and port opened when the request first gets forwarded to the destination. By doing this we can catch the reply from the service even it has been sent via a different interface (and a different source IP address).

## Build with Caddy

Follow the guide here [https://github.com/caddyserver/caddy/wiki/Plugging-in-Plugins-Yourself](https://github.com/caddyserver/caddy/wiki/Plugging-in-Plugins-Yourself)

The line to be added to `run.go` in the import section is

```go
_ "github.com/def1oyd/udpproxy"
```

## Example

Suppose

- The service recives requests at `10.0.0.254:12345`
- `10.0.0.254` is a VIP, floating between `10.0.0.252` and `10.0.0.253`
- The service is not able to use the VIP to send reply since it lacks capability of interface binding
- The reply may come from `10.0.0.252` or `10.0.0.253`

The `Caddyfile` will look like

```
proxy :12345 10.0.0.254:12345 {
    reply-addr-alias 10.0.0.253:12345 10.0.0.252:12345
}
```

Then start caddy with the following command

```bash
$ caddy -type=udpproxy
```

## References

1. [https://lists.zx2c4.com/pipermail/wireguard/2017-November/002016.html](https://lists.zx2c4.com/pipermail/wireguard/2017-November/002016.html)
2. [https://utcc.utoronto.ca/~cks/space/blog/linux/WireGuardEarlyNotes](https://utcc.utoronto.ca/~cks/space/blog/linux/WireGuardEarlyNotes)
3. [https://blog.kechako.com/entry/create-udp-conn-each-client](https://blog.kechako.com/entry/create-udp-conn-each-client)
