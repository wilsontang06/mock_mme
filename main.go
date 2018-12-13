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
	"strconv"
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
	// configuration variables for hss connection
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

	// test arrays of imsi's
	ueIMSIs = []*string{
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

	badUeIMSIs = []*string{
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

	mixUeIMSIs = []*string{
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

	loadTestRequestNums = []int{10, 100, 500, 1000, 3000, 6000}
)

// received channel to put into ULR handler
var received = make(chan ReceivedResult)

func main() {
	var i int
	var cfgs [2]*sm.Settings
	var conns [2]diam.Conn

	var successes, failures int
	var duration time.Duration

	log.Printf("Begin Connection...\n")

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

	log.Printf("Connected\n")
	log.Printf("Begin Tests...\n")

	// run load tests with 1 single imsi
	for i := 0; i < len(loadTestRequestNums); i++ {
		successes, failures, duration = runTest(
			loadTest(conns[0], cfgs[0]), []*string{ueIMSIs[0]}, loadTestRequestNums[i], 1, false)

		printResults(i,
			"Load Testing 1 HSS Results with "+strconv.Itoa(loadTestRequestNums[i])+" requests",
			successes, failures, loadTestRequestNums[i], duration)
	}

	// 2 hss test - success
	successes, failures, duration = runTest(
		twoHSSTest(conns[0], conns[1], cfgs[0], cfgs[1]),
		ueIMSIs, len(ueIMSIs), 2, false)
	printResults(1, "2 HSS Testing - success cases",
		successes, failures, len(ueIMSIs)*2, duration)

	// 2 hss test - failure
	successes, failures, duration = runTest(
		twoHSSTest(conns[0], conns[1], cfgs[0], cfgs[1]),
		badUeIMSIs, len(badUeIMSIs), 2, false)
	printResults(2, "2 HSS Testing - failure cases",
		successes, failures, len(badUeIMSIs)*2, duration)

	// 2 hss test - mixture
	successes, failures, duration = runTest(
		twoHSSTest(conns[0], conns[1], cfgs[0], cfgs[1]),
		mixUeIMSIs, len(mixUeIMSIs), 2, false)
	printResults(3, "2 HSS Testing - mixture cases",
		successes, failures, len(mixUeIMSIs)*2, duration)

	log.Printf("Testing Completed. Goodbye :)")
}

func printErrors(ec <-chan *diam.ErrorReport) {
	for err := range ec {
		log.Println(err)
	}
}

// loadTest() is an example of a testFunc that will be passed into the runTest method.
// return: a function that takes in an []int of sids, an imsi, and two sent channels (one good, one error)
// parameters: the connection and cfg of the hss
// the use case is to send one imsi to multiple hss', but since this is a load test, there is
// only one request to an hss
func loadTest(connection diam.Conn, cfgs *sm.Settings) func([]int, *string, chan int, chan struct{}) {
	return func(sids []int, imsi *string, sent chan int, sentErr chan struct{}) {
		sendULR(connection, cfgs, imsi, sids[0], sent, sentErr)
	}
}

// twoHSSTest() is an example of a testFunc that will be passed into the runTest method.
// return: a function that takes in an []int of sids, an imsi, and two sent channels (one good, one error)
// parameters: two hss connections and two hss cfgs
// sends a ULR to the first hss, wait 2 seconds, and then send a ULR with the same imsi to a 2nd HSS
// different sids so each request can be tracked in our tests
func twoHSSTest(hss1 diam.Conn, hss2 diam.Conn,
	cfg1 *sm.Settings, cfg2 *sm.Settings) func([]int, *string, chan int, chan struct{}) {
	return func(sids []int, imsi *string, sent chan int, sentErr chan struct{}) {
		sendULR(hss1, cfg1, imsi, sids[0], sent, sentErr)
		time.Sleep(2 * time.Second)
		sendULR(hss2, cfg2, imsi, sids[1], sent, sentErr)
	}
}

// print the results of the test
func printResults(index int, testName string, successes int, failures int,
	total int, duration time.Duration) {
	log.Printf("%d. %s:", index, testName)
	log.Printf("   Successes: %d\n", successes)
	log.Printf("   Failures: %d\n", failures)
	log.Printf("   Missing: %d\n", total-(successes+failures))
	log.Printf("   Finished in: %v\n", duration)
}
