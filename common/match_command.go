package common

import (
	"encoding/binary"
	"fmt"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
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
	Version int
	Command byte

	Reservation       *MatchCommandDataReservation
	ResvOK            *MatchCommandDataResvOK
	ResvDeny          *MatchCommandDataResvDeny
	TellAddr          *MatchCommandDataTellAddr
	ServerCloseClient *MatchCommandDataServerCloseClient
	SuspendMatch      *MatchCommandDataSuspendMatch
}

type MatchCommandDataReservation struct {
	MatchType        byte
	HasPublicIP      bool
	PublicIP         uint32
	PublicPort       uint16
	LocalIP          uint32
	LocalPort        uint16
	Unknown          uint32
	IsFriend         bool
	LocalPlayerCount uint32
	ResvCheckValue   uint32

	UserData []byte
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

	// Version 3 and 11
	ProfileIDs []uint32

	// Version 11
	IsFriend bool

	UserData []byte
}

type MatchCommandDataResvDeny struct {
	Reason       uint32
	ReasonString string

	UserData []byte
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
	HostProfileID  uint32
	IsHostFlag     uint32
	Short          bool
	SuspendValue   uint32
	ClientAIDValue uint32
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

func DecodeMatchCommand(command byte, buffer []byte, version int) (MatchCommandData, bool) {
	if version != 3 && version != 11 && version != 90 {
		return MatchCommandData{}, false
	}

	// Match commands must be 4 byte aligned
	if (len(buffer) & 3) != 0 {
		return MatchCommandData{}, false
	}

	switch command {
	case MatchReservation:
		if version == 3 && len(buffer) < 0x0C {
			break
		}

		if (version == 11 && len(buffer) < 0x14) || (version == 90 && len(buffer) < 0x24) {
			break
		}

		matchType := binary.LittleEndian.Uint32(buffer[0x00:0x04])
		if matchType > 3 {
			break
		}

		if version == 3 && len(buffer) < 0x0C {
			return MatchCommandData{
				Version: version,
				Command: command,
				Reservation: &MatchCommandDataReservation{
					MatchType:   byte(matchType),
					HasPublicIP: false,
				},
			}, true
		}

		publicPort := binary.LittleEndian.Uint32(buffer[0x08:0x0C])
		if publicPort > 0xffff {
			break
		}

		switch version {
		case 3:
			return MatchCommandData{
				Version: version,
				Command: command,
				Reservation: &MatchCommandDataReservation{
					MatchType:   byte(matchType),
					HasPublicIP: true,
					PublicIP:    binary.BigEndian.Uint32(buffer[0x04:0x08]),
					PublicPort:  uint16(publicPort),
					UserData:    buffer[0x0C:],
				},
			}, true

		case 11:
			isFriendValue := binary.LittleEndian.Uint32(buffer[0x0C:0x10])
			if isFriendValue > 1 {
				break
			}
			isFriend := isFriendValue != 0

			return MatchCommandData{
				Version: version,
				Command: command,
				Reservation: &MatchCommandDataReservation{
					MatchType:        byte(matchType),
					HasPublicIP:      true,
					PublicIP:         binary.BigEndian.Uint32(buffer[0x04:0x08]),
					PublicPort:       uint16(publicPort),
					IsFriend:         isFriend,
					LocalPlayerCount: binary.LittleEndian.Uint32(buffer[0x10:0x14]),
					UserData:         buffer[0x14:],
				},
			}, true

		case 90:
			localPort := binary.LittleEndian.Uint32(buffer[0x10:0x14])
			if localPort > 0xffff {
				break
			}

			isFriendValue := binary.LittleEndian.Uint32(buffer[0x18:0x1C])
			if isFriendValue > 1 {
				break
			}
			isFriend := isFriendValue != 0

			return MatchCommandData{
				Version: version,
				Command: command,
				Reservation: &MatchCommandDataReservation{
					MatchType:        byte(matchType),
					HasPublicIP:      true,
					PublicIP:         binary.BigEndian.Uint32(buffer[0x04:0x08]),
					PublicPort:       uint16(publicPort),
					LocalIP:          binary.BigEndian.Uint32(buffer[0x0C:0x10]),
					LocalPort:        uint16(localPort),
					Unknown:          binary.LittleEndian.Uint32(buffer[0x14:0x18]),
					IsFriend:         isFriend,
					LocalPlayerCount: binary.LittleEndian.Uint32(buffer[0x1C:0x20]),
					ResvCheckValue:   binary.LittleEndian.Uint32(buffer[0x20:0x24]),
					UserData:         buffer[0x24:],
				}}, true
		}

	case MatchResvOK:
		if version == 3 || version == 11 {
			if len(buffer) < 0xC {
				break
			}

			clientCount := binary.LittleEndian.Uint32(buffer[0x00:0x04])
			if version == 3 && (clientCount > 29 || len(buffer) < int(0x0C+clientCount*0x4)) {
				break
			}
			if version == 11 && (clientCount > 24 || len(buffer) < int(0x20+clientCount*0x4)) {
				break
			}

			var profileIDs []uint32
			for i := uint32(0); i < clientCount; i++ {
				profileIDs = append(profileIDs, binary.LittleEndian.Uint32(buffer[0x04+i*4:0x04+i*4+4]))
			}

			index := 0x04 + clientCount*4

			publicPort := binary.LittleEndian.Uint32(buffer[index+0x04 : index+0x08])
			if publicPort > 0xffff {
				break
			}

			if version == 3 {
				return MatchCommandData{
					Version: version,
					Command: command,
					ResvOK: &MatchCommandDataResvOK{
						PublicIP:    binary.BigEndian.Uint32(buffer[index : index+0x04]),
						PublicPort:  uint16(publicPort),
						ClientCount: clientCount,
						ProfileIDs:  profileIDs,
						UserData:    buffer[index+0x8:],
					},
				}, true
			} else if version == 11 {
				isFriendValue := binary.LittleEndian.Uint32(buffer[index+0x08 : index+0x0C])
				if isFriendValue > 1 {
					break
				}
				isFriend := isFriendValue != 0

				return MatchCommandData{
					Version: version,
					Command: command,
					ResvOK: &MatchCommandDataResvOK{
						MaxPlayers:  binary.LittleEndian.Uint32(buffer[index+0x14 : index+0x18]),
						SenderAID:   binary.LittleEndian.Uint32(buffer[index+0x0C : index+0x10]),
						PublicIP:    binary.BigEndian.Uint32(buffer[index : index+0x04]),
						PublicPort:  uint16(publicPort),
						GroupID:     binary.LittleEndian.Uint32(buffer[index+0x10 : index+0x14]),
						ClientCount: clientCount,
						ProfileIDs:  profileIDs,
						IsFriend:    isFriend,
						UserData:    buffer[index+0x18:],
					}}, true
			}
			break
		}

		// Version 90
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

		return MatchCommandData{
			Version: version,
			Command: command,
			ResvOK: &MatchCommandDataResvOK{
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
				UserData:         buffer[0x34:],
			},
		}, true

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

		return MatchCommandData{
			Version: version,
			Command: command,
			ResvDeny: &MatchCommandDataResvDeny{
				Reason:       reason,
				ReasonString: reasonString,
				UserData:     buffer[0x4:],
			},
		}, true

	case MatchResvWait:
		if len(buffer) != 0x00 {
			break
		}
		return MatchCommandData{
			Version: version,
			Command: command,
		}, true

	case MatchResvCancel:
		if len(buffer) != 0x00 {
			break
		}
		return MatchCommandData{
			Version: version,
			Command: command,
		}, true

	case MatchTellAddr:
		if len(buffer) != 0x08 {
			break
		}

		localPort := binary.LittleEndian.Uint32(buffer[0x04:0x08])
		if localPort > 0xffff {
			break
		}

		return MatchCommandData{
			Version: version,
			Command: command,
			TellAddr: &MatchCommandDataTellAddr{
				LocalIP:   binary.BigEndian.Uint32(buffer[0x00:0x04]),
				LocalPort: uint16(localPort),
			},
		}, true

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
		var pids []uint32
		for i := 0; i < pidCount; i++ {
			pids = append(pids, binary.LittleEndian.Uint32(buffer[i*4:i*4+4]))
		}

		return MatchCommandData{
			Version: version,
			Command: command,
			ServerCloseClient: &MatchCommandDataServerCloseClient{
				ProfileIDs: pids,
			},
		}, true

	case MatchSuspendMatch:
		if len(buffer) == 0x08 {
			return MatchCommandData{
				Version: version,
				Command: command,
				SuspendMatch: &MatchCommandDataSuspendMatch{
					HostProfileID: binary.LittleEndian.Uint32(buffer[0x00:0x04]),
					IsHostFlag:    binary.LittleEndian.Uint32(buffer[0x04:0x08]),
					Short:         true,
				},
			}, true
		} else if len(buffer) == 0x10 {
			return MatchCommandData{
				Version: version,
				Command: command,
				SuspendMatch: &MatchCommandDataSuspendMatch{
					HostProfileID:  binary.LittleEndian.Uint32(buffer[0x00:0x04]),
					IsHostFlag:     binary.LittleEndian.Uint32(buffer[0x04:0x08]),
					Short:          false,
					SuspendValue:   binary.LittleEndian.Uint32(buffer[0x08:0x0C]),
					ClientAIDValue: binary.LittleEndian.Uint32(buffer[0x0C:0x10]),
				},
			}, true
		}
	}

	return MatchCommandData{}, false
}

func EncodeMatchCommand(command byte, data MatchCommandData) ([]byte, bool) {
	version := data.Version

	if version != 3 && version != 11 && version != 90 {
		return []byte{}, false
	}

	switch command {
	case MatchReservation:
		message := binary.LittleEndian.AppendUint32([]byte{}, uint32(data.Reservation.MatchType))

		if version == 3 && !data.Reservation.HasPublicIP {
			return message, true
		}

		message = binary.BigEndian.AppendUint32(message, data.Reservation.PublicIP)
		message = binary.LittleEndian.AppendUint32(message, uint32(data.Reservation.PublicPort))

		if version == 11 {
			isFriendInt := uint32(0)
			if data.Reservation.IsFriend {
				isFriendInt = 1
			}
			message = binary.LittleEndian.AppendUint32(message, isFriendInt)

			message = binary.LittleEndian.AppendUint32(message, data.Reservation.LocalPlayerCount)
		} else if version == 90 {
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
		}

		message = append(message, data.Reservation.UserData...)
		if (len(message) & 3) != 0 {
			return []byte{}, false
		}

		return message, true

	case MatchResvOK:
		if version == 3 || version == 11 {
			if int(data.ResvOK.ClientCount) != len(data.ResvOK.ProfileIDs) {
				return []byte{}, false
			}
			message := binary.LittleEndian.AppendUint32([]byte{}, data.ResvOK.ClientCount)
			for _, pid := range data.ResvOK.ProfileIDs {
				message = binary.LittleEndian.AppendUint32(message, pid)
			}
			message = binary.BigEndian.AppendUint32(message, data.ResvOK.PublicIP)
			message = binary.LittleEndian.AppendUint32(message, uint32(data.ResvOK.PublicPort))

			if version == 3 {
				message = append(message, data.ResvOK.UserData...)
				if (len(message) & 3) != 0 {
					return []byte{}, false
				}

				return message, true
			}

			// Version 11
			isFriendInt := uint32(0)
			if data.ResvOK.IsFriend {
				isFriendInt = 1
			}
			message = binary.LittleEndian.AppendUint32(message, isFriendInt)

			message = binary.LittleEndian.AppendUint32(message, data.ResvOK.SenderAID)
			message = binary.LittleEndian.AppendUint32(message, data.ResvOK.GroupID)
			message = binary.LittleEndian.AppendUint32(message, data.ResvOK.MaxPlayers)

			message = append(message, data.ResvOK.UserData...)
			if (len(message) & 3) != 0 {
				return []byte{}, false
			}

			return message, true
		}

		// Version 90
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

		message = append(message, data.ResvOK.UserData...)
		if (len(message) & 3) != 0 {
			return []byte{}, false
		}

		return message, true

	case MatchResvDeny:
		message := binary.LittleEndian.AppendUint32([]byte{}, data.ResvDeny.Reason)

		message = append(message, data.ResvDeny.UserData...)
		if (len(message) & 3) != 0 {
			return []byte{}, false
		}

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
		var message []byte
		for i := 0; i < len(data.ServerCloseClient.ProfileIDs); i++ {
			message = binary.LittleEndian.AppendUint32(message, data.ServerCloseClient.ProfileIDs[i])
		}
		return message, true

	case MatchSuspendMatch:
		message := binary.LittleEndian.AppendUint32([]byte{}, data.SuspendMatch.HostProfileID)
		message = binary.LittleEndian.AppendUint32(message, data.SuspendMatch.IsHostFlag)

		if !data.SuspendMatch.Short {
			message = binary.LittleEndian.AppendUint32(message, data.SuspendMatch.SuspendValue)
			message = binary.LittleEndian.AppendUint32(message, data.SuspendMatch.ClientAIDValue)
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
