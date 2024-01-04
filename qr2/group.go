package qr2

import (
	"fmt"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

type Group struct {
	GroupID       uint32
	GroupName     string
	GameName      string
	MatchType     string
	MKWRegion     string
	LastJoinIndex int
	Server        *Session
	Players       map[*Session]bool
}

var groups = map[string]*Group{}

func processResvOK(moduleName string, matchVersion int, reservation common.MatchCommandDataReservation, resvOK common.MatchCommandDataResvOK, sender, destination *Session) bool {
	if len(groups) >= 100000 {
		logging.Error(moduleName, "Hit arbitrary global maximum group count (somehow)")
		return false
	}

	group := sender.GroupPointer
	if group == nil {
		group = &Group{
			GroupID:       resvOK.GroupID,
			GroupName:     "",
			GameName:      sender.Data["gamename"],
			MatchType:     sender.Data["dwc_mtype"],
			MKWRegion:     "",
			LastJoinIndex: 0,
			Server:        sender,
			Players:       map[*Session]bool{sender: true},
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

		sender.Data["+joinindex"] = "0"
		if matchVersion == 90 {
			sender.Data["+localplayers"] = strconv.FormatUint(uint64(resvOK.LocalPlayerCount), 10)
		}

		sender.GroupPointer = group
		groups[group.GroupName] = group

		logging.Notice(moduleName, "Created new group", aurora.Cyan(group.GroupName))
	}

	// Keep group ID updated
	group.GroupID = resvOK.GroupID

	if group.Players[destination] {
		// Player is already in the group
		return true
	}

	logging.Info(moduleName, "New player", aurora.BrightCyan(destination.Data["dwc_pid"]), "in group", aurora.Cyan(group.GroupName))

	group.LastJoinIndex++
	destination.Data["+joinindex"] = strconv.Itoa(group.LastJoinIndex)
	if matchVersion == 90 {
		destination.Data["+localplayers"] = strconv.FormatUint(uint64(reservation.LocalPlayerCount), 10)
	}

	group.Players[destination] = true
	destination.GroupPointer = group

	return true
}

func ProcessGPResvOK(matchVersion int, reservation common.MatchCommandDataReservation, resvOK common.MatchCommandDataResvOK, senderIP uint64, senderPid uint32, destIP uint64, destPid uint32) bool {
	senderPidStr := strconv.FormatUint(uint64(senderPid), 10)
	destPidStr := strconv.FormatUint(uint64(destPid), 10)

	moduleName := "QR2:GPMsg:" + senderPidStr + "->" + destPidStr

	mutex.Lock()
	defer mutex.Unlock()

	from := sessions[senderIP]
	if from == nil {
		logging.Error(moduleName, "Sender IP does not exist:", aurora.Cyan(fmt.Sprintf("%012x", senderIP)))
		return false
	}

	to := sessions[destIP]
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

	return processResvOK(moduleName, matchVersion, reservation, resvOK, from, to)
}

func ProcessGPStatusUpdate(profileID uint32, senderIP uint64, status string) {
	moduleName := "QR2/GPStatus:" + strconv.FormatUint(uint64(profileID), 10)

	mutex.Lock()
	defer mutex.Unlock()

	login, exists := logins[profileID]
	if !exists || login == nil {
		logging.Error(moduleName, "Received status update for non-existent profile ID", aurora.Cyan(profileID))
		return
	}

	session := login.Session
	if session == nil {
		if senderIP == 0 {
			logging.Info(moduleName, "Received status update for profile ID", aurora.Cyan(profileID), "but no session exists")
			return
		}

		// Login with this profile ID
		session, exists = sessions[senderIP]
		if !exists || session == nil {
			logging.Info(moduleName, "Received status update for profile ID", aurora.Cyan(profileID), "but no session exists")
			return
		}

		if !session.setProfileID(moduleName, strconv.FormatUint(uint64(profileID), 10)) {
			return
		}
	}

	// Send the client message exploit if not received yet
	if status != "0" && status != "1" && !session.ExploitReceived && session.Login != nil && session.Login.NeedsExploit {
		sessionCopy := *session

		mutex.Unlock()
		logging.Notice(moduleName, "Sending SBCM exploit to DNS patcher client")
		sendClientExploit(moduleName, sessionCopy)
		mutex.Lock()
	}

	if status == "0" || status == "1" || status == "3" || status == "4" {
		session := sessions[senderIP]
		if session == nil || session.GroupPointer == nil {
			return
		}

		delete(session.GroupPointer.Players, session)

		if len(session.GroupPointer.Players) == 0 {
			logging.Notice("QR2", "Deleting group", aurora.Cyan(session.GroupPointer.GroupName))
			delete(groups, session.GroupPointer.GroupName)
		} else if session.GroupPointer.Server == session {
			logging.Notice("QR2", "Server down in group", aurora.Cyan(session.GroupPointer.GroupName))
			session.GroupPointer.findNewServer()
		}

		session.GroupPointer = nil
	}
}

// findNewServer attempts to find the new server/host in the group when the current server goes down.
// If no server is found, the group's server pointer is set to nil.
// Expects the mutex to be locked.
func (g *Group) findNewServer() {
	server := (*Session)(nil)
	serverJoinIndex := -1
	for session := range g.Players {
		if session.Data["dwc_hoststate"] != "2" {
			continue
		}

		joinIndex, err := strconv.Atoi(session.Data["+joinindex"])
		if err != nil {
			continue
		}

		if server == nil || joinIndex < serverJoinIndex {
			server = session
			serverJoinIndex = joinIndex
		}
	}

	g.Server = server
	g.updateMatchType()
}

// updateMatchType updates the match type of the group based on the host's dwc_mtype value.
// Expects the mutex to be locked.
func (g *Group) updateMatchType() {
	if g.Server == nil || g.Server.Data["dwc_mtype"] == "" {
		return
	}

	g.MatchType = g.Server.Data["dwc_mtype"]
}

type GroupInfo struct {
	GroupName   string                       `json:"id"`
	GameName    string                       `json:"game"`
	MatchType   string                       `json:"type"`
	Suspend     bool                         `json:"suspend"`
	ServerIndex string                       `json:"host,omitempty"`
	MKWRegion   string                       `json:"rk,omitempty"`
	Players     map[string]map[string]string `json:"players"`
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
			GroupName:   group.GroupName,
			GameName:    group.GameName,
			MatchType:   "",
			Suspend:     true,
			ServerIndex: "",
			MKWRegion:   "",
			Players:     map[string]map[string]string{},
		}

		if group.MatchType == "0" || group.MatchType == "1" {
			groupInfo.MatchType = "anybody"
		} else if group.MatchType == "2" || group.MatchType == "3" {
			groupInfo.MatchType = "private"
		} else {
			groupInfo.MatchType = "unknown"
		}

		if group.Server != nil {
			groupInfo.ServerIndex = group.Server.Data["+joinindex"]
		}

		if groupInfo.GameName == "mariokartwii" {
			groupInfo.MKWRegion = group.MKWRegion
		}

		for session := range group.Players {
			mapData := map[string]string{}
			for k, v := range session.Data {
				mapData[k] = v
			}

			mapData["+ingamesn"] = session.Login.InGameName

			groupInfo.Players[mapData["+joinindex"]] = mapData

			if mapData["dwc_hoststate"] == "2" && mapData["dwc_suspend"] == "0" {
				groupInfo.Suspend = false
			}
		}

		groupsCopy = append(groupsCopy, groupInfo)
	}
	mutex.Unlock()

	return groupsCopy
}
