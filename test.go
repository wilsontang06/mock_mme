package main

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

var lock sync.Mutex

func runTest(testFunc func(int, chan int, chan struct{}), numRequests int) (int, int) {
	successes := 0
	failures := 0

	sentCount := 0
	recCount := 0

	sentIds := make([]int, numRequests)
	sentTimes := make(map[int]time.Time)
	receivedTimes := make(map[int]time.Duration)

	sent := make(chan int)
	sentErr := make(chan struct{})
	for i := 0; i < numRequests; i++ {
		// time delay?
		randomVal := int(rand.Uint32())
		_, ok := sentTimes[randomVal]
		for ok {
			randomVal = int(rand.Uint32())
			_, ok = sentTimes[randomVal]
		}
		go testFunc(randomVal, sent, sentErr)
		// go sendULR(conns[0], cfgs[0], ueIMSIs[0], randomVal, sent, sentErr)
	}

Wait:
	// the first response that comes back isn't necessarily the first one that was sent
	for recCount < numRequests {
		var r int
		select {
		case r = <-sent:
			currTime := time.Now()
			lock.Lock()
			sentTimes[r] = currTime
			sentIds[sentCount] = r
			sentCount++
			lock.Unlock()
			if sentCount == numRequests {
				log.Printf("sent all requests for 0-th test")
			}
			// log.Printf("sent %d %v\n", sentCount, sentTimes[r])
		case r := <-received:
			currTime := time.Now()
			lock.Lock()
			if r.sid == 0 {
				successes++
			} else {
				failures++
			}
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

	return successes, failures
}
