package gamestats

import (
	"context"
	"fmt"
	"strings"
	"wwfc/common"
	"wwfc/database"
	"wwfc/gpcm"
	"wwfc/logging"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
	"github.com/sasha-s/go-deadlock"
)

var ServerName = "gamestats"

type GameStatsSession struct {
	ConnIndex  uint64
	RemoteAddr string
	ModuleName string
	Challenge  string

	SessionKey int32
	GameInfo   *common.GameInfo

	Authenticated bool
	LoginID       int
	User          database.User

	ReadBuffer  []byte
	WriteBuffer []byte
}

var (
	ctx  = context.Background()
	pool *pgxpool.Pool

	serverName string
	webSalt    string

	sessionsByConnIndex = make(map[uint64]*GameStatsSession)
	mutex               = deadlock.RWMutex{}
)

func StartServer(reload bool) {
	// Get config
	config := common.GetConfig()

	serverName = config.ServerName
	webSalt = common.RandomString(32)

	common.ReadGameList()

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
}

func Shutdown() {
}

func NewConnection(index uint64, address string) {
	session := &GameStatsSession{
		ConnIndex:  index,
		RemoteAddr: address,
		ModuleName: "GSTATS:" + address,
		Challenge:  common.RandomString(10),

		SessionKey: 0,
		GameInfo:   nil,

		Authenticated: false,
		LoginID:       0,
		User:          database.User{},

		ReadBuffer:  []byte{},
		WriteBuffer: []byte{},
	}

	session.Write(common.GameSpyCommand{
		Command:      "lc",
		CommandValue: "1",
		OtherValues: map[string]string{
			"challenge": session.Challenge,
			"id":        "1",
		},
	})
	common.SendPacket(ServerName, index, []byte(session.WriteBuffer))
	session.WriteBuffer = []byte{}

	logging.Notice(session.ModuleName, "Connection established from", address)

	mutex.Lock()
	sessionsByConnIndex[index] = session
	mutex.Unlock()
}

func CloseConnection(index uint64) {
	mutex.RLock()
	session := sessionsByConnIndex[index]
	mutex.RUnlock()

	if session == nil {
		logging.Error("GSTATS", "Cannot find session for this connection index:", aurora.Cyan(index))
		return
	}

	logging.Notice(session.ModuleName, "Connection closed")

	mutex.Lock()
	delete(sessionsByConnIndex, index)
	mutex.Unlock()
}

func HandlePacket(index uint64, data []byte) {
	mutex.RLock()
	session := sessionsByConnIndex[index]
	mutex.RUnlock()

	if session == nil {
		logging.Error("GSTATS", "Cannot find session for this connection index:", aurora.Cyan(index))
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

	// Decrypt the data, can decrypt multiple packets
	decrypted := strings.Builder{}
	decrypted.Grow(length)
	p := 0
	for i := 0; i < length; i++ {
		if string(session.ReadBuffer[i:i+7]) == `\final\` {
			decrypted.WriteString(`\final\`)

			i += 6
			p = 0
			continue
		}

		decrypted.WriteRune(rune(session.ReadBuffer[i] ^ "GameSpy3D"[p]))
		p = (p + 1) % 9
	}

	message := decrypted.String()

	commands, err := common.ParseGameSpyMessage(message)
	if err != nil {
		logging.Error(session.ModuleName, "Error parsing message:", err.Error())
		logging.Error(session.ModuleName, "Raw data:", message)
		session.replyError(gpcm.ErrParse)
		return
	}

	commands = session.handleCommand("ka", commands, func(command common.GameSpyCommand) {
		session.Write(common.GameSpyCommand{
			Command: "ka",
		})
	})

	commands = session.handleCommand("auth", commands, session.auth)
	commands = session.handleCommand("authp", commands, session.authp)

	if len(commands) != 0 && !session.Authenticated {
		logging.Error(session.ModuleName, "Attempt to run command before authentication:", aurora.Cyan(commands[0]))
		session.replyError(gpcm.ErrNotLoggedIn)
		return
	}

	commands = session.handleCommand("getpd", commands, session.getpd)
	commands = session.handleCommand("setpd", commands, session.setpd)
	common.UNUSED(session.ignoreCommand)

	for _, command := range commands {
		logging.Error(session.ModuleName, "Unknown command:", aurora.Cyan(command))
	}

	if len(session.WriteBuffer) > 0 {
		common.SendPacket(ServerName, session.ConnIndex, session.WriteBuffer)
		session.WriteBuffer = []byte{}
	}
}

func (g *GameStatsSession) handleCommand(name string, commands []common.GameSpyCommand, handler func(command common.GameSpyCommand)) []common.GameSpyCommand {
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

func (g *GameStatsSession) ignoreCommand(name string, commands []common.GameSpyCommand) []common.GameSpyCommand {
	var unhandled []common.GameSpyCommand

	for _, command := range commands {
		if command.Command != name {
			unhandled = append(unhandled, command)
		}
	}

	return unhandled
}

func (g *GameStatsSession) Write(command common.GameSpyCommand) {
	// Encrypt the data and append it to be sent
	payload := []byte(common.CreateGameSpyMessage(command))
	// Exclude trailing \final\
	for i := 0; i < len(payload)-7; i++ {
		payload[i] ^= "GameSpy3D"[i%9]
	}
	g.WriteBuffer = append(g.WriteBuffer, payload...)
}
