package udpproxy

import (
	// plug in the server
	_ "github.com/def1oyd/udpproxy/server"
	// plug in the standard directives
	_ "github.com/def1oyd/udpproxy/directive"
)
