package server

import (
	"fmt"
	"strings"

	"github.com/caddyserver/caddy"
	"github.com/caddyserver/caddy/caddyfile"
)

const serverType = "udpproxy"

var directives = []string{"reply-addr-alias"}

func init() {
	caddy.RegisterServerType(serverType, caddy.ServerType{
		Directives: func() []string { return directives },
		DefaultInput: func() caddy.Input {
			return caddy.CaddyfileInput{
				ServerTypeName: serverType,
			}
		},
		NewContext: newCaddyContext,
	})
}

func newCaddyContext(inst *caddy.Instance) caddy.Context {
	return &caddyContext{instance: inst, keysToConfigs: make(map[string]*Config)}
}

// Config contains configuration details about a net server type
type Config struct {
	Parameters       []string
	Tokens           map[string][]string
	ReplyAddrAliases []string
}

type caddyContext struct {
	instance *caddy.Instance
	// keysToConfigs maps an address at the top of a
	// server block (a "key") to its Config. Not all
	// Configs will be represented here, only ones
	// that appeared in the Caddyfile.
	keysToConfigs map[string]*Config

	// configs is the master list of all site configs.
	configs []*Config
}

func (c *caddyContext) saveConfig(key string, config *Config) {
	c.configs = append(c.configs, config)
	c.keysToConfigs[key] = config
}

type configTokens map[string][]string

// InspectServerBlocks make sure that everything checks out before
// executing directives and otherwise prepares the directives to
// be parsed and executed.
func (c *caddyContext) InspectServerBlocks(sourceFile string, serverBlocks []caddyfile.ServerBlock) ([]caddyfile.ServerBlock, error) {
	sbTokens := make(map[string]configTokens)

	// Example:
	// proxy :12017 :22017 {
	//     reply-addr-alias 1.2.3.4:22017 5.6.7.8:22017
	// }
	// ServerBlock Keys will be proxy :12017 :22017 and Tokens will be reply-addr-alias

	// For each key in each server block, make a new config
	for _, sb := range serverBlocks {
		// build unique key from server block keys and join with '~' i.e echo~:12345
		key := ""
		for _, k := range sb.Keys {
			k = strings.ToLower(k)
			if key == "" {
				key = k
			} else {
				key += fmt.Sprintf("~%s", k)
			}
		}
		if _, dup := c.keysToConfigs[key]; dup {
			return serverBlocks, fmt.Errorf("duplicate key: %s", key)
		}

		tokens := make(map[string][]string)
		for k, v := range sb.Tokens {
			tokens[k] = []string{}
			for _, token := range v {
				tokens[k] = append(tokens[k], token.Text)
			}
		}
		sbTokens[key] = tokens
	}

	// build the actual Config from gathered data
	// key is the server block key joined by ~
	for k, v := range sbTokens {
		params := strings.Split(k, "~")
		listenType := params[0]
		params = params[1:]

		if len(params) == 0 {
			return serverBlocks, fmt.Errorf("invalid configuration: %s", k)
		}

		if listenType == "proxy" && len(params) < 2 {
			return serverBlocks, fmt.Errorf("invalid configuration: proxy server block expects a source and destination address")
		}

		// Save the config to our master list, and key it for lookups
		config := &Config{
			Parameters: params,
			Tokens:     v,
		}

		c.saveConfig(k, config)
	}

	return serverBlocks, nil
}

// MakeServers uses the newly-created configs to create and return a list of server instances.
func (c *caddyContext) MakeServers() ([]caddy.Server, error) {
	var servers []caddy.Server
	for _, config := range c.configs {
		s, err := NewProxyServer(config.Parameters[0], config.Parameters[1], config)
		if err != nil {
			return nil, err
		}
		servers = append(servers, s)
	}
	return servers, nil
}

// GetConfig gets the Config that corresponds to c.
// If none exist (should only happen in tests), then a
// new, empty one will be created.
func GetConfig(c *caddy.Controller) *Config {
	ctx := c.Context().(*caddyContext)
	key := strings.Join(c.ServerBlockKeys, "~")
	//only check for config if the value is proxy or echo
	//we need to do this because we specify the ports in the server block
	//and those values need to be ignored as they are also sent from caddy main process.
	if strings.Contains(key, "proxy") {
		if config, ok := ctx.keysToConfigs[key]; ok {
			return config
		}
	}
	return nil
}
