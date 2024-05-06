package gpcm

import (
	"context"
	"fmt"
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
	WriteBuffer         string
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

	database.UpdateTables(pool, ctx)

	allowDefaultDolphinKeys = config.AllowDefaultDolphinKeys
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

	commands, err := common.ParseGameSpyMessage(string(data))
	if err != nil {
		logging.Error(session.ModuleName, "Error parsing message:", err.Error())
		logging.Error(session.ModuleName, "Raw data:", string(data))
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
		common.SendPacket(ServerName, session.ConnIndex, []byte(session.WriteBuffer))
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
