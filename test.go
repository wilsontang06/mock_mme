package main

import (
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

var lock sync.Mutex

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
	// these values would probably make more sense in a struct instead of 3 maps
	sentTimes := make(map[int]time.Time)
	receivedTimes := make(map[int]time.Duration)
	sidToImsi := make(map[int]string)
	sidToRemoteAddr := make(map[int]net.Addr)

	sent := make(chan int)
	sentErr := make(chan struct{})
	for i := 0; i < numRequests; i, currIMSIIndex = i+1, currIMSIIndex+1 {
		// loops through imsi indices with numRequests
		if currIMSIIndex == len(imsis) {
			currIMSIIndex = 0
		}

		// assign random values to sids for each request
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

		go testFunc(randomVals, imsis[currIMSIIndex], sent, sentErr)
		// go sendULR(conns[0], cfgs[0], ueIMSIs[0], randomVal, sent, sentErr)
	}

	startTime = time.Now()

Wait:
	// the first response that comes back isn't necessarily the first one that was sent
	for recCount < numRequests*numRequestsPerTest {
		var r int
		select {
		case r = <-sent:
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
			if r.result == 0 {
				successes++
			} else {
				failures++
			}
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
