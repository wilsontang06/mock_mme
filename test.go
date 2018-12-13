package main

import (
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

var lock sync.Mutex

// the main fun of this repo!
// runs the given testFunc and returns success/failure count
// purpose is to create any testFunc (load test, test multiple hss) and
// use this method to run the test
// return: success count, failure count, duration of test
// parameters:
// - testFunc: a function with parameters[]int of sids, imsi string,
// 			   sent channel, sentErr channel
// - imsis: string array of imsis to test
// - numRequests: number of times we will call testFunc
// - numRequestsPerTest: number of requests in each testFunc
// - printReceived: print stats of each individual received answer
func runTest(testFunc func([]int, *string, chan int, chan struct{}),
	imsis []*string, numRequests int, numRequestsPerTest int,
	printReceived bool) (int, int, time.Duration) {
	var startTime, endTime time.Time

	successes := 0
	failures := 0
	sentCount := 0
	recCount := 0
	currIMSIIndex := 0

	sentIds := make([]int, numRequests*numRequestsPerTest)
	// these maps would probably make more sense in a struct instead of 4 individual maps
	// sid to time the request was sent
	sentTimes := make(map[int]time.Time)
	// sid to time the request was received
	receivedTimes := make(map[int]time.Duration)
	// sid to imsi that was sent in request
	sidToImsi := make(map[int]string)
	// sid to remote address request was sent to/received from
	sidToRemoteAddr := make(map[int]net.Addr)

	sent := make(chan int)
	sentErr := make(chan struct{})

	startTime = time.Now()

	for i := 0; i < numRequests; i, currIMSIIndex = i+1, currIMSIIndex+1 {
		// loops through imsi indices with numRequests
		if currIMSIIndex == len(imsis) {
			currIMSIIndex = 0
		}

		// assign random values to sids for each request for testFunc
		randomVals := make([]int, numRequestsPerTest)
		for j := 0; j < numRequestsPerTest; j++ {
			randomVals[j] = int(rand.Uint32())
			_, ok := sentTimes[randomVals[j]]
			for ok {
				randomVals[j] = int(rand.Uint32())
				_, ok = sentTimes[randomVals[j]]
			}
			sidToImsi[randomVals[j]] = *imsis[currIMSIIndex]
		}

		// start a goroutine for the testFunc
		go testFunc(randomVals, imsis[currIMSIIndex], sent, sentErr)
	}

Wait:
	// wait for all of the requests to be sent and
	// all of the requests to be answered
	for recCount < numRequests*numRequestsPerTest {
		var r int
		select {
		case r = <-sent:
			// record sent time of request
			currTime := time.Now()
			lock.Lock()
			sentTimes[r] = currTime
			sentIds[sentCount] = r
			sentCount++
			lock.Unlock()
			// log.Printf("sent %d %v\n", sentCount, sentTimes[r])
		case r := <-received:
			currTime := time.Now()
			lock.Lock()
			// record result
			if r.result == 0 {
				successes++
			} else {
				failures++
			}
			// record how long the request took to come back
			receivedTimes[r.sid] = currTime.Sub(sentTimes[r.sid])
			sidToRemoteAddr[r.sid] = r.remoteAddr
			recCount++
			lock.Unlock()
			// log.Printf("received %d from %s\n", r.sid, r.remoteAddr)
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

	endTime = time.Now()

	// if we want to log all stats of each received answer
	// log the sid, imsi, remote address, and duration of request
	if printReceived {
		for i := 0; i < len(sentIds); i++ {
			dur, ok := receivedTimes[sentIds[i]]
			if ok {
				log.Printf("received\t%d of sid\t%d\tand imsi\t%s\tfrom\t%s\tin\t%v\n",
					i+1, sentIds[i], sidToImsi[sentIds[i]], sidToRemoteAddr[sentIds[i]], dur)
			} else {
				log.Printf("failed to receive %d with sid %d\n", i+1, sentIds[i])
			}
		}
	}

	return successes, failures, endTime.Sub(startTime)
}
