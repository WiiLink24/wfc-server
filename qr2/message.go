package qr2

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/logrusorgru/aurora/v3"
	"strconv"
	"wwfc/common"
	"wwfc/logging"
)

func printHex(data []byte) string {
	logMsg := ""
	for i := 0; i < len(data); i++ {
		if (i % 32) == 0 {
			logMsg += "\n"
		}
		logMsg += fmt.Sprintf("%02x ", data[i])
	}

	return logMsg
}

func SendClientMessage(senderIP string, destSearchID uint64, message []byte) {
	moduleName := "QR2/MSG"

	var matchData common.MatchCommandData
	var sender *Session
	var receiver *Session
	senderIPInt, _ := common.IPFormatToInt(senderIP)

	useSearchID := destSearchID < (0x400 << 32)
	if useSearchID {
		receiver = sessionBySearchID[destSearchID]
	} else {
		// It's an IP address, used in some circumstances
		receiver = sessions[destSearchID]
	}

	if receiver == nil || !receiver.Authenticated {
		logging.Error(moduleName, "Destination", aurora.Cyan(destSearchID), "does not exist")
		return
	}

	if destPid, ok := receiver.Data["dwc_pid"]; !ok || destPid == "" {
		logging.Error(moduleName, "Destination", aurora.Cyan(destSearchID), "has no profile ID")
		return
	}

	// Decode and validate the message
	isNatnegPacket := false
	if bytes.Equal(message[:2], []byte{0xfd, 0xfc}) {
		// Sending natneg cookie
		isNatnegPacket = true
		if len(message) != 0xA {
			logging.Error(moduleName, "Received invalid length NATNEG packet")
			return
		}

		natnegID := binary.LittleEndian.Uint32(message[0x6:0xA])
		moduleName = "QR2/MSG:s" + strconv.FormatUint(uint64(natnegID), 10)
	} else if bytes.Equal(message[:4], []byte{0xbb, 0x49, 0xcc, 0x4d}) || bytes.Equal(message[:4], []byte("SBCM")) {
		// DWC match command
		if len(message) < 0x14 {
			logging.Error(moduleName, "Received invalid length match command packet")
			return
		}

		version := int(binary.LittleEndian.Uint32(message[0x04:0x08]))
		if version != 3 && version != 11 && version != 90 {
			logging.Error(moduleName, "Received invalid match version")
			return
		}

		senderProfileID := binary.LittleEndian.Uint32(message[0x10:0x14])
		moduleName = "QR2/MSG:p" + strconv.FormatUint(uint64(senderProfileID), 10)

		if (int(message[9]) + 0x14) != len(message) {
			logging.Error(moduleName, "Received invalid match command packet header")
			return
		}

		qr2IP := binary.BigEndian.Uint32(message[0x0C:0x10])
		qr2Port := binary.LittleEndian.Uint16(message[0x0A:0x0C])

		if senderIPInt != int32(qr2IP) {
			logging.Error(moduleName, "Wrong QR2 IP in match command packet header")
			return
		}

		sender = sessions[(uint64(qr2Port)<<32)|uint64(qr2IP)]
		if sender == nil || !sender.Authenticated {
			logging.Error(moduleName, "Session does not exist with QR2 IP and port")
			return
		}

		if !sender.setProfileID(moduleName, strconv.FormatUint(uint64(senderProfileID), 10)) {
			// Error already logged
			return
		}

		var ok bool
		matchData, ok = common.DecodeMatchCommand(message[8], message[0x14:], version)
		if !ok {
			logging.Error(moduleName, "Received invalid match command:", aurora.Cyan(printHex(message)))
			return
		}

		if message[8] == common.MatchReservation {
			if matchData.Reservation.MatchType == 3 {
				// TODO: Check that this is correct
				logging.Error(moduleName, "RESERVATION: Attempt to join a private room over ServerBrowser")
				return
			}

			if matchData.Reservation.HasPublicIP {
				if qr2IP != matchData.Reservation.PublicIP {
					logging.Error(moduleName, "RESERVATION: Public IP mismatch in header and command")
					return
				}

				if qr2Port != matchData.Reservation.PublicPort {
					logging.Error(moduleName, "RESERVATION: Public port mismatch in header and command")
					return
				}
			}

			if version == 90 {
				if matchData.Reservation.LocalPort < 1024 {
					logging.Error(moduleName, "RESERVATION: Local port is reserved")
					return
				}
			}

			if useSearchID {
				matchData.Reservation.PublicIP = uint32(sender.SearchID & 0xffffffff)
				matchData.Reservation.PublicPort = uint16((sender.SearchID >> 32) & 0xffff)
				matchData.Reservation.LocalIP = 0
				matchData.Reservation.LocalPort = 0
			}
		}

		if message[8] == common.MatchResvOK {
			if qr2IP != matchData.ResvOK.PublicIP {
				logging.Error(moduleName, "RESERVATION: Public IP mismatch in header and command")
				return
			}

			if qr2Port != matchData.ResvOK.PublicPort {
				logging.Error(moduleName, "RESERVATION: Public port mismatch in header and command")
				return
			}

			if version == 90 {
				if matchData.ResvOK.LocalPort < 1024 {
					logging.Error(moduleName, "RESV_OK: Local port is reserved")
					return
				}

				if matchData.ResvOK.ProfileID != senderProfileID {
					logging.Error(moduleName, "RESV_OK: Profile ID mismatch in header")
					return
				}
			}

			if useSearchID {
				matchData.ResvOK.PublicIP = uint32(sender.SearchID & 0xffffffff)
				matchData.ResvOK.PublicPort = uint16((sender.SearchID >> 32) & 0xffff)
				matchData.ResvOK.LocalIP = 0
				matchData.ResvOK.LocalPort = 0
			}
		}

		if message[8] == common.MatchTellAddr {
			mutex.Lock()
			if sender.Data["publicip"] != receiver.Data["publicip"] {
				mutex.Unlock()
				logging.Error(moduleName, "TELL_ADDR: Public IP does not match receiver")
				return
			}
			mutex.Unlock()

			if matchData.TellAddr.LocalPort < 1024 {
				logging.Error(moduleName, "TELL_ADDR: Local port is reserved")
				return
			}
		}

		if useSearchID {
			// Convert public IP to search ID
			qr2SearchID := binary.LittleEndian.AppendUint16([]byte{}, uint16((sender.SearchID>>32)&0xffff))
			qr2SearchID = binary.BigEndian.AppendUint32(qr2SearchID, uint32(sender.SearchID&0xffffffff))
			message = append(message[:0x0A], append(qr2SearchID, message[0x10:0x14]...)...)
		} else {
			message = message[:0x14]
		}

		var matchMessage []byte
		matchMessage, ok = common.EncodeMatchCommand(message[8], matchData, version)
		if !ok {
			logging.Error(moduleName, "Failed to reencode match command:", aurora.Cyan(printHex(message)))
			return
		}

		if len(matchMessage) != 0 {
			message = append(message, matchMessage...)
		}
	} else {
		logging.Error(moduleName, "Invalid message:", aurora.Cyan(printHex(message)))
	}

	mutex.Lock()

	destPid, ok := receiver.Data["dwc_pid"]
	if !ok || destPid == "" {
		destPid = "<UNKNOWN>"
	}

	destSessionID := receiver.SessionID
	packetCount := receiver.PacketCount + 1
	receiver.PacketCount = packetCount
	destAddr := receiver.Addr

	mutex.Unlock()

	if isNatnegPacket {
		cookie := binary.BigEndian.Uint32(message[0x2:0x6])
		logging.Notice(moduleName, "Send NN cookie", aurora.Cyan(cookie), "to", aurora.BrightCyan(destPid))
	} else {
		cmd := message[8]
		common.LogMatchCommand(moduleName, destPid, cmd, matchData)

		if cmd == common.MatchReservation {
			sender.ReservationID = receiver.SearchID
		} else if cmd == common.MatchResvOK || cmd == common.MatchResvDeny || cmd == common.MatchResvWait {
			if receiver.ReservationID != sender.SearchID {
				logging.Error(moduleName, "Destination has no reservation with the sender")
				return
			}

			if cmd == common.MatchResvOK {
				mutex.Lock()
				if !processResvOK(moduleName, *matchData.ResvOK, sender, receiver) {
					mutex.Unlock()
					return
				}
				mutex.Unlock()
			}
		}
	}

	payload := createResponseHeader(ClientMessageRequest, destSessionID)

	payload = append(payload, []byte{0, 0, 0, 0}...)
	binary.BigEndian.PutUint32(payload[len(payload)-4:], packetCount)
	payload = append(payload, message...)

	_, err := masterConn.WriteTo(payload, destAddr)
	if err != nil {
		logging.Error(moduleName, "Error sending message:", err.Error())
	}
}