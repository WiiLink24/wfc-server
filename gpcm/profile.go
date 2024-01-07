package gpcm

import (
	"strconv"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

func (g *GameSpySession) getProfile(command common.GameSpyCommand) {
	strProfileId := command.OtherValues["profileid"]
	profileId, err := strconv.ParseUint(strProfileId, 10, 32)
	if err != nil {
		// There was an error getting profile info.
		g.replyError(ErrGetProfile)
		return
	}

	logging.Info(g.ModuleName, "Looking up the profile of", aurora.Cyan(profileId).String())

	user := database.User{}
	locstring := ""

	mutex.Lock()
	if session, ok := sessions[uint32(profileId)]; ok && session.LoggedIn {
		locstring = session.LocString
		user = session.User
		mutex.Unlock()
	} else {
		mutex.Unlock()
		user, ok = database.GetProfile(pool, ctx, uint32(profileId))
		if !ok {
			// The profile info was requested on is invalid.
			g.replyError(ErrGetProfileBadProfile)
			return
		}
	}

	response := common.CreateGameSpyMessage(common.GameSpyCommand{
		Command:      "pi",
		CommandValue: "",
		OtherValues: map[string]string{
			"profileid":  command.OtherValues["profileid"],
			"nick":       user.UniqueNick,
			"userid":     strconv.FormatUint(uint64(user.UserId), 10),
			"email":      user.Email,
			"sig":        common.RandomHexString(32),
			"uniquenick": user.UniqueNick,
			"firstname":  user.FirstName,
			"lastname":   user.LastName,
			"pid":        "11",
			"lon":        "0.000000",
			"lat":        "0.000000",
			"loc":        locstring,
			"id":         command.OtherValues["id"],
		},
	})

	g.Conn.Write([]byte(response))
}

func (g *GameSpySession) updateProfile(command common.GameSpyCommand) {
	g.User.UpdateProfile(pool, ctx, command.OtherValues)
}
