package main

import (
	"log"
	"strconv"

	"github.com/fiorix/go-diameter/diam"
	"github.com/fiorix/go-diameter/diam/avp"
	"github.com/fiorix/go-diameter/diam/datatype"
	"github.com/fiorix/go-diameter/diam/dict"
	"github.com/fiorix/go-diameter/diam/sm"
	"github.com/fiorix/go-diameter/diam/sm/smpeer"
)

// Create & send Update-Location Request
// sent back the sid through the sent channel
func sendULR(c diam.Conn, cfg *sm.Settings, imsi *string, randomVal int, sent chan int, sentErr chan struct{}) {
	meta, ok := smpeer.FromContext(c.Context())
	if !ok {
		sentErr <- struct{}{}
	}
	sid := "session;" + strconv.Itoa(randomVal)
	m := diam.NewRequest(diam.UpdateLocation, diam.TGPP_S6A_APP_ID, dict.Default)
	m.NewAVP(avp.SessionID, avp.Mbit, 0, datatype.UTF8String(sid))
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, cfg.OriginHost)
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, cfg.OriginRealm)
	m.NewAVP(avp.DestinationRealm, avp.Mbit, 0, meta.OriginRealm)
	m.NewAVP(avp.DestinationHost, avp.Mbit, 0, meta.OriginHost)
	m.NewAVP(avp.UserName, avp.Mbit, 0, datatype.UTF8String(*imsi))
	m.NewAVP(avp.AuthSessionState, avp.Mbit, 0, datatype.Enumerated(0))
	m.NewAVP(avp.RATType, avp.Mbit, uint32(*vendorID), datatype.Enumerated(1004))
	m.NewAVP(avp.ULRFlags, avp.Vbit|avp.Mbit, uint32(*vendorID), datatype.Unsigned32(ULR_FLAGS))
	m.NewAVP(avp.VisitedPLMNID, avp.Vbit|avp.Mbit, uint32(*vendorID), datatype.OctetString(*plmnID))
	// log.Printf("\nSending ULR to %s\n%s\n", c.RemoteAddr(), m)
	_, err := m.WriteTo(c)
	if err != nil {
		sentErr <- struct{}{}
	} else {
		sent <- randomVal
	}
}

// Handle ULA
// send back the result through the ReceivedResult channel
func handleUpdateLocationAnswer(received chan ReceivedResult) diam.HandlerFunc {
	return func(c diam.Conn, m *diam.Message) {
		// log.Printf("Received Update-Location Answer from %s\n%s\n", c.RemoteAddr(), m)
		var ula ULA
		err := m.Unmarshal(&ula)
		if err != nil {
			log.Printf("ULA Unmarshal failed: %s", err)
			received <- ReceivedResult{0, -2, c.RemoteAddr()}
		} else {
			sid, _ := strconv.Atoi(ula.SessionID[8:])
			if validateULAResponse(ula) == 1 {
				received <- ReceivedResult{sid, 0, c.RemoteAddr()}
			} else {
				received <- ReceivedResult{sid, -1, c.RemoteAddr()}
			}
			// log.Printf("Unmarshaled UL Answer:\n%#+v\n", ula)
			// log.Printf("ULA result code: 0x%x\n", ula.ResultCode)
		}
	}
}

func validateULAResponse(ula ULA) int {
	if ula.ResultCode != 0x7d1 {
		return 0
	}
	return 1
}

const ULR_FLAGS = 1<<1 | 1<<5
