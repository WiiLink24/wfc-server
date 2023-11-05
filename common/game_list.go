package common

import (
	"encoding/csv"
	"os"
	"strconv"
	"sync"
)

type GameInfo struct {
	GameID      int
	Name        string
	SecretKey   string
	Description string
}

var (
	gameListReady      = false
	gameList           = []GameInfo{}
	gameListIDLookup   = map[int]int{}
	gameListNameLookup = map[string]int{}
	mutex              = sync.RWMutex{}
)

func GetGameList() []GameInfo {
	if !gameListReady {
		ReadGameList()
	}

	return gameList
}

func GetGameInfoByID(gameId int) (GameInfo, bool) {
	if !gameListReady {
		ReadGameList()
	}

	if index, ok := gameListIDLookup[gameId]; ok && index < len(gameList) {
		return gameList[index], true
	}

	return GameInfo{}, false
}

func GetGameInfoByName(name string) (GameInfo, bool) {
	if !gameListReady {
		ReadGameList()
	}

	if index, ok := gameListNameLookup[name]; ok && index < len(gameList) {
		return gameList[index], true
	}

	return GameInfo{}, false
}

func ReadGameList() {
	mutex.Lock()
	defer mutex.Unlock()

	if gameListReady {
		return
	}

	file, err := os.Open("game_list.tsv")
	if err != nil {
		panic(err)
	}

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	csvList, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	gameList = []GameInfo{}
	gameListIDLookup = map[int]int{}
	gameListNameLookup = map[string]int{}

	for index, entry := range csvList {
		gameId := -1

		if entry[2] != "" {
			gameId, err = strconv.Atoi(entry[2])
			if err != nil {
				panic(err)
			}
		}

		gameList = append(gameList, GameInfo{
			GameID:      gameId,
			Name:        entry[1],
			SecretKey:   entry[3],
			Description: entry[0],
		})

		// Create lookup tables
		if gameId != -1 {
			gameListIDLookup[gameId] = index
		}
		gameListNameLookup[entry[1]] = index
	}

	gameListReady = true
}
