package gpcm

import (
	"strconv"
	"wwfc/common"
)

var gpcmErrors = map[int]string{
	// General errors
	0: "There was an unknown error.",
	1: "There was an error parsing an incoming request.",
	2: "This request cannot be processed because you are not logged in.",
	3: "This request cannot be processed because the session key is invalid.",
	4: "This request cannot be processed because of a database error.",
	5: "There was an error connecting a network socket.",
	6: "This profile has been disconnected by another login.",
	7: "The server has closed the connection.",
	8: "There was a problem with the UDP layer.",

	// Log-in errors
	256: "There was an error logging in to the GP backend.",
	257: "The login attempt timed out.",
	258: "The nickname provided was incorrect.",
	259: "The email address provided was incorrect.",
	260: "The password provided is incorrect.",
	261: "The profile provided was incorrect.",
	262: "The profile has been deleted.",
	263: "The server has refused the connection.",
	264: "The server could not be authenticated.",
	265: "The uniquenick provided is incorrect.",
	266: "There was an error validating the pre-authentication.",
	267: "The login ticket was unable to be validated.",
	268: "The login ticket had expired and could not be used.",

	// New user errors
	512: "There was an error creating a new user.",
	513: "A profile with that nick already exists.",
	514: "The password does not match the email address.",
	515: "The uniquenick is invalid.",
	516: "The uniquenick is already in use.",

	// User updating errors
	768: "There was an error updating the user information.",
	769: "A user with the email adress provided already exists.",

	// New profile errors
	1024: "There was an error creating a new profile.",
	1025: "The nickname to be replaced does not exist.",
	1026: "A profile with the nickname provided already exists.",

	// Updating profile errors
	1280: "There was an error updating the profile information. ",
	1281: "A user with the nickname provided already exists.",

	// Buddy list errors
	1536: "There was an error adding a buddy.",
	1537: "The profile requesting to add a buddy is invalid.",
	1538: "The profile requested is invalid.",
	1539: "The profile requested is already a buddy.",
	1540: "The profile requested is on the local profile's block list.",
	1541: "The profile requested is blocking you.",

	// Errors while Buddy add auth
	1792: "There was an error authorizing an add buddy request.",
	1793: "The profile being authorized is invalid.",
	1794: "The signature for the authorization is invalid.",
	1795: "The profile requesting authorization is on a block list.",
	1796: "The profile requested is blocking you.",

	// Status errors
	2048: "There was an error with the status string.",

	// Buddy message errors
	2304: "There was an error sending a buddy message.",
	2305: "The profile the message was to be sent to is not a buddy.",
	2306: "The profile does not support extended info keys.",
	2307: "The buddy to send a message to is offline.",

	// Profile information getting errors
	2560: "There was an error getting profile info.",
	2561: "The profile info was requested on is invalid.",

	// Deleting buddy errors
	2816: "There was an error deleting the buddy.",
	2817: "The buddy to be deleted is not a buddy.",

	// Deleting profile errors
	3072: "There was an error deleting the profile.",
	3073: "The last profile cannot be deleted.",

	// Profile searching errors
	3328: "There was an error searching for a profile.",
	3329: "The search attempt failed to connect to the server.",
	3330: "The search did not return in a timely fashion.",

	// User checking errors
	3584: "There was an error checking the user account.",
	3585: "No account exists with the provided e-mail address.",
	3586: "No such profile exists for the provided e-mail adress.",
	3587: "The password is incorrect.",

	// Revoking buddy errors
	3840: "There was an error revoking the buddy.",
	3841: "You are not a buddy of the profile.",

	// Registering new unique nick errors
	4096: "There was an error registering the uniquenick.",
	4097: "The uniquenick is already taken.",
	4098: "The uniquenick is reserved.",
	4099: "Tried to register a nick with no namespace set.",

	// CD key errors
	4352: "There was an error registering the cdkey.",
	4353: "The cdkey is invalid.",
	4354: "The profile has already been registered with a different cdkey.",
	4355: "The cdkey has already been registered to another profile.",

	// Adding to block list errors
	4608: "There was an error adding the player to the blocked list.",
	4609: "The profile specified is already blocked.",

	// Removing from block list errors
	4864: "There was an error removing the player from the blocked list.",
	4865: "The profile specified was not a member of the blocked list.",
}

func createGameSpyError(err int) string {
	errMsg, ok := gpcmErrors[err]
	if !ok {
		errMsg = ""
	}

	command := common.GameSpyCommand{
		Command:      "error",
		CommandValue: "",
		OtherValues: map[string]string{
			"err":    strconv.Itoa(err),
			"errmsg": errMsg,
		},
	}

	if err < 300 {
		command.OtherValues["fatal"] = ""
	}

	return common.CreateGameSpyMessage(command)
}

func (g *GameSpySession) replyError(err int) {
	g.Conn.Write([]byte(createGameSpyError(err)))
}
