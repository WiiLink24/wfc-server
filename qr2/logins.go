package qr2

type LoginInfo struct {
	ProfileID         uint32
	InGameName        string
	ConsoleFriendCode string
	GPPublicIP        string
}

var logins = map[uint32]LoginInfo{}

func Login(profileID uint32, inGameName, consoleFriendCode, publicIP string) {
	mutex.Lock()
	logins[profileID] = LoginInfo{
		ProfileID:         profileID,
		InGameName:        inGameName,
		ConsoleFriendCode: consoleFriendCode,
		GPPublicIP:        publicIP,
	}
	mutex.Unlock()
}

func Logout(profileID uint32) {
	mutex.Lock()
	delete(logins, profileID)
	mutex.Unlock()
}
