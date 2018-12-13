package main

import (
	"github.com/fiorix/go-diameter/diam"
	"github.com/fiorix/go-diameter/diam/sm"
	"github.com/fiorix/go-diameter/diam/sm/smparser"
	"github.com/fiorix/go-diameter/diam/sm/smpeer"
)

// handleCEA handles Capabilities-Exchange-Answer messages.
func handleCEAClient(sme *sm.StateMachine, errc chan error) diam.HandlerFunc {
	return func(c diam.Conn, m *diam.Message) {
		// log.Printf("Received Capabilities-Exchange-Answer from %s\n%s\n", c.RemoteAddr(), m)
		cea := new(smparser.CEA)
		if err := cea.Parse(m, smparser.Client); err != nil {
			errc <- err
			return
		}
		// log.Printf("CEA parsed:\n%#+v\n", cea)
		// log.Printf("CEA result code: %#+v\n", cea.ResultCode)
		meta := smpeer.FromCEA(cea)
		c.SetContext(smpeer.NewContext(c.Context(), meta))
		// Notify about peer passing the handshake.
		select {
		// case sme.HandshakeNotify <- c:
		default:
		}
		// Done receiving and validating this CEA.
		close(errc)
	}
}
