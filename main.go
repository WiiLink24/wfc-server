package main

import (
	"sync"
	"wwfc/gcsp"
	"wwfc/gpcm"
	"wwfc/master"
	"wwfc/matchmaking"
	"wwfc/nas"
)

func main() {
	wg := &sync.WaitGroup{}
	actions := []func(){nas.StartServer, gpcm.StartServer, master.StartServer, gcsp.StartServer, matchmaking.StartServer}
	wg.Add(4)
	for _, action := range actions {
		go func(ac func()) {
			defer wg.Done()
			ac()
		}(action)
	}

	wg.Wait()
}
