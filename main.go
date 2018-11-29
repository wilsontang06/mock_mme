// Copyright 2013-2018 go-diameter authors.  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Diameter S6A client example.
package main

import (
	"flag"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/fiorix/go-diameter/diam"
	"github.com/fiorix/go-diameter/diam/avp"
	"github.com/fiorix/go-diameter/diam/datatype"
	"github.com/fiorix/go-diameter/diam/dict"
	"github.com/fiorix/go-diameter/diam/sm"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var (
	realm           = flag.String("diam_realm", "OpenAir5G.Alliance", "diameter identity realm")
	networkType     = flag.String("network_type", "tcp", "protocol type tcp/sctp/tcp4/tcp6/sctp4/sctp6")
	retries         = flag.Uint("retries", 3, "Maximum number of retransmits")
	watchdog        = flag.Uint("watchdog", 10, "Diameter watchdog interval in seconds. 0 to disable watchdog.")
	vendorID        = flag.Uint("vendor", 10415, "Vendor ID")
	appID           = flag.Uint("app", 16777251, "AuthApplicationID")
	plmnID          = flag.String("plmnid", "\x00\xF1\x10", "Client (UE) PLMN ID")
	vectors         = flag.Uint("vectors", 3, "Number Of Requested Auth Vectors")
	completionSleep = flag.Uint("sleep", 10, "After Completion Sleep Time (seconds)")

	addrs = [2]*string{
		flag.String("addr1", "127.0.0.1:3868", "address in form of ip:port to connect to"),
		flag.String("addr2", "127.0.0.1:3869", "address in form of ip:port to connect to"),
	}

	// oai_hss only supports mme.OpenAir5G.Alliance
	hosts = [2]*string{
		flag.String("diam_host1", "mme.OpenAir5G.Alliance", "diameter identity host1"),
		flag.String("diam_host2", "mme.OpenAir5G.Alliance", "diameter identity host2"),
	}

	ueIMSIs = [12]*string{
		flag.String("imsi1", "001010123456789", "Client (UE) IMSI 1"),
		flag.String("imsi2", "208920100001100", "Client (UE) IMSI 2"),
		flag.String("imsi3", "208920100001101", "Client (UE) IMSI 3"),
		flag.String("imsi4", "208920100001102", "Client (UE) IMSI 4"),
		flag.String("imsi5", "208920100001103", "Client (UE) IMSI 5"),
		flag.String("imsi6", "208920100001104", "Client (UE) IMSI 6"),
		flag.String("imsi7", "208920100001105", "Client (UE) IMSI 7"),
		flag.String("imsi8", "208920100001106", "Client (UE) IMSI 8"),
		flag.String("imsi9", "208920100001107", "Client (UE) IMSI 9"),
		flag.String("imsi10", "208920100001108", "Client (UE) IMSI 10"),
		flag.String("imsi11", "208920100001109", "Client (UE) IMSI 11"),
		flag.String("imsi12", "208920100001111", "Client (UE) IMSI 12"),
	}

	badUeIMSIs = [12]*string{
		flag.String("badimsi1", "123456789123456", "Bad Client (UE) IMSI 1"),
		flag.String("badimsi2", "123456789123457", "Bad Client (UE) IMSI 2"),
		flag.String("badimsi3", "123456789123458", "Bad Client (UE) IMSI 3"),
		flag.String("badimsi4", "123456789123459", "Bad Client (UE) IMSI 4"),
		flag.String("badimsi5", "123456789123460", "Bad Client (UE) IMSI 5"),
		flag.String("badimsi6", "123456789123461", "Bad Client (UE) IMSI 6"),
		flag.String("badimsi7", "123456789123462", "Bad Client (UE) IMSI 7"),
		flag.String("badimsi8", "123456789123463", "Bad Client (UE) IMSI 8"),
		flag.String("badimsi9", "123456789123464", "Bad Client (UE) IMSI 9"),
		flag.String("badimsi10", "123456789123465", "Bad Client (UE) IMSI 10"),
		flag.String("badimsi11", "123456789123466", "Bad Client (UE) IMSI 11"),
		flag.String("badimsi12", "123456789123467", "Bad Client (UE) IMSI 12"),
	}

	mixUeIMSIs = [12]*string{
		flag.String("miximsi1", "001010123456789", "Client (UE) IMSI 1"),
		flag.String("miximsi7", "123456789123456", "Bad Client (UE) IMSI 1"),
		flag.String("miximsi2", "208920100001100", "Client (UE) IMSI 2"),
		flag.String("miximsi8", "123456789123457", "Bad Client (UE) IMSI 2"),
		flag.String("miximsi3", "208920100001101", "Client (UE) IMSI 3"),
		flag.String("miximsi9", "123456789123458", "Bad Client (UE) IMSI 3"),
		flag.String("miximsi4", "208920100001102", "Client (UE) IMSI 4"),
		flag.String("miximsi10", "123456789123459", "Bad Client (UE) IMSI 4"),
		flag.String("miximsi5", "208920100001103", "Client (UE) IMSI 5"),
		flag.String("miximsi11", "123456789123460", "Bad Client (UE) IMSI 5"),
		flag.String("miximsi6", "208920100001104", "Client (UE) IMSI 6"),
		flag.String("miximsi12", "123456789123461", "Bad Client (UE) IMSI 6"),
	}
)

var received = make(chan ReceivedResult)

func main() {

	flag.Parse()

	var i int
	// var j int
	var cfgs [2]*sm.Settings
	var conns [2]diam.Conn

	var lock sync.Mutex

	// connect the mme's to the hss's
	// 2 hss' is the current pattern
	for i = 0; i < 2; i++ {
		cfgs[i] = &sm.Settings{
			OriginHost:       datatype.DiameterIdentity(*hosts[i]),
			OriginRealm:      datatype.DiameterIdentity(*realm),
			OriginStateID:    datatype.Unsigned32(time.Now().Unix()),
			VendorID:         datatype.Unsigned32(*vendorID),
			ProductName:      "go-diameter-s6a",
			FirmwareRevision: 1,
			HostIPAddresses: []datatype.Address{
				datatype.Address(net.ParseIP("127.0.0.1")),
			},
		}

		// Create the state machine (it's a diam.ServeMux) and client.
		mux := sm.New(cfgs[i])

		cli := &sm.Client{
			Dict:               dict.Default,
			Handler:            mux,
			MaxRetransmits:     *retries,
			RetransmitInterval: time.Second,
			EnableWatchdog:     *watchdog != 0,
			WatchdogInterval:   time.Duration(*watchdog) * time.Second,
			SupportedVendorID: []*diam.AVP{
				diam.NewAVP(avp.SupportedVendorID, avp.Mbit, 0, datatype.Unsigned32(*vendorID)),
			},
			VendorSpecificApplicationID: []*diam.AVP{
				diam.NewAVP(avp.VendorSpecificApplicationID, avp.Mbit, 0, &diam.GroupedAVP{
					AVP: []*diam.AVP{
						diam.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(*appID)),
						diam.NewAVP(avp.VendorID, avp.Mbit, 0, datatype.Unsigned32(*vendorID)),
					},
				}),
			},
		}

		// Set message handlers.

		mux.HandleIdx(
			diam.CommandIndex{AppID: diam.TGPP_S6A_APP_ID, Code: diam.UpdateLocation, Request: false},
			handleUpdateLocationAnswer(received))

		// Print error reports.
		go printErrors(mux.ErrorReports())

		conn, err := cli.DialNetwork(*networkType, *addrs[i], handleCEAClient)
		if err != nil {
			log.Printf("failed to connect mme %d\n", i)
			// log.Fatal(err)
		}
		log.Printf("connected to %s\n", *addrs[i])
		conns[i] = conn
	}

	requests := 100
	recCount := 0
	sentCount := 0

	// load test hss 1 with 1 imsi a lot of times
	// seems like ~9000 (8500-9500) is the limit i got to with a 1/1000th second gap between each call
	// hits "readBody Error: unexpected EOF, 308 bytes read" inconsistently

	// HSS sents back packets in batches when it sees a lot of requests. But when the
	// requests hit around 30-35, it will send one batch and then sent all the next responses
	// one at a time and performance suffers
	// though when i test it in the thousands, it will batch it again

	sentIds := make([]int, requests)
	sentTimes := make(map[int]time.Time)
	receivedTimes := make(map[int]time.Duration)

	sent := make(chan int)
	sentErr := make(chan struct{})
	for i = 0; i < requests; i++ {
		// time delay?
		randomVal := int(rand.Uint32())
		_, ok := sentTimes[randomVal]
		for ok {
			randomVal = int(rand.Uint32())
			_, ok = sentTimes[randomVal]
		}
		go sendULR(conns[0], cfgs[0], ueIMSIs[0], randomVal, sent, sentErr)
	}

Wait:
	// the first response that comes back isn't necessarily the first one that was sent
	for recCount < requests {
		var r int
		select {
		case r = <-sent:
			currTime := time.Now()
			lock.Lock()
			sentTimes[r] = currTime
			sentIds[sentCount] = r
			sentCount++
			lock.Unlock()
			if sentCount == requests {
				log.Printf("sent all requests for 0-th test")
			}
			// log.Printf("sent %d %v\n", sentCount, sentTimes[r])
		case r := <-received:
			currTime := time.Now()
			lock.Lock()
			receivedTimes[r.sid] = currTime.Sub(sentTimes[r.sid])
			recCount++
			lock.Unlock()
			// log.Printf("received %d %v\n", r, receivedTimes[r])
		case <-sentErr:
			sentCount++
			log.Printf("sending %d request failed", sentCount+1)
			break Wait
		// wait 20 seconds for responses to come back
		case <-time.After(20 * time.Second):
			log.Printf("timed out waiting for ULR")
			break Wait
		}
	}

	for i = 0; i < len(sentIds); i++ {
		dur, ok := receivedTimes[sentIds[i]]
		if ok {
			log.Printf("received %d of sid %d in %v\n", i+1, sentIds[i], dur)
		} else {
			log.Printf("failed to receive %d with sid %d\n", i+1, sentIds[i])
		}
	}

	log.Printf("0.Load Testing 1 HSS Results:")
	/*
		log.Printf("Successes: %d\n", successes)
		log.Printf("Failures: %d\n", failures)
		log.Printf("Missing: %d\n", requests-(successes+failures))
	*/

	// load test hss 1 and 2 with 12 imsi's 10 times with all success
	/*
			successes = 0
			failures = 0

			requests = 10
			count = 0
			for i = 0; i < requests; i++ {
				for j = 0; j < len(ueIMSIs); j++ {
					err := sendULR(conns[0], cfgs[0], ueIMSIs[j])
					if err != nil {
						log.Fatal(err)
					}
					// log.Printf("sent %d\n", i+1)

					err2 := sendULR(conns[1], cfgs[0], ueIMSIs[j])
					if err2 != nil {
						log.Fatal(err2)
					}
				}
			}

		Wait1:
			for count < requests*2*12 {
				select {
				case <-done:
					count++
					// log.Printf("received %d\n", count)
				case <-time.After(10 * time.Second):
					log.Printf("1.'all successes' test failed to receive all responses")
					break Wait1
				}
			}

			log.Printf("1.'All Successes' Test Results:")
			log.Printf("Successes: %d\n", successes)
			log.Printf("Failures: %d\n", failures)
	*/

	// load test hss 1 and 2 with 12 imsi's 10 times with all fails
	/*
			successes = 0
			failures = 0

			requests = 10
			count = 0
			for i = 0; i < requests; i++ {
				for j = 0; j < len(ueIMSIs); j++ {
					err := sendULR(conns[0], cfgs[0], badUeIMSIs[j])
					if err != nil {
						log.Fatal(err)
					}
					// log.Printf("sent %d\n", i+1)

					err2 := sendULR(conns[1], cfgs[0], badUeIMSIs[j])
					if err2 != nil {
						log.Fatal(err2)
					}
				}
			}

		Wait2:
			for count < requests*2*12 {
				select {
				case <-done:
					count++
					// log.Printf("received %d\n", count)
				case <-time.After(10 * time.Second):
					log.Printf("2.'all failures' test failed to receive all responses")
					break Wait2
				}
			}

			log.Printf("2.'All Faiures' Test Results:")
			log.Printf("Successes: %d\n", successes)
			log.Printf("Failures: %d\n", failures)
	*/

	// load test hss 1 and 2 with 12 imsi's 10 times with half success, half fail
	/*
			successes = 0
			failures = 0

			requests = 10
			count = 0
			for i = 0; i < requests; i++ {
				for j = 0; j < len(ueIMSIs); j++ {
					err := sendULR(conns[0], cfgs[0], mixUeIMSIs[j])
					if err != nil {
						log.Fatal(err)
					}
					// log.Printf("sent %d\n", i+1)

					err2 := sendULR(conns[1], cfgs[0], mixUeIMSIs[j])
					if err2 != nil {
						log.Fatal(err2)
					}
				}
			}

		Wait3:
			for count < requests*2*12 {
				select {
				case <-done:
					count++
					// log.Printf("received %d\n", count)
				case <-time.After(10 * time.Second):
					log.Printf("3.'half successes/half failures' test failed to receive all responses")
					break Wait3
				}
			}

			log.Printf("3.'Half Success/Half Failure' Test Results:")
			log.Printf("Successes: %d\n", successes)
			log.Printf("Failures: %d\n", failures)
	*/
}

func printErrors(ec <-chan *diam.ErrorReport) {
	for err := range ec {
		log.Println(err)
	}
}

/*
func loadTestHSS(numRequests int, sent chan int, sentErr chan struct{}) {
	sendULR(conns[0], cfgs[0], ueIMSIs[0], randomVal, sent, sentErr)
}
*/
