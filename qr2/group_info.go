package qr2

import (
	"sort"
	"strconv"
	"time"
	"wwfc/common"
)

type MiiInfo struct {
	MiiData string `json:"data"`
	MiiName string `json:"name"`
}

type PlayerInfo struct {
	Count      string `json:"count"`
	ProfileID  string `json:"pid"`
	InGameName string `json:"name"`
	ConnMap    string `json:"conn_map"`
	ConnFail   string `json:"conn_fail"`
	Suspend    string `json:"suspend"`

	// Mario Kart Wii-specific fields
	FriendCode string    `json:"fc,omitempty"`
	VersusELO  string    `json:"ev,omitempty"`
	BattleELO  string    `json:"eb,omitempty"`
	Mii        []MiiInfo `json:"mii,omitempty"`
}

type GroupInfo struct {
	GroupName   string    `json:"id"`
	GameName    string    `json:"game"`
	CreateTime  time.Time `json:"created"`
	MatchType   string    `json:"type"`
	Suspend     bool      `json:"suspend"`
	ServerIndex string    `json:"host,omitempty"`
	MKWRegion   string    `json:"rk,omitempty"`

	Players  map[string]PlayerInfo `json:"players"`
	RaceInfo *RaceInfo             `json:"race,omitempty"`

	PlayersRaw      map[string]map[string]string `json:"-"`
	SortedJoinIndex []string                     `json:"-"`
}

type RaceInfo struct {
	RaceNumber    int `json:"num"`
	CourseID      int `json:"course"`
	EngineClassID int `json:"cc"`
}

func getGroupsRaw(gameNames []string, groupNames []string) []GroupInfo {
	var groupsCopy []GroupInfo

	mutex.Lock()
	defer mutex.Unlock()

	for _, group := range groups {
		if len(gameNames) > 0 && !common.StringInSlice(group.GameName, gameNames) {
			continue
		}

		if len(groupNames) > 0 && !common.StringInSlice(group.GroupName, groupNames) {
			continue
		}

		groupInfo := GroupInfo{
			GroupName:       group.GroupName,
			GameName:        group.GameName,
			CreateTime:      group.CreateTime,
			MatchType:       "",
			Suspend:         true,
			ServerIndex:     "",
			MKWRegion:       "",
			Players:         map[string]PlayerInfo{},
			PlayersRaw:      map[string]map[string]string{},
			SortedJoinIndex: []string{},
		}

		if group.MatchType == "0" || group.MatchType == "1" {
			groupInfo.MatchType = "anybody"
		} else if group.MatchType == "2" || group.MatchType == "3" {
			groupInfo.MatchType = "private"
		} else {
			groupInfo.MatchType = "unknown"
		}

		if group.server != nil {
			groupInfo.ServerIndex = group.server.Data["+joinindex"]
		}

		if groupInfo.GameName == "mariokartwii" {
			groupInfo.MKWRegion = group.MKWRegion

			if group.MKWRaceNumber != 0 {
				groupInfo.RaceInfo = &RaceInfo{
					RaceNumber:    group.MKWRaceNumber,
					CourseID:      group.MKWCourseID,
					EngineClassID: group.MKWEngineClassID,
				}
			}
		}

		for session := range group.players {
			mapData := map[string]string{}
			for k, v := range session.Data {
				mapData[k] = v
			}

			if login := session.login; login != nil {
				mapData["+ingamesn"] = login.InGameName
			} else {
				mapData["+ingamesn"] = ""
			}

			groupInfo.PlayersRaw[mapData["+joinindex"]] = mapData

			if mapData["dwc_hoststate"] == "2" && mapData["dwc_suspend"] == "0" {
				groupInfo.Suspend = false
			}

			// Add the join index to the sorted list
			myJoinIndex, _ := strconv.Atoi(mapData["+joinindex"])
			added := false

			for i, joinIndex := range groupInfo.SortedJoinIndex {
				if joinIndex == mapData["+joinindex"] {
					added = true
					break
				}

				intJoinIndex, _ := strconv.Atoi(joinIndex)
				if intJoinIndex > myJoinIndex {
					groupInfo.SortedJoinIndex = append(groupInfo.SortedJoinIndex, "")
					copy(groupInfo.SortedJoinIndex[i+1:], groupInfo.SortedJoinIndex[i:])
					groupInfo.SortedJoinIndex[i] = mapData["+joinindex"]
					added = true
					break
				}
			}

			if !added {
				groupInfo.SortedJoinIndex = append(groupInfo.SortedJoinIndex, mapData["+joinindex"])
			}
		}

		groupsCopy = append(groupsCopy, groupInfo)
	}

	return groupsCopy
}

// GetGroups returns a copy of all online rooms
func GetGroups(gameNames []string, groupNames []string, sorted bool) []GroupInfo {
	groupsCopy := getGroupsRaw(gameNames, groupNames)

	for i, group := range groupsCopy {
		for joinIndex, rawPlayer := range group.PlayersRaw {
			playerInfo := PlayerInfo{
				Count:      rawPlayer["+localplayers"],
				ProfileID:  rawPlayer["dwc_pid"],
				InGameName: rawPlayer["+ingamesn"],
			}

			pid, err := strconv.ParseUint(rawPlayer["dwc_pid"], 10, 32)
			if err == nil {
				if fcGame := rawPlayer["+fcgameid"]; len(fcGame) == 4 {
					playerInfo.FriendCode = common.CalcFriendCodeString(uint32(pid), fcGame)
				}
			}

			if rawPlayer["gamename"] == "mariokartwii" {
				playerInfo.VersusELO = rawPlayer["ev"]
				playerInfo.BattleELO = rawPlayer["eb"]
			}

			for i := 0; i < 32; i++ {
				miiData := rawPlayer["+mii"+strconv.Itoa(i)]
				if miiData == "" {
					continue
				}

				playerInfo.Mii = append(playerInfo.Mii, MiiInfo{
					MiiData: miiData,
					MiiName: rawPlayer["+mii_name"+strconv.Itoa(i)],
				})
			}

			for _, newIndex := range group.SortedJoinIndex {
				if newIndex == joinIndex {
					continue
				}

				if rawPlayer["+conn_"+newIndex] == "" {
					playerInfo.ConnMap += "0"
					continue
				}

				playerInfo.ConnMap += rawPlayer["+conn_"+newIndex]
			}

			playerInfo.ConnFail = rawPlayer["+conn_fail"]
			if playerInfo.ConnFail == "" {
				playerInfo.ConnFail = "0"
			}

			playerInfo.Suspend = rawPlayer["dwc_suspend"]

			groupsCopy[i].Players[joinIndex] = playerInfo
		}
	}

	if sorted {
		sort.Slice(groupsCopy, func(i, j int) bool {
			if groupsCopy[i].CreateTime.Equal(groupsCopy[j].CreateTime) {
				return groupsCopy[i].GroupName < groupsCopy[j].GroupName
			}

			return groupsCopy[i].CreateTime.Before(groupsCopy[j].CreateTime)
		})
	}

	return groupsCopy
}
