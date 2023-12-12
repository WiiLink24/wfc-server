package qr2

import (
	"fmt"
	"github.com/logrusorgru/aurora/v3"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/logging"
)

type Group struct {
	GroupID   uint32
	GroupName string
	GameName  string
	MatchType string
	MKWRegion string
	Server    *Session
	Players   map[*Session]bool
}

var groups = map[string]*Group{}

func processResvOK(moduleName string, cmd common.MatchCommandDataResvOK, sender, destination *Session) bool {
	if len(groups) >= 100000 {
		logging.Error(moduleName, "Hit arbitrary global maximum group count (somehow)")
		return false
	}

	group := sender.GroupPointer
	if group == nil {
		group = &Group{
			GroupID:   cmd.GroupID,
			GroupName: "",
			GameName:  sender.Data["gamename"],
			MatchType: sender.Data["dwc_mtype"],
			MKWRegion: "",
			Server:    sender,
			Players:   map[*Session]bool{sender: true},
		}

		for {
			groupName := common.RandomString(6)
			if groups[groupName] != nil {
				continue
			}

			group.GroupName = groupName
			break
		}

		if group.GameName == "mariokartwii" {
			rk := sender.Data["rk"]

			// Check and remove regional searches due to the limited player count
			// China (ID 6) gets a pass because it was never released
			if len(rk) == 4 && (strings.HasPrefix(rk, "vs_") || strings.HasPrefix(rk, "bt_")) && rk[3] >= '0' && rk[3] < '6' {
				rk = rk[:2]
			}

			group.MKWRegion = rk
		}

		sender.GroupPointer = group
		groups[group.GroupName] = group

		logging.Notice(moduleName, "Created new group", aurora.Cyan(group.GroupName))
	}

	// TODO: Check if the sender is the actual server (host) once host migration works

	// Keep group ID updated
	sender.GroupPointer.GroupID = cmd.GroupID

	logging.Info(moduleName, "New player", aurora.BrightCyan(destination.Data["dwc_pid"]), "in group", aurora.Cyan(group.GroupName))
	group.Players[destination] = true
	destination.GroupPointer = group

	return true
}

func ProcessGPResvOK(cmd common.MatchCommandDataResvOK, senderIP uint64, senderPid uint32, destIP uint64, destPid uint32) bool {
	senderPidStr := strconv.FormatUint(uint64(senderPid), 10)
	destPidStr := strconv.FormatUint(uint64(destPid), 10)

	moduleName := "QR2:GPMsg:" + senderPidStr + "->" + destPidStr

	mutex.Lock()
	defer mutex.Unlock()

	from := sessionByPublicIP[senderIP]
	if from == nil {
		logging.Error(moduleName, "Sender IP does not exist:", aurora.Cyan(fmt.Sprintf("%012x", senderIP)))
		return false
	}

	to := sessionByPublicIP[destIP]
	if to == nil {
		logging.Error(moduleName, "Destination IP does not exist:", aurora.Cyan(fmt.Sprintf("%012x", destIP)))
		return false
	}

	// Validate dwc_pid values
	if !from.setProfileID(moduleName, senderPidStr) {
		return false
	}

	if !to.setProfileID(moduleName, destPidStr) {
		return false
	}

	return processResvOK(moduleName, cmd, from, to)
}

func ProcessGPStatusUpdate(senderIP uint64, status string) {
	if status == "0" || status == "1" || status == "3" || status == "4" {
		mutex.Lock()
		defer mutex.Unlock()

		session := sessionByPublicIP[senderIP]
		if session == nil || session.GroupPointer == nil {
			return
		}

		delete(session.GroupPointer.Players, session)

		if len(session.GroupPointer.Players) == 0 {
			logging.Notice("QR2", "Deleting group", aurora.Cyan(session.GroupPointer.GroupName))
			delete(groups, session.GroupPointer.GroupName)
		} else if session.GroupPointer.Server == session {
			logging.Notice("QR2", "Server down in group", aurora.Cyan(session.GroupPointer.GroupName))
			session.GroupPointer.Server = nil
			// TODO: Search for new host via dwc_hoststate
		}

		session.GroupPointer = nil
	}
}

type GroupInfo struct {
	GroupName string              `json:"id"`
	GameName  string              `json:"gamename"`
	MatchType string              `json:"type"`
	Suspend   bool                `json:"suspend"`
	ServerPID string              `json:"host,omitempty"`
	MKWRegion string              `json:"rk,omitempty"`
	Players   []map[string]string `json:"players"`
}

// GetGroups returns an unsorted copy of all online rooms
func GetGroups(gameName string) []GroupInfo {
	var groupsCopy []GroupInfo

	mutex.Lock()
	for _, group := range groups {
		if gameName != "" && gameName != group.GameName {
			continue
		}

		groupInfo := GroupInfo{
			GroupName: group.GroupName,
			GameName:  group.GameName,
			Suspend:   true,
			ServerPID: "",
			MKWRegion: "",
			Players:   []map[string]string{},
		}

		if group.MatchType == "0" || group.MatchType == "1" {
			groupInfo.MatchType = "anybody"
		} else if group.MatchType == "2" || group.MatchType == "3" {
			groupInfo.MatchType = "private"
		} else {
			groupInfo.MatchType = "unknown"
		}

		if groupInfo.MatchType == "private" && group.Server != nil {
			groupInfo.ServerPID = group.Server.Data["dwc_pid"]
		}

		if groupInfo.GameName == "mariokartwii" {
			groupInfo.MKWRegion = group.MKWRegion
		}

		for session, _ := range group.Players {
			mapData := map[string]string{}
			for k, v := range session.Data {
				mapData[k] = v
			}
			groupInfo.Players = append(groupInfo.Players, mapData)

			if mapData["dwc_hoststate"] == "2" && mapData["dwc_suspend"] == "0" {
				groupInfo.Suspend = false
			}
		}

		groupsCopy = append(groupsCopy, groupInfo)
	}
	mutex.Unlock()

	return groupsCopy
}
