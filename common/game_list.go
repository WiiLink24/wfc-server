package common

import (
	"encoding/csv"
	"os"
	"strconv"
	"strings"
	"sync"
)

type GameInfo struct {
	GameID           int
	Name             string
	SecretKey        string
	GameStatsVersion int
	GameStatsKey     string
	Description      string
}

var (
	gameList           []GameInfo
	readGameList       = false
	gameListIDLookup   = map[int]int{}
	gameListNameLookup = map[string]int{}
	mutex              = sync.RWMutex{}
)

func GetGameInfoByID(gameId int) *GameInfo {
	ReadGameList()

	mutex.Lock()
	defer mutex.Unlock()

	if index, ok := gameListIDLookup[gameId]; ok && index < len(gameList) {
		return &gameList[index]
	}

	return nil
}

func GetGameInfoByName(name string) *GameInfo {
	ReadGameList()

	mutex.Lock()
	defer mutex.Unlock()

	if index, ok := gameListNameLookup[name]; ok && index < len(gameList) {
		return &gameList[index]
	}

	return nil
}

func GetGameID(name string) int {
	info := GetGameInfoByName(name)
	if info != nil {
		return info.GameID
	}

	return -1
}

func GetGameIDOrPanic(name string) int {
	id := GetGameID(name)
	if id == -1 {
		panic("Game not found: " + name)
	}

	return id
}

func ReadGameList() {
	mutex.Lock()
	defer mutex.Unlock()

	if readGameList {
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

		gameStatsVer := -1

		if entry[4] != "" {
			gameStatsVer, err = strconv.Atoi(entry[4])
			if err != nil {
				panic(err)
			}
		}

		gameList = append(gameList, GameInfo{
			GameID:           gameId,
			Name:             entry[1],
			SecretKey:        entry[3],
			GameStatsVersion: gameStatsVer,
			GameStatsKey:     entry[5],
			Description:      entry[0],
		})

		// Create lookup tables
		if gameId != -1 {
			gameListIDLookup[gameId] = index
		}
		gameListNameLookup[entry[1]] = index
	}

	readGameList = true
}

func GetExpectedUnitCode(gameName string) byte {
	if strings.HasSuffix(gameName, "wii") || strings.HasSuffix(gameName, "wiiam") {
		return 1
	}

	if gameName == "sneezieswiiw" || gameName == "wormswiiware" || gameName == "wormswiiwaream" {
		return 1
	}

	// Games with weird other regions
	if gameName == "jockracerna" || gameName == "jockracereu" || gameName == "sengo3wiijp" {
		return 1
	}

	// Cross-platform games
	if gameName == "mahjongkcds" || gameName == "puyopuyo7ds" || gameName == "puyopuyo20ds" {
		return 0xff
	}

	return 0
}

func DoesGameNeedExploit(gameName string) bool {
	// Exploit is only implemented for Mario Kart Wii and Mario Kart DS currently
	return gameName == "mariokartwii" || gameName == "mariokartds"
}
