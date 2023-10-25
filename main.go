package main

import (
	"sync"
	"wwfc/gpcm"
	"wwfc/gpsp"
	"wwfc/matchmaking"
	"wwfc/nas"
	"wwfc/qr2"
)

func main() {
	wg := &sync.WaitGroup{}
	actions := []func(){nas.StartServer, gpcm.StartServer, qr2.StartServer, gpsp.StartServer, matchmaking.StartServer}
	wg.Add(5)
	for _, action := range actions {
		go func(ac func()) {
			defer wg.Done()
			ac()
		}(action)
	}

	wg.Wait()
}
