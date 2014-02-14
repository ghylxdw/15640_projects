// Epoch timer which sends a signal to notify event handelr every $epochMillis$ milliseconds
// @author: Chun Chen

package lsp

import (
	"time"
)

func epochTimer(requestc chan *request, closeSignal chan struct{}, epochMillis int) {
	for {
		select {
		case <-time.After(time.Millisecond * time.Duration(epochMillis)):
			req := &request{epochtimer, nil, make(chan *retType)}
			requestc <- req
		case <-closeSignal:
			// Shutdown the goroutine.
			return
		}
	}
}
