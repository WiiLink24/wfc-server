package main

import (
	"sync"
	"wwfc/gpcm"
	"wwfc/gpsp"
	"wwfc/nas"
	"wwfc/qr2"
	"wwfc/sake"
	"wwfc/serverbrowser"
)

func main() {
	wg := &sync.WaitGroup{}
	actions := []func(){nas.StartServer, gpcm.StartServer, qr2.StartServer, gpsp.StartServer, serverbrowser.StartServer, sake.StartServer}
	wg.Add(5)
	for _, action := range actions {
		go func(ac func()) {
			defer wg.Done()
			ac()
		}(action)
	}

	wg.Wait()
}
