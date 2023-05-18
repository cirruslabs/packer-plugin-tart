package tartcmd

import (
	"bytes"
	"sync"
)

type lockedBuffer struct {
	buffer bytes.Buffer
	mutex  sync.Mutex
}

func (lb *lockedBuffer) Write(p []byte) (n int, err error) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	return lb.buffer.Write(p)
}

func (lb *lockedBuffer) String() string {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	return lb.buffer.String()
}
