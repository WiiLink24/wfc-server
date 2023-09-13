package main

import (
	"sync"
	"wwfc/gpcm"
	"wwfc/gpsp"
	"wwfc/nas"
)

func main() {
	wg := &sync.WaitGroup{}
	actions := []func(){nas.StartServer, gpcm.StartServer, gpsp.StartServer}
	wg.Add(3)
	for _, action := range actions {
		go func(ac func()) {
			defer wg.Done()
			ac()
		}(action)
	}

	wg.Wait()
}
