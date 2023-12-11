package qr2

import (
	"fmt"
	"github.com/logrusorgru/aurora/v3"
	"strconv"
	"wwfc/common"
	"wwfc/logging"
)

type Group struct {
	GroupID   uint32
	ServerAID uint8
	Players   map[uint8]*Session
}

var groups = map[*Group]bool{}

func processResvOK(moduleName string, cmd common.MatchCommandDataResvOK, sender, destination *Session) bool {
	group := sender.GroupPointer
	if group == nil {
		logging.Notice(moduleName, "Creating new group", aurora.Cyan(cmd.GroupID), "/", aurora.Cyan(cmd.ProfileID))
		group = &Group{
			GroupID:   cmd.GroupID,
			ServerAID: uint8(cmd.SenderAID),
			Players:   map[uint8]*Session{uint8(cmd.SenderAID): sender},
		}
		sender.GroupPointer = group
		groups[group] = true
	}

	if uint8(cmd.SenderAID) != sender.GroupAID {
		logging.Error(moduleName, "ResvOK: Invalid sender AID")
		return false
	}

	// TODO: Check if the sender is the actual server (host) once host migration works

	// Keep group ID updated
	sender.GroupPointer.GroupID = cmd.GroupID

	logging.Info(moduleName, "New AID", aurora.Cyan(uint8(cmd.ReceiverNewAID)), "in group", aurora.Cyan(group.GroupID))
	group.Players[uint8(cmd.ReceiverNewAID)] = destination
	destination.GroupPointer = group
	destination.GroupAID = uint8(cmd.ReceiverNewAID)

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
