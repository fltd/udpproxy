package directive

import (
	"github.com/def1oyd/udpproxy/server"
	"github.com/caddyserver/caddy"
)

func init() {
	caddy.RegisterPlugin("reply-addr-alias", caddy.Plugin{
		ServerType: "udpproxy",
		Action:     setupReplyAddrAliases,
	})
}

func setupReplyAddrAliases(c *caddy.Controller) error {
	config := server.GetConfig(c)
	if c.Key != "proxy" {
		return nil
	}
	for c.Next() {
		for c.NextArg() {
			config.ReplyAddrAliases = append(config.ReplyAddrAliases, c.Val())
		}
	}
	return nil
}
