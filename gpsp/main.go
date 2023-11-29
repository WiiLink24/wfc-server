package gpsp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
	"io"
	"net"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"
)

var (
	ctx    = context.Background()
	pool   *pgxpool.Pool
	userId int64
)

func StartServer() {
	// Get config
	config := common.GetConfig()

	// Start SQL
	dbString := fmt.Sprintf("postgres://%s:%s@%s/%s", config.Username, config.Password, config.DatabaseAddress, config.DatabaseName)
	dbConf, err := pgxpool.ParseConfig(dbString)
	if err != nil {
		panic(err)
	}

	pool, err = pgxpool.ConnectConfig(ctx, dbConf)
	if err != nil {
		panic(err)
	}

	address := config.Address + ":29901"
	l, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	// Close the listener when the application closes.
	defer l.Close()
	logging.Notice("GPSP", "Listening on", address)

	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}

		// Handle connections in a new goroutine.
		go handleRequest(conn)
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn) {
	defer conn.Close()

	moduleName := "GPSP"
	knownProfileId := uint32(0)

	err := conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		logging.Notice(moduleName, "Unable to set keepalive:", err.Error())
	}

	logging.Notice(moduleName, "Connection established from", aurora.BrightCyan(conn.RemoteAddr()))

	// Here we go into the listening loop
	for {
		buffer := make([]byte, 1024)
		_, err := bufio.NewReader(conn).Read(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				// Client closed connection, terminate.
				logging.Notice(moduleName, "Client closed connection")
				return
			}

			logging.Notice(moduleName, "Connection lost")
			return
		}

		commands, err := common.ParseGameSpyMessage(string(buffer))
		if err != nil {
			logging.Error(moduleName, err)
			return
		}

		for _, command := range commands {
			logging.Notice(moduleName, "Command:", aurora.Yellow(command.Command))
			switch command.Command {
			case "ka":
				conn.Write([]byte(`\ka\\final\`))
				break

			case "otherslist":
				strProfileId, ok := command.OtherValues["profileid"]
				if !ok {
					logging.Error(moduleName, "Missing profileid in otherslist")
					return
				}

				profileId, err := strconv.ParseUint(strProfileId, 10, 32)
				if err != nil {
					logging.Error(moduleName, err)
					return
				}

				if knownProfileId == 0 {
					knownProfileId = uint32(profileId)
					moduleName = "GPSP:" + strconv.FormatUint(profileId, 10)
					moduleName += "/" + common.CalcFriendCodeString(uint32(profileId), "RMCJ")
				} else if uint32(profileId) != knownProfileId {
					logging.Warn(moduleName, "Mismatched profile ID in otherslist:", aurora.Cyan(strProfileId))
				}

				logging.Notice(moduleName, "Lookup otherslist for", aurora.Cyan(profileId))
				conn.Write([]byte(handleOthersList(moduleName, uint32(profileId), command)))
				break
			}
		}
	}
}

func handleOthersList(moduleName string, profileId uint32, command common.GameSpyCommand) string {
	empty := `\otherslist\\final\`

	_, ok := command.OtherValues["sesskey"]
	if !ok {
		logging.Error(moduleName, "Missing sesskey in otherslist")
		return empty
	}

	numopids, ok := command.OtherValues["numopids"]
	if !ok {
		logging.Error(moduleName, "Missing numopids in otherslist")
		return empty
	}

	opids, ok := command.OtherValues["opids"]
	if !ok {
		logging.Error(moduleName, "Missing opids in otherslist")
		return empty
	}

	_, ok = command.OtherValues["gamename"]
	if !ok {
		logging.Error(moduleName, "Missing gamename in otherslist")
		return empty
	}

	numOpidsValue, err := strconv.Atoi(numopids)
	if err != nil {
		logging.Error(moduleName, err)
		return empty
	}

	opidsSplit := []string{}
	if strings.Contains(opids, "|") {
		opidsSplit = strings.Split(opids, "|")
	} else if opids != "" && opids != "0" {
		opidsSplit = append(opidsSplit, opids)
	}

	if len(opidsSplit) != numOpidsValue && opids != "0" {
		logging.Error(moduleName, "Mismatch opids length with numopids:", aurora.Cyan(len(opidsSplit)), "!=", aurora.Cyan(numOpidsValue))
		return empty
	}

	payload := `\otherslist\`
	for _, strOtherId := range opidsSplit {
		otherId, err := strconv.ParseUint(strOtherId, 10, 32)
		if err != nil {
			logging.Error(moduleName, err)
			continue
		}

		// TODO: Perhaps this could be condensed into one database query
		// Also TODO: Check if the players are actually friends
		user, ok := database.GetProfile(pool, ctx, uint32(otherId))
		if !ok {
			logging.Error(moduleName, "Other ID doesn't exist:", aurora.Cyan(strOtherId))
			// If the profile doesn't exist then skip adding it
			continue
		}

		payload += `\o\` + strconv.FormatUint(uint64(user.ProfileId), 10)
		payload += `\uniquenick\` + user.UniqueNick
	}

	payload += `\oldone\\final\`
	return payload
}
