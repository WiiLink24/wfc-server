package common

import (
	"encoding/binary"
	"fmt"
	"github.com/logrusorgru/aurora/v3"
	"wwfc/logging"
)

const (
	MatchReservation        = 0x01
	MatchResvOK             = 0x02
	MatchResvDeny           = 0x03
	MatchResvWait           = 0x04
	MatchResvCancel         = 0x05
	MatchTellAddr           = 0x06
	MatchNewPidAid          = 0x07
	MatchLinkClientsRequest = 0x08
	MatchLinkClientsSuccess = 0x09
	MatchCloseLink          = 0x0A
	MatchResvPrior          = 0x0B
	MatchCancel             = 0x0C
	MatchCancelSyn          = 0x0D
	MatchCancelSynAck       = 0x0E
	MatchCancelAck          = 0x0F
	MatchServerCloseClient  = 0x10
	MatchPollTimeout        = 0x11
	MatchPollToAck          = 0x12
	MatchServerConnBlock    = 0x13
	MatchFriendAccept       = 0x20
	MatchClientWaitPoll     = 0x40
	MatchKeepAliveToClient  = 0x41
	MatchServerDownQuery    = 0x52
	MatchServerDownAck      = 0x53
	MatchServerDownNak      = 0x54
	MatchServerDownKeep     = 0x55
	MatchSuspendMatch       = 0x82
	MatchClientAIDUsage     = 0x83
)

type MatchCommandData struct {
	Reservation       *MatchCommandDataReservation
	ResvOK            *MatchCommandDataResvOK
	ResvDeny          *MatchCommandDataResvDeny
	TellAddr          *MatchCommandDataTellAddr
	ServerCloseClient *MatchCommandDataServerCloseClient
	SuspendMatch      *MatchCommandDataSuspendMatch
}

type MatchCommandDataReservation struct {
	MatchType        byte
	PublicIP         uint32
	PublicPort       uint16
	LocalIP          uint32
	LocalPort        uint16
	Unknown          uint32
	IsFriend         bool
	LocalPlayerCount uint32
	ResvCheckValue   uint32
}

type MatchCommandDataResvOK struct {
	MaxPlayers       uint32
	SenderAID        uint32
	ProfileID        uint32
	PublicIP         uint32
	PublicPort       uint16
	LocalIP          uint32
	LocalPort        uint16
	Unknown          uint32
	LocalPlayerCount uint32
	GroupID          uint32
	ReceiverNewAID   uint32
	ClientCount      uint32
	ResvCheckValue   uint32
}

type MatchCommandDataResvDeny struct {
	Reason       uint32
	ReasonString string
}

type MatchCommandDataTellAddr struct {
	LocalIP   uint32
	LocalPort uint16
}

type MatchCommandDataServerCloseClient struct {
	// TODO: Check are these really profile IDs
	ProfileIDs []uint32
}

type MatchCommandDataSuspendMatch struct {
	HostProfileID      uint32
	IsHost             bool
	SuspendValue       *bool
	ClientAID          *uint32
	ClientAIDUsageMask *uint32
}

func GetMatchCommandString(command byte) string {
	switch command {
	case MatchReservation:
		return "RESERVATION"
	case MatchResvOK:
		return "RESV_OK"
	case MatchResvDeny:
		return "RESV_DENY"
	case MatchResvWait:
		return "RESV_WAIT"
	case MatchResvCancel:
		return "RESV_CANCEL"
	case MatchTellAddr:
		return "TELL_ADDR"
	case MatchNewPidAid:
		return "NEW_PID_AID"
	case MatchLinkClientsRequest:
		return "LINK_CLS_REQ"
	case MatchLinkClientsSuccess:
		return "LINK_CLS_SUC"
	case MatchCloseLink:
		return "CLOSE_LINK"
	case MatchResvPrior:
		return "RESV_PRIOR"
	case MatchCancel:
		return "CANCEL"
	case MatchCancelSyn:
		return "CANCEL_SYN"
	case MatchCancelSynAck:
		return "CANCEL_SYN_ACK"
	case MatchCancelAck:
		return "CANCEL_ACK"
	case MatchServerCloseClient:
		return "SC_CLOSE_CL"
	case MatchPollTimeout:
		return "POLL_TIMEOUT"
	case MatchPollToAck:
		return "POLL_TO_ACK"
	case MatchServerConnBlock:
		return "SC_CONN_BLOCK"
	case MatchFriendAccept:
		return "FRIEND_ACCEPT"
	case MatchClientWaitPoll:
		return "CL_WAIT_POLL"
	case MatchKeepAliveToClient:
		return "SV_KA_TO_CL"
	case MatchServerDownQuery:
		return "SVDOWNQUERY"
	case MatchServerDownAck:
		return "SVDOWN_ACK"
	case MatchServerDownNak:
		return "SVDOWN_NAK"
	case MatchServerDownKeep:
		return "SVDOWN_KEEP"
	case MatchSuspendMatch:
		return "SUSPEND_MATCH"
	case MatchClientAIDUsage:
		return "CLIENT_AID_USAGE"
	}
	return "UNKNOWN"
}

func DecodeMatchCommand(command byte, buffer []byte) (MatchCommandData, bool) {
	switch command {
	case MatchReservation:
		if len(buffer) != 0x24 {
			break
		}

		matchType := binary.LittleEndian.Uint32(buffer[0x00:0x04])
		if matchType > 3 {
			break
		}

		isFriendValue := binary.LittleEndian.Uint32(buffer[0x18:0x1C])
		if isFriendValue > 1 {
			break
		}
		isFriend := isFriendValue != 0

		publicPort := binary.LittleEndian.Uint32(buffer[0x08:0x0C])
		if publicPort > 0xffff {
			break
		}

		localPort := binary.LittleEndian.Uint32(buffer[0x10:0x14])
		if localPort > 0xffff {
			break
		}

		return MatchCommandData{Reservation: &MatchCommandDataReservation{
			MatchType:        byte(matchType),
			PublicIP:         binary.BigEndian.Uint32(buffer[0x04:0x08]),
			PublicPort:       uint16(publicPort),
			LocalIP:          binary.BigEndian.Uint32(buffer[0x0C:0x10]),
			LocalPort:        uint16(localPort),
			Unknown:          binary.LittleEndian.Uint32(buffer[0x14:0x18]),
			IsFriend:         isFriend,
			LocalPlayerCount: binary.LittleEndian.Uint32(buffer[0x1C:0x20]),
			ResvCheckValue:   binary.LittleEndian.Uint32(buffer[0x20:0x24]),
		}}, true

	case MatchResvOK:
		if len(buffer) != 0x34 {
			break
		}

		publicPort := binary.LittleEndian.Uint32(buffer[0x10:0x14])
		if publicPort > 0xffff {
			break
		}

		localPort := binary.LittleEndian.Uint32(buffer[0x18:0x1C])
		if localPort > 0xffff {
			break
		}

		return MatchCommandData{ResvOK: &MatchCommandDataResvOK{
			MaxPlayers:       binary.LittleEndian.Uint32(buffer[0x00:0x04]),
			SenderAID:        binary.LittleEndian.Uint32(buffer[0x04:0x08]),
			ProfileID:        binary.LittleEndian.Uint32(buffer[0x08:0x0C]),
			PublicIP:         binary.BigEndian.Uint32(buffer[0x0C:0x10]),
			PublicPort:       uint16(publicPort),
			LocalIP:          binary.BigEndian.Uint32(buffer[0x14:0x18]),
			LocalPort:        uint16(localPort),
			Unknown:          binary.LittleEndian.Uint32(buffer[0x1C:0x20]),
			LocalPlayerCount: binary.LittleEndian.Uint32(buffer[0x20:0x24]),
			GroupID:          binary.LittleEndian.Uint32(buffer[0x24:0x28]),
			ReceiverNewAID:   binary.LittleEndian.Uint32(buffer[0x28:0x2C]),
			ClientCount:      binary.LittleEndian.Uint32(buffer[0x2C:0x30]),
			ResvCheckValue:   binary.LittleEndian.Uint32(buffer[0x30:0x34]),
		}}, true

	case MatchResvDeny:
		if len(buffer) != 0x04 {
			break
		}

		reason := binary.LittleEndian.Uint32(buffer[0x00:0x04])
		reasonString := "Unknown"
		switch reason {
		case 0x10:
			reasonString = "Room is full"
			break
		case 0x11:
			reasonString = "Room has already started"
			break
		case 0x12:
			reasonString = "Room is suspended"
			break
		}

		return MatchCommandData{ResvDeny: &MatchCommandDataResvDeny{
			Reason:       reason,
			ReasonString: reasonString,
		}}, true

	case MatchResvWait:
		if len(buffer) != 0x00 {
			break
		}
		return MatchCommandData{}, true

	case MatchResvCancel:
		if len(buffer) != 0x00 {
			break
		}
		return MatchCommandData{}, true

	case MatchTellAddr:
		if len(buffer) != 0x08 {
			break
		}

		localPort := binary.LittleEndian.Uint32(buffer[0x04:0x08])
		if localPort > 0xffff {
			break
		}

		return MatchCommandData{TellAddr: &MatchCommandDataTellAddr{
			LocalIP:   binary.BigEndian.Uint32(buffer[0x00:0x04]),
			LocalPort: uint16(localPort),
		}}, true

	case MatchServerCloseClient:
		// Max match command buffer size for QR2/GT2
		// This gives a maximum of 32 clients in the room
		if len(buffer) > 0x80 {
			break
		}

		if (len(buffer) & 3) != 0 {
			break
		}

		pidCount := len(buffer) >> 2
		pids := []uint32{}
		for i := 0; i < pidCount; i++ {
			pids = append(pids, binary.LittleEndian.Uint32(buffer[i*4:i*4+4]))
		}

		return MatchCommandData{ServerCloseClient: &MatchCommandDataServerCloseClient{
			ProfileIDs: pids,
		}}, true

	case MatchSuspendMatch:
		if len(buffer) != 0x08 && len(buffer) != 0x10 {
			break
		}

		isHostValue := binary.LittleEndian.Uint32(buffer[0x04:0x08])
		if isHostValue > 1 {
			break
		}
		isHost := isHostValue != 0

		if len(buffer) == 0x8 {
			if !isHost {
				// This is just the host acknowledging a client request
				break
			}

			return MatchCommandData{SuspendMatch: &MatchCommandDataSuspendMatch{
				HostProfileID: binary.LittleEndian.Uint32(buffer[0x00:0x04]),
				IsHost:        true,
			}}, true
		}

		suspendValueInt := binary.LittleEndian.Uint32(buffer[0x04:0x08])
		if suspendValueInt > 1 {
			break
		}
		suspendValue := suspendValueInt != 0
		suspendArgument := binary.LittleEndian.Uint32(buffer[0x0C:0x10])

		if isHost {
			return MatchCommandData{SuspendMatch: &MatchCommandDataSuspendMatch{
				HostProfileID:      binary.LittleEndian.Uint32(buffer[0x00:0x04]),
				IsHost:             true,
				SuspendValue:       &suspendValue,
				ClientAIDUsageMask: &suspendArgument,
			}}, true
		}

		return MatchCommandData{SuspendMatch: &MatchCommandDataSuspendMatch{
			HostProfileID: binary.LittleEndian.Uint32(buffer[0x00:0x04]),
			IsHost:        false,
			SuspendValue:  &suspendValue,
			ClientAID:     &suspendArgument,
		}}, true

	}

	return MatchCommandData{}, false
}

func EncodeMatchCommand(command byte, data MatchCommandData) ([]byte, bool) {
	switch command {
	case MatchReservation:
		message := binary.LittleEndian.AppendUint32([]byte{}, uint32(data.Reservation.MatchType))
		message = binary.BigEndian.AppendUint32(message, data.Reservation.PublicIP)
		message = binary.LittleEndian.AppendUint32(message, uint32(data.Reservation.PublicPort))
		message = binary.BigEndian.AppendUint32(message, data.Reservation.LocalIP)
		message = binary.LittleEndian.AppendUint32(message, uint32(data.Reservation.LocalPort))
		message = binary.LittleEndian.AppendUint32(message, data.Reservation.Unknown)

		isFriendInt := uint32(0)
		if data.Reservation.IsFriend {
			isFriendInt = 1
		}
		message = binary.LittleEndian.AppendUint32(message, isFriendInt)

		message = binary.LittleEndian.AppendUint32(message, data.Reservation.LocalPlayerCount)
		message = binary.LittleEndian.AppendUint32(message, data.Reservation.ResvCheckValue)
		return message, true

	case MatchResvOK:
		message := binary.LittleEndian.AppendUint32([]byte{}, data.ResvOK.MaxPlayers)
		message = binary.LittleEndian.AppendUint32(message, data.ResvOK.SenderAID)
		message = binary.LittleEndian.AppendUint32(message, data.ResvOK.ProfileID)
		message = binary.BigEndian.AppendUint32(message, data.ResvOK.PublicIP)
		message = binary.LittleEndian.AppendUint32(message, uint32(data.ResvOK.PublicPort))
		message = binary.BigEndian.AppendUint32(message, data.ResvOK.LocalIP)
		message = binary.LittleEndian.AppendUint32(message, uint32(data.ResvOK.LocalPort))
		message = binary.LittleEndian.AppendUint32(message, data.ResvOK.Unknown)
		message = binary.LittleEndian.AppendUint32(message, data.ResvOK.LocalPlayerCount)
		message = binary.LittleEndian.AppendUint32(message, data.ResvOK.GroupID)
		message = binary.LittleEndian.AppendUint32(message, data.ResvOK.ReceiverNewAID)
		message = binary.LittleEndian.AppendUint32(message, data.ResvOK.ClientCount)
		message = binary.LittleEndian.AppendUint32(message, data.ResvOK.ResvCheckValue)
		return message, true

	case MatchResvDeny:
		message := binary.LittleEndian.AppendUint32([]byte{}, data.ResvDeny.Reason)
		return message, true

	case MatchResvWait:
		return []byte{}, true

	case MatchResvCancel:
		return []byte{}, true

	case MatchTellAddr:
		message := binary.BigEndian.AppendUint32([]byte{}, data.TellAddr.LocalIP)
		message = binary.LittleEndian.AppendUint32(message, uint32(data.TellAddr.LocalPort))
		return message, true

	case MatchServerCloseClient:
		message := []byte{}
		for i := 0; i < len(data.ServerCloseClient.ProfileIDs); i++ {
			message = binary.LittleEndian.AppendUint32(message, data.ServerCloseClient.ProfileIDs[i])
		}
		return message, true

	case MatchSuspendMatch:
		message := binary.LittleEndian.AppendUint32([]byte{}, data.SuspendMatch.HostProfileID)

		isHostInt := uint32(0)
		if data.SuspendMatch.IsHost {
			isHostInt = 1
		}
		message = binary.LittleEndian.AppendUint32(message, isHostInt)

		if data.SuspendMatch.SuspendValue != nil {
			suspendValueInt := uint32(0)
			if *data.SuspendMatch.SuspendValue {
				suspendValueInt = 1
			}
			message = binary.LittleEndian.AppendUint32(message, suspendValueInt)

			if data.SuspendMatch.ClientAID != nil {
				message = binary.LittleEndian.AppendUint32(message, *data.SuspendMatch.ClientAID)
			} else if data.SuspendMatch.ClientAIDUsageMask != nil {
				message = binary.LittleEndian.AppendUint32(message, *data.SuspendMatch.ClientAIDUsageMask)
			}
		}
		return message, true
	}

	return []byte{}, false
}

func LogMatchCommand(moduleName string, dest string, command byte, data MatchCommandData) {
	logging.Notice(moduleName, "Match", aurora.Yellow(GetMatchCommandString(command)), "to", aurora.BrightCyan(dest))

	if command == MatchReservation && data.Reservation != nil {
		logging.Info(moduleName, "Match type:", aurora.Cyan(fmt.Sprintf("0x%02X", data.Reservation.MatchType)))
		logging.Info(moduleName, "Local player count:", aurora.Cyan(data.Reservation.LocalPlayerCount))
	} else if command == MatchResvOK && data.ResvOK != nil {
		logging.Info(moduleName, "Group ID:", aurora.Cyan(data.ResvOK.GroupID))
		logging.Info(moduleName, "Local player count:", aurora.Cyan(data.ResvOK.LocalPlayerCount))
		logging.Info(moduleName, "Current client count:", aurora.Cyan(data.ResvOK.ClientCount))
		logging.Info(moduleName, "Max client count:", aurora.Cyan(data.ResvOK.MaxPlayers))
		logging.Info(moduleName, "Sender slot:", aurora.Cyan(data.ResvOK.SenderAID))
		logging.Info(moduleName, "Receiver's new slot:", aurora.Cyan(data.ResvOK.ReceiverNewAID))
	} else if command == MatchResvDeny && data.ResvDeny != nil {
		reasonByte := fmt.Sprintf("0x%02X", data.ResvDeny.Reason)
		logging.Notice(moduleName, "Reason:", aurora.BrightRed(data.ResvDeny.ReasonString), "("+aurora.Cyan(reasonByte).String()+")")
	}
}
