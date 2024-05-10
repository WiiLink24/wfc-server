package gpcm

import (
	"context"
	"encoding/gob"
	"fmt"
	"os"
	"strings"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"
	"wwfc/qr2"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
	"github.com/sasha-s/go-deadlock"
)

var ServerName = "gpcm"

type GameSpySession struct {
	ConnIndex           uint64
	RemoteAddr          string
	User                database.User
	ModuleName          string
	LoggedIn            bool
	DeviceAuthenticated bool
	Challenge           string
	AuthToken           string
	LoginTicket         string
	SessionKey          int32

	LoginInfoSet      bool
	GameName          string
	GameCode          string
	Region            byte
	Language          byte
	InGameName        string
	ConsoleFriendCode uint64
	DeviceId          uint32
	HostPlatform      string
	UnitCode          byte

	StatusSet      bool
	Status         string
	LocString      string
	FriendList     []uint32
	AuthFriendList []uint32
	// For syncing with local GS SDK buddy list
	RecvStatusFromList []uint32

	QR2IP          uint64
	Reservation    common.MatchCommandData
	ReservationPID uint32

	NeedsExploit bool

	ReadBuffer  []byte
	WriteBuffer string
}

var (
	ctx  = context.Background()
	pool *pgxpool.Pool
	// I would use a sync.Map instead of the map mutex combo, but this performs better.
	sessions            = map[uint32]*GameSpySession{}
	sessionsByConnIndex = map[uint64]*GameSpySession{}
	mutex               = deadlock.Mutex{}

	allowDefaultDolphinKeys bool
)

func StartServer(reload bool) {
	qr2.SetGPErrorCallback(KickPlayer)

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

	database.UpdateTables(pool, ctx)

	allowDefaultDolphinKeys = config.AllowDefaultDolphinKeys

	if reload {
		err := loadState()
		if err != nil {
			logging.Error("GPCM", "Failed to load state:", err)
			os.Exit(1)
		}

		logging.Notice("GPCM", "Loaded", aurora.Cyan(len(sessions)), "sessions")
	}
}

func Shutdown() {
	err := saveState()
	if err != nil {
		logging.Error("GPCM", "Failed to save state:", err)
	}
	logging.Notice("GPCM", "Saved", aurora.Cyan(len(sessions)), "sessions")
}

func CloseConnection(index uint64) {
	mutex.Lock()
	session := sessionsByConnIndex[index]
	mutex.Unlock()

	if session == nil {
		logging.Error("GPCM", "Cannot find session for this connection index:", aurora.Cyan(index))
		return
	}

	logging.Notice(session.ModuleName, "Connection closed")

	if session.LoggedIn {
		qr2.Logout(session.User.ProfileId)
		if session.QR2IP != 0 {
			qr2.ProcessGPStatusUpdate(session.User.ProfileId, session.QR2IP, "0")
		}
		session.sendLogoutStatus()
	}

	mutex.Lock()
	defer mutex.Unlock()

	if session.LoggedIn {
		session.LoggedIn = false
		delete(sessions, session.User.ProfileId)
	}
}

func NewConnection(index uint64, address string) {
	session := &GameSpySession{
		ConnIndex:      index,
		RemoteAddr:     address,
		User:           database.User{},
		ModuleName:     "GPCM:" + address,
		LoggedIn:       false,
		Challenge:      common.RandomString(10),
		StatusSet:      false,
		Status:         "",
		LocString:      "",
		FriendList:     []uint32{},
		AuthFriendList: []uint32{},
	}

	payload := common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "lc",
		CommandValue: "1",
		OtherValues: map[string]string{
			"challenge": session.Challenge,
			"id":        "1",
		},
	})
	common.SendPacket(ServerName, index, []byte(payload))

	logging.Notice(session.ModuleName, "Connection established from", address)

	mutex.Lock()
	sessionsByConnIndex[index] = session
	mutex.Unlock()
}

func HandlePacket(index uint64, data []byte) {
	mutex.Lock()
	session := sessionsByConnIndex[index]
	mutex.Unlock()

	if session == nil {
		logging.Error("GPCM", "Cannot find session for this connection index:", aurora.Cyan(index))
		return
	}

	defer func() {
		if r := recover(); r != nil {
			logging.Error(session.ModuleName, "Panic:", r)
		}
	}()

	// Enforce maximum buffer size
	length := len(session.ReadBuffer) + len(data)
	if length > 0x4000 {
		logging.Error(session.ModuleName, "Buffer overflow")
		return
	}

	session.ReadBuffer = append(session.ReadBuffer, data...)

	// Packets can be received in fragments, so make sure we're at the end of a packet
	if string(session.ReadBuffer[max(0, length-7):length]) != `\final\` {
		return
	}

	builder := strings.Builder{}
	builder.Grow(length)

	// Copy one rune at a time to enforce ASCII (rather than UTF-8)
	for i := 0; i < length; i++ {
		if session.ReadBuffer[i] == 0 {
			logging.Error(session.ModuleName, "Null byte in packet")
			logging.Error(session.ModuleName, "Raw data:", string(data))
			session.replyError(ErrParse)
			session.ReadBuffer = []byte{}
			return
		}

		builder.WriteRune(rune(session.ReadBuffer[i]))
	}

	message := builder.String()
	session.ReadBuffer = []byte{}

	commands, err := common.ParseGameSpyMessage(message)
	if err != nil {
		logging.Error(session.ModuleName, "Error parsing message:", err.Error())
		logging.Error(session.ModuleName, "Raw data:", message)
		session.replyError(ErrParse)
		return
	}

	// Commands must be handled in a certain order, not in the order supplied by the client

	commands = session.handleCommand("ka", commands, func(command common.GameSpyCommand) {
		common.SendPacket(ServerName, session.ConnIndex, []byte(`\ka\\final\`))
	})
	commands = session.handleCommand("login", commands, session.login)
	commands = session.handleCommand("wwfc_exlogin", commands, session.exLogin)
	commands = session.ignoreCommand("logout", commands)

	if len(commands) != 0 && !session.LoggedIn {
		logging.Error(session.ModuleName, "Attempt to run command before login:", aurora.Cyan(commands[0]))
		session.replyError(ErrNotLoggedIn)
		return
	}

	commands = session.handleCommand("wwfc_report", commands, session.handleWWFCReport)
	commands = session.handleCommand("updatepro", commands, session.updateProfile)
	commands = session.handleCommand("status", commands, session.setStatus)
	commands = session.handleCommand("addbuddy", commands, session.addFriend)
	commands = session.handleCommand("delbuddy", commands, session.removeFriend)
	commands = session.handleCommand("authadd", commands, session.authAddFriend)
	commands = session.handleCommand("bm", commands, session.bestieMessage)
	commands = session.handleCommand("getprofile", commands, session.getProfile)

	for _, command := range commands {
		logging.Error(session.ModuleName, "Unknown command:", aurora.Cyan(command))
	}

	if session.WriteBuffer != "" {
		data := []byte{}
		logged := false
		for c := 0; c < len(session.WriteBuffer); c++ {
			if session.WriteBuffer[c] > 0xff || session.WriteBuffer[c] == 0x00 {
				if !logged {
					logging.Warn(session.ModuleName, "Non-char or null byte in response packet:", session.WriteBuffer)
					logged = true
				}
				continue
			}

			data = append(data, session.WriteBuffer[c])
		}

		common.SendPacket(ServerName, session.ConnIndex, data)
		session.WriteBuffer = ""
	}
}

func (g *GameSpySession) handleCommand(name string, commands []common.GameSpyCommand, handler func(command common.GameSpyCommand)) []common.GameSpyCommand {
	var unhandled []common.GameSpyCommand

	for _, command := range commands {
		if command.Command != name {
			unhandled = append(unhandled, command)
			continue
		}

		logging.Info(g.ModuleName, "Command:", aurora.Yellow(command.Command))
		handler(command)
	}

	return unhandled
}

func (g *GameSpySession) ignoreCommand(name string, commands []common.GameSpyCommand) []common.GameSpyCommand {
	var unhandled []common.GameSpyCommand

	for _, command := range commands {
		if command.Command != name {
			unhandled = append(unhandled, command)
		}
	}

	return unhandled
}

func saveState() error {
	file, err := os.OpenFile("state/gpcm_sessions.gob", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	encoder := gob.NewEncoder(file)

	mutex.Lock()
	defer mutex.Unlock()

	err = encoder.Encode(sessions)
	file.Close()
	return err
}

func loadState() error {
	file, err := os.Open("state/gpcm_sessions.gob")
	if err != nil {
		return err
	}

	decoder := gob.NewDecoder(file)

	mutex.Lock()
	defer mutex.Unlock()

	err = decoder.Decode(&sessions)
	file.Close()
	if err != nil {
		return err
	}

	for _, session := range sessions {
		sessionsByConnIndex[session.ConnIndex] = session
	}

	return nil
}
