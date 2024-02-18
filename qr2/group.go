package qr2

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

type Group struct {
	GroupID       uint32
	GroupName     string
	CreateTime    time.Time
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
			CreateTime:    time.Now(),
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

	logging.Notice(moduleName, "New player", aurora.BrightCyan(destination.Data["dwc_pid"]), "in group", aurora.Cyan(group.GroupName))

	group.LastJoinIndex++
	destination.Data["+joinindex"] = strconv.Itoa(group.LastJoinIndex)
	if matchVersion == 90 {
		destination.Data["+localplayers"] = strconv.FormatUint(uint64(reservation.LocalPlayerCount), 10)
	}

	if destination.GroupPointer != nil && destination.GroupPointer != group {
		destination.removeFromGroup()
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
	if !from.setProfileID(moduleName, senderPidStr, "") {
		return false
	}

	if !to.setProfileID(moduleName, destPidStr, "") {
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
		logging.Info(moduleName, "Received status update for non-existent profile ID", aurora.Cyan(profileID))
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

		if !session.setProfileID(moduleName, strconv.FormatUint(uint64(profileID), 10), "") {
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

		session.removeFromGroup()
	}
}

func checkReservationAllowed(moduleName string, sender, destination *Session, joinType byte) string {
	if sender.Login == nil || destination.Login == nil {
		return ""
	}

	if !sender.Login.Restricted && !destination.Login.Restricted {
		return "ok"
	}

	if joinType != 2 && joinType != 3 {
		return "restricted_join"
	}

	// TODO: Once OpenHost is implemented, disallow joining public rooms

	if destination.GroupPointer == nil {
		// Destination is not in a group, check their dwc_mtype instead
		if destination.Data["dwc_mtype"] != "2" && destination.Data["dwc_mtype"] != "3" {
			return "restricted_join"
		}

		// This is fine
		return "ok"
	}

	if destination.GroupPointer.MatchType != "private" {
		return "restricted_join"
	}

	return "ok"
}

func CheckGPReservationAllowed(senderIP uint64, senderPid uint32, destPid uint32, joinType byte) string {
	senderPidStr := strconv.FormatUint(uint64(senderPid), 10)
	destPidStr := strconv.FormatUint(uint64(destPid), 10)

	moduleName := "QR2:CheckReservation:" + senderPidStr + "->" + destPidStr

	mutex.Lock()
	defer mutex.Unlock()

	from := sessions[senderIP]
	if from == nil {
		logging.Error(moduleName, "Sender IP does not exist:", aurora.Cyan(fmt.Sprintf("%012x", senderIP)))
		return ""
	}

	toLogin := logins[destPid]
	if toLogin == nil {
		logging.Error(moduleName, "Destination profile ID does not exist:", aurora.Cyan(destPid))
		return ""
	}

	to := toLogin.Session
	if to == nil {
		logging.Error(moduleName, "Destination profile ID does not have a session")
		return ""
	}

	// Validate dwc_pid value
	if !from.setProfileID(moduleName, senderPidStr, "") || from.Login == nil {
		return ""
	}

	return checkReservationAllowed(moduleName, from, to, joinType)
}

func ProcessUSER(senderPid uint32, senderIP uint64, packet []byte) {
	moduleName := "QR2:ProcessUSER/" + strconv.FormatUint(uint64(senderPid), 10)

	mutex.Lock()
	login := logins[senderPid]
	if login == nil {
		mutex.Unlock()
		logging.Warn(moduleName, "Received USER packet from non-existent profile ID", aurora.Cyan(senderPid))
		return
	}

	gpErrorCallback := login.GPErrorCallback

	session := login.Session
	if session == nil {
		mutex.Unlock()
		logging.Warn(moduleName, "Received USER packet from profile ID", aurora.Cyan(senderPid), "but no session exists")
		return
	}
	mutex.Unlock()

	miiGroupCount := binary.BigEndian.Uint16(packet[0x04:0x06])
	if miiGroupCount != 2 {
		logging.Error(moduleName, "Received USER packet with unexpected Mii group count", aurora.Cyan(miiGroupCount))
		// Kick the client
		gpErrorCallback(senderPid, "malpacket")
		return
	}

	miiGroupBitflags := binary.BigEndian.Uint32(packet[0x00:0x04])

	var miiData []string
	var miiName []string
	for i := 0; i < int(miiGroupCount); i++ {
		if miiGroupBitflags&(1<<uint(i)) == 0 {
			continue
		}

		index := 0x08 + i*0x4C
		if common.RFLCalculateCRC(packet[index:index+0x4C]) != 0x0000 {
			logging.Error(moduleName, "Received USER packet with invalid Mii data CRC")
			gpErrorCallback(senderPid, "malpacket")
			return
		}

		createId := binary.BigEndian.Uint64(packet[index+0x18 : index+0x20])
		official, _ := common.RFLSearchOfficialData(createId)
		if official {
			miiName = append(miiName, "Player")
		} else {
			decodedName, err := common.GetWideString(packet[index+0x2:index+0x2+20], binary.BigEndian)
			if err != nil {
				logging.Error(moduleName, "Failed to parse Mii name:", err)
				gpErrorCallback(senderPid, "malpacket")
				return
			}

			miiName = append(miiName, decodedName)
		}

		miiData = append(miiData, base64.StdEncoding.EncodeToString(packet[index:index+0x4A]))
	}

	mutex.Lock()
	defer mutex.Unlock()

	for i, name := range miiName {
		session.Data["+mii"+strconv.Itoa(i)] = miiData[i]
		session.Data["+mii_name"+strconv.Itoa(i)] = name
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

type MiiInfo struct {
	MiiData string `json:"data"`
	MiiName string `json:"name"`
}

type PlayerInfo struct {
	Count      string `json:"count"`
	ProfileID  string `json:"pid"`
	InGameName string `json:"name"`

	// Mario Kart Wii-specific fields
	FriendCode string    `json:"fc,omitempty"`
	VersusELO  string    `json:"ev,omitempty"`
	BattleELO  string    `json:"eb,omitempty"`
	Mii        []MiiInfo `json:"mii,omitempty"`
}

type GroupInfo struct {
	GroupName   string                       `json:"id"`
	GameName    string                       `json:"game"`
	CreateTime  time.Time                    `json:"created"`
	MatchType   string                       `json:"type"`
	Suspend     bool                         `json:"suspend"`
	ServerIndex string                       `json:"host,omitempty"`
	MKWRegion   string                       `json:"rk,omitempty"`
	PlayersRaw  map[string]map[string]string `json:"-"`
	Players     map[string]PlayerInfo        `json:"players"`
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
			GroupName:   group.GroupName,
			GameName:    group.GameName,
			CreateTime:  group.CreateTime,
			MatchType:   "",
			Suspend:     true,
			ServerIndex: "",
			MKWRegion:   "",
			PlayersRaw:  map[string]map[string]string{},
			Players:     map[string]PlayerInfo{},
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

			if login := session.Login; login != nil {
				mapData["+ingamesn"] = login.InGameName
			} else {
				mapData["+ingamesn"] = ""
			}

			groupInfo.PlayersRaw[mapData["+joinindex"]] = mapData

			if mapData["dwc_hoststate"] == "2" && mapData["dwc_suspend"] == "0" {
				groupInfo.Suspend = false
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

			if rawPlayer["gamename"] == "mariokartwii" {
				playerInfo.VersusELO = rawPlayer["ev"]
				playerInfo.BattleELO = rawPlayer["eb"]

				pid, err := strconv.ParseUint(rawPlayer["dwc_pid"], 10, 32)
				if err == nil {
					playerInfo.FriendCode = common.CalcFriendCodeString(uint32(pid), "RMCJ")
				}
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
