package udpproxy

import (
	// plug in the server
	_ "github.com/fltd/udpproxy/server"
	// plug in the standard directives
	_ "github.com/fltd/udpproxy/directive"
)
