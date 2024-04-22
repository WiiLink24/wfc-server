package main

import (
	"sync"
	"wwfc/api"
	"wwfc/common"
	"wwfc/gamestats"
	"wwfc/gpcm"
	"wwfc/gpsp"
	"wwfc/logging"
	"wwfc/nas"
	"wwfc/natneg"
	"wwfc/qr2"
	"wwfc/sake"
	"wwfc/serverbrowser"
)

func main() {
	config := common.GetConfig()
	logging.SetLevel(*config.LogLevel)
	if err := logging.SetOutput(config.LogOutput); err != nil {
		logging.Error("MAIN", err)
	}

	wg := &sync.WaitGroup{}
	actions := []func(){nas.StartServer, gpcm.StartServer, qr2.StartServer, gpsp.StartServer, serverbrowser.StartServer, sake.StartServer, natneg.StartServer, api.StartServer, gamestats.StartServer}
	wg.Add(len(actions))
	for _, action := range actions {
		go func(ac func()) {
			defer wg.Done()
			ac()
		}(action)
	}

	wg.Wait()
}
