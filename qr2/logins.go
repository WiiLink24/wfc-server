package qr2

type LoginInfo struct {
	ProfileID           uint32
	GameCode            string
	InGameName          string
	ConsoleFriendCode   uint64
	GPPublicIP          string
	NeedsExploit        bool
	DeviceAuthenticated bool
	Session             *Session
}

var logins = map[uint32]*LoginInfo{}

func Login(profileID uint32, gameCode string, inGameName string, consoleFriendCode uint64, publicIP string, needsExploit bool, deviceAuthenticated bool) {
	mutex.Lock()

	logins[profileID] = &LoginInfo{
		ProfileID:           profileID,
		GameCode:            gameCode,
		InGameName:          inGameName,
		ConsoleFriendCode:   consoleFriendCode,
		GPPublicIP:          publicIP,
		NeedsExploit:        needsExploit,
		DeviceAuthenticated: deviceAuthenticated,
		Session:             nil,
	}

	mutex.Unlock()
}

func SetDeviceAuthenticated(profileID uint32) {
	mutex.Lock()

	if login, exists := logins[profileID]; exists {
		login.DeviceAuthenticated = true
		if login.Session != nil {
			login.Session.Data["+deviceauth"] = "1"
		}
	}

	mutex.Unlock()
}

func Logout(profileID uint32) {
	mutex.Lock()

	// Delete login's session
	if login, exists := logins[profileID]; exists {
		if login.Session != nil {
			removeSession(makeLookupAddr(login.Session.Addr.String()))
		}
	}

	delete(logins, profileID)

	mutex.Unlock()
}
