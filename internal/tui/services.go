package tui

import "strings"

// serviceSubcommands maps canonical service names to their available subcommands.
// Help text sourced from ergochat/ergo irc/nickserv.go, chanserv.go, hostserv.go, histserv.go.
var serviceSubcommands = map[string][]Command{
	"NICKSERV": {
		{Name: "IDENTIFY", Desc: "Login to your account\n" +
			"Syntax: IDENTIFY <username> [password]\n\n" +
			"IDENTIFY lets you login to the given username using either password auth, or\n" +
			"certfp (your client certificate) if a password is not given."},
		{Name: "REGISTER", Desc: "Create a new user account\n" +
			"Syntax: REGISTER <password> [email]\n\n" +
			"REGISTER lets you register your current nickname as a user account. If the\n" +
			"server allows anonymous registration, you can omit the e-mail address.\n\n" +
			"If you are currently logged in with a TLS client certificate and wish to use\n" +
			"it instead of a password to log in, send * as the password."},
		{Name: "GROUP", Desc: "Link current nickname to your account\n" +
			"Syntax: GROUP\n\n" +
			"GROUP links your current nickname with your logged-in account, so other people\n" +
			"will not be able to use it."},
		{
			Name: "DROP", Desc: "De-link a nickname from your account\n" +
				"Syntax: DROP [nickname]\n\n" +
				"DROP de-links the given (or your current) nickname from your user account.",
			Args: []ArgType{ArgNick},
		},
		{Name: "UNREGISTER", Desc: "Delete your user account\n" +
			"Syntax: UNREGISTER <username> [code]\n\n" +
			"UNREGISTER lets you delete your user account (or someone else's, if you're an\n" +
			"IRC operator with the correct permissions). To prevent accidental\n" +
			"unregistrations, a verification code is required; invoking the command without\n" +
			"a code will display the necessary code."},
		{
			Name: "INFO", Desc: "Display account information\n" +
				"Syntax: INFO [username]\n\n" +
				"INFO gives you information about the given (or your own) user account.",
			Args: []ArgType{ArgNick},
		},
		{
			Name: "GHOST", Desc: "Reclaim your nickname\n" +
				"Syntax: GHOST <nickname>\n\n" +
				"GHOST disconnects the given user from the network if they're logged in with the\n" +
				"same user account, letting you reclaim your nickname.",
			Args: []ArgType{ArgNick},
		},
		{Name: "PASSWD", Aliases: []string{"PASSWORD"}, Desc: "Change your account password\n" +
			"Syntax: PASSWD <current> <new> <new_again>\n" +
			"Or:     PASSWD <username> <new>\n\n" +
			"PASSWD lets you change your account password. You must supply your current\n" +
			"password and confirm the new one by typing it twice. If you're an IRC operator\n" +
			"with the correct permissions, you can use PASSWD to reset someone else's\n" +
			"password by supplying their username and then the desired password. To\n" +
			"indicate an empty password, use * instead."},
		{Name: "VERIFY", Desc: "Complete account registration with a code\n" +
			"Syntax: VERIFY <username> <code>\n\n" +
			"VERIFY lets you complete an account registration, if the server requires email\n" +
			"or other verification."},
		{Name: "SET", Desc: "Modify your account settings\n" +
			"Syntax: SET <setting> <value>\n\n" +
			"SET modifies your account settings. Available settings: ENFORCE, MULTICLIENT,\n" +
			"AUTOREPLAY-LINES, REPLAY-JOINS, ALWAYS-ON, AUTOREPLAY-MISSED, DM-HISTORY,\n" +
			"AUTO-AWAY, EMAIL. Use HELP SET for details on each setting."},
		{Name: "GET", Desc: "Query your account settings\n" +
			"Syntax: GET <setting>\n\n" +
			"GET queries the current values of your account settings. For more information\n" +
			"on the settings and their possible values, see HELP SET."},
		{Name: "LIST", Desc: "Search registered nicknames\n" +
			"Syntax: LIST [regex]\n\n" +
			"LIST returns the list of registered nicknames, which match the given regex.\n" +
			"If no regex is provided, all registered nicknames are returned."},
		{Name: "CERT", Desc: "Manage TLS certificate fingerprints\n" +
			"Syntax: CERT <LIST | ADD | DEL> [account] [certfp]\n\n" +
			"CERT examines or modifies the SHA-256 TLS certificate fingerprints that can\n" +
			"be used to log into an account. CERT LIST lists the authorized fingerprints,\n" +
			"CERT ADD <fingerprint> adds a new fingerprint, and CERT DEL <fingerprint>\n" +
			"removes a fingerprint.", Subcommands: []Command{
			{Name: "LIST", Desc: "List authorized fingerprints\n" +
				"Syntax: CERT LIST [account]\n\n" +
				"Lists the SHA-256 TLS certificate fingerprints authorized to log into\n" +
				"your account (or the given account)."},
			{Name: "ADD", Desc: "Add a new fingerprint\n" +
				"Syntax: CERT ADD <fingerprint>\n\n" +
				"Adds a new SHA-256 TLS certificate fingerprint to your account."},
			{Name: "DEL", Desc: "Remove a fingerprint\n" +
				"Syntax: CERT DEL <fingerprint>\n\n" +
				"Removes a SHA-256 TLS certificate fingerprint from your account."},
		}},
		{Name: "CLIENTS", Desc: "List or logout sessions on your account\n" +
			"Syntax: CLIENTS LIST [nickname]\n" +
			"        CLIENTS LOGOUT [nickname] [client_id/all]\n\n" +
			"CLIENTS LIST shows information about the clients currently attached, via\n" +
			"the server's multiclient functionality, to your nickname. An administrator\n" +
			"can use this command to list another user's clients.\n\n" +
			"CLIENTS LOGOUT detaches a single client, or all clients currently attached\n" +
			"to your nickname.", Subcommands: []Command{
			{Name: "LIST", Desc: "List attached clients\n" +
				"Syntax: CLIENTS LIST [nickname]\n\n" +
				"Shows information about clients currently attached to your nickname via\n" +
				"the server's multiclient functionality. An administrator can list another\n" +
				"user's clients.", Args: []ArgType{ArgNick}},
			{Name: "LOGOUT", Desc: "Detach a client session\n" +
				"Syntax: CLIENTS LOGOUT [nickname] [client_id/all]\n\n" +
				"Detaches a single client, or all clients currently attached to your\n" +
				"nickname.", Args: []ArgType{ArgNick}},
		}},
		{Name: "SUSPEND", Desc: "Manage account suspensions\n" +
			"Syntax: SUSPEND ADD <nickname> [DURATION duration] [reason]\n" +
			"        SUSPEND DEL <nickname>\n" +
			"        SUSPEND LIST\n\n" +
			"Suspending an account disables it (preventing new logins) and disconnects\n" +
			"all associated clients. You can specify a time limit or a reason for\n" +
			"the suspension. The DEL subcommand reverses a suspension, and the LIST\n" +
			"command lists all current suspensions.", Subcommands: []Command{
			{Name: "ADD", Desc: "Suspend an account\n" +
				"Syntax: SUSPEND ADD <nickname> [DURATION duration] [reason]\n\n" +
				"Suspends the given account, disabling logins and disconnecting all\n" +
				"associated clients. You can specify a time limit or a reason.", Args: []ArgType{ArgNick}},
			{Name: "DEL", Desc: "Remove a suspension\n" +
				"Syntax: SUSPEND DEL <nickname>\n\n" +
				"Reverses a suspension, allowing the account to log in again.", Args: []ArgType{ArgNick}},
			{Name: "LIST", Desc: "List current suspensions\n" +
				"Syntax: SUSPEND LIST\n\n" +
				"Lists all currently suspended accounts."},
		}},
		{Name: "RENAME", Desc: "Rename an account\n" +
			"Syntax: RENAME <account> <newname>\n\n" +
			"RENAME allows a server administrator to change the name of an account.\n" +
			"Currently, you can only change the canonical casefolding of an account\n" +
			"(e.g., you can change \"Alice\" to \"alice\", but not \"Alice\" to \"Amanda\")."},
		{
			Name: "SADROP", Desc: "Forcibly de-link a nickname\n" +
				"Syntax: SADROP <nickname>\n\n" +
				"SADROP forcibly de-links the given nickname from the attached user account.",
			Args: []ArgType{ArgNick},
		},
		{Name: "SAREGISTER", Desc: "Register an account on someone else's behalf\n" +
			"Syntax: SAREGISTER <username> [password]\n\n" +
			"SAREGISTER registers an account on someone else's behalf.\n" +
			"This is for use in configurations that require SASL for all connections;\n" +
			"an administrator can use this command to set up user accounts."},
		{Name: "SAVERIFY", Desc: "Manually verify a pending account\n" +
			"Syntax: SAVERIFY <username>\n\n" +
			"SAVERIFY manually verifies an account that is pending verification."},
		{Name: "ERASE", Desc: "Erase all records of an account\n" +
			"Syntax: ERASE <username> [code]\n\n" +
			"ERASE deletes all records of an account, allowing it to be re-registered.\n" +
			"This should be used with caution, because it violates an expectation that\n" +
			"account names are permanent identifiers. Typically, UNREGISTER should be\n" +
			"used instead. A confirmation code is required; invoking the command\n" +
			"without a code will display the necessary code."},
		{Name: "PUSH", Desc: "View or modify push subscriptions\n" +
			"Syntax: PUSH LIST\n" +
			"Or:     PUSH DELETE <endpoint>\n\n" +
			"PUSH lets you view or modify the state of your push subscriptions.", Subcommands: []Command{
			{Name: "LIST", Desc: "List push subscriptions\n" +
				"Syntax: PUSH LIST\n\n" +
				"Lists your current push subscriptions."},
			{Name: "DELETE", Desc: "Delete a push subscription\n" +
				"Syntax: PUSH DELETE <endpoint>\n\n" +
				"Deletes a push subscription by endpoint."},
		}},
		{Name: "SAGET", Desc: "Query another user's account settings\n" +
			"Syntax: SAGET <account> <setting>\n\n" +
			"SAGET queries the values of someone else's account settings. For more\n" +
			"information on the settings and their possible values, see HELP SET."},
		{Name: "SASET", Desc: "Modify another user's account settings\n" +
			"Syntax: SASET <account> <setting> <value>\n\n" +
			"SASET modifies the values of someone else's account settings. For more\n" +
			"information on the settings and their possible values, see HELP SET."},
		{Name: "SENDPASS", Desc: "Initiate email-based password reset\n" +
			"Syntax: SENDPASS <account>\n\n" +
			"SENDPASS sends a password reset email to the email address associated with\n" +
			"the target account. The reset code in the email can then be used with the\n" +
			"RESETPASS command."},
		{Name: "RESETPASS", Desc: "Complete email-based password reset\n" +
			"Syntax: RESETPASS <account> <code> <password>\n\n" +
			"RESETPASS resets an account password, using a reset code that was emailed as\n" +
			"the result of a previous SENDPASS command."},
	},
	"CHANSERV": {
		{
			Name: "OP", Desc: "Grant channel operator status\n" +
				"Syntax: OP #channel [nickname]\n\n" +
				"OP makes the given nickname, or yourself, a channel admin. You can only use\n" +
				"this command if you're a founder or in the AMODEs of the channel.",
			Args: []ArgType{ArgChannel, ArgNick},
		},
		{
			Name: "DEOP", Desc: "Remove channel operator status\n" +
				"Syntax: DEOP #channel [nickname]\n\n" +
				"DEOP removes the given nickname, or yourself, the channel admin. You can only\n" +
				"use this command if you're the founder of the channel.",
			Args: []ArgType{ArgChannel, ArgNick},
		},
		{
			Name: "REGISTER", Desc: "Register channel ownership\n" +
				"Syntax: REGISTER #channel\n\n" +
				"REGISTER lets you own the given channel. If you rejoin this channel, you'll be\n" +
				"given admin privs on it. Modes set on the channel and the topic will also be\n" +
				"remembered.",
			Args: []ArgType{ArgChannel},
		},
		{
			Name: "UNREGISTER", Aliases: []string{"DROP"}, Desc: "Delete channel registration\n" +
				"Syntax: UNREGISTER #channel [code]\n\n" +
				"UNREGISTER deletes a channel registration, allowing someone else to claim it.\n" +
				"To prevent accidental unregistrations, a verification code is required;\n" +
				"invoking the command without a code will display the necessary code.",
			Args: []ArgType{ArgChannel},
		},
		{
			Name: "INFO", Desc: "Display channel registration info\n" +
				"Syntax: INFO #channel\n\n" +
				"INFO displays info about a registered channel.",
			Args: []ArgType{ArgChannel},
		},
		{
			Name: "GET", Desc: "Query channel settings\n" +
				"Syntax: GET #channel <setting>\n\n" +
				"GET queries the current values of the channel settings. For more information\n" +
				"on the settings and their possible values, see HELP SET.",
			Args: []ArgType{ArgChannel},
		},
		{
			Name: "SET", Desc: "Modify channel settings\n" +
				"Syntax: SET #channel <setting> <value>\n\n" +
				"SET modifies a channel's settings. Available settings: HISTORY,\n" +
				"QUERY-CUTOFF. Use HELP SET for details on each setting.",
			Args: []ArgType{ArgChannel},
		},
		{
			Name: "AMODE", Desc: "Persistent mode settings for members\n" +
				"Syntax: AMODE #channel [mode change] [account]\n\n" +
				"AMODE lists or modifies persistent mode settings that affect channel members.\n" +
				"For example, AMODE #channel +o dan grants the holder of the \"dan\" account the\n" +
				"+o operator mode every time they join #channel. To list current accounts and\n" +
				"modes, use AMODE #channel. Note that users are always referenced by their\n" +
				"registered account names, not their nicknames.",
			Args: []ArgType{ArgChannel},
		},
		{
			Name: "CLEAR", Desc: "Remove users or settings from channel\n" +
				"Syntax: CLEAR #channel target\n\n" +
				"CLEAR removes users or settings from a channel. Specifically:\n" +
				"CLEAR #channel users — kicks all users except for you.\n" +
				"CLEAR #channel access — resets all stored bans, invites, ban exceptions,\n" +
				"and persistent user-mode grants made with CS AMODE.",
			Args: []ArgType{ArgChannel},
		},
		{
			Name: "TRANSFER", Desc: "Transfer channel ownership\n" +
				"Syntax: TRANSFER [accept] #channel user [code]\n\n" +
				"TRANSFER transfers ownership of a channel from one user to another.\n" +
				"To prevent accidental transfers, a verification code is required. For\n" +
				"example, TRANSFER #channel alice displays the required confirmation\n" +
				"code, then TRANSFER #channel alice 2930242125 initiates the transfer.\n" +
				"Unless you are an IRC operator with the correct permissions, alice must\n" +
				"then accept the transfer with TRANSFER accept #channel.",
			Args: []ArgType{ArgChannel},
		},
		{Name: "PURGE", Desc: "Blacklist a channel from the server\n" +
			"Syntax: PURGE <ADD | DEL | LIST> #channel [code] [reason]\n\n" +
			"PURGE ADD blacklists a channel from the server, making it impossible to join\n" +
			"or otherwise interact with the channel. If the channel currently has members,\n" +
			"they will be kicked from it. PURGE may also be applied preemptively to\n" +
			"channels that do not currently have members. A purge can be undone with\n" +
			"PURGE DEL. To list purged channels, use PURGE LIST.", Subcommands: []Command{
			{Name: "ADD", Desc: "Blacklist a channel\n" +
				"Syntax: PURGE ADD #channel [code] [reason]\n\n" +
				"Blacklists a channel from the server, making it impossible to join or\n" +
				"otherwise interact with the channel. If the channel currently has members,\n" +
				"they will be kicked from it.", Args: []ArgType{ArgChannel}},
			{Name: "DEL", Desc: "Remove a channel blacklist\n" +
				"Syntax: PURGE DEL #channel\n\n" +
				"Removes a purge, allowing the channel to be used again.", Args: []ArgType{ArgChannel}},
			{Name: "LIST", Desc: "List purged channels\n" +
				"Syntax: PURGE LIST\n\n" +
				"Lists all currently purged channels."},
		}},
		{Name: "LIST", Desc: "Search registered channels\n" +
			"Syntax: LIST [regex]\n\n" +
			"LIST returns the list of registered channels, which match the given regex.\n" +
			"If no regex is provided, all registered channels are returned."},
		{
			Name: "HOWTOBAN", Desc: "Suggest best way to ban a user\n" +
				"Syntax: HOWTOBAN #channel <nick>\n\n" +
				"The best way to ban a user from a channel will depend on how they are\n" +
				"connected to the server. HOWTOBAN suggests a ban command that will\n" +
				"(ideally) prevent the user from returning to the channel.",
			Args: []ArgType{ArgChannel, ArgNick},
		},
	},
	"HOSTSERV": {
		{Name: "ON", Desc: "Enable your approved vhost\n" +
			"Syntax: ON\n\n" +
			"ON enables your vhost, if you have one approved."},
		{Name: "OFF", Desc: "Disable your vhost\n" +
			"Syntax: OFF\n\n" +
			"OFF disables your vhost, if you have one approved."},
		{Name: "STATUS", Desc: "Show your current vhost status\n" +
			"Syntax: STATUS [user]\n\n" +
			"STATUS displays your current vhost, if any, and whether it is enabled or\n" +
			"disabled. A server operator can view someone else's status."},
		{Name: "SET", Desc: "Set a user's vhost\n" +
			"Syntax: SET <user> <vhost>\n\n" +
			"SET sets a user's vhost."},
		{Name: "DEL", Desc: "Delete a user's vhost\n" +
			"Syntax: DEL <user>\n\n" +
			"DEL deletes a user's vhost."},
		{Name: "SETCLOAKSECRET", Desc: "Modify the IP cloaking secret\n" +
			"Syntax: SETCLOAKSECRET <secret> [code]\n\n" +
			"SETCLOAKSECRET can be used to set or rotate the cloak secret. You should use\n" +
			"a cryptographically strong secret. To prevent accidental modification, a\n" +
			"verification code is required; invoking the command without a code will\n" +
			"display the necessary code."},
	},
	"HISTSERV": {
		{
			Name: "PLAY", Desc: "Replay historical messages as notices\n" +
				"Syntax: PLAY <target> [limit]\n\n" +
				"PLAY plays back history messages, rendering them into direct messages from\n" +
				"HistServ. 'target' is a channel name or nickname to query, and 'limit'\n" +
				"is a message count or a time duration. Note that message playback may be\n" +
				"incomplete or degraded, relative to direct playback from /HISTORY or\n" +
				"CHATHISTORY.",
			Args: []ArgType{ArgTarget},
		},
		{
			Name: "DELETE", Desc: "Delete a message by target and msgid\n" +
				"Syntax: DELETE <target> <msgid>\n\n" +
				"DELETE deletes an individual message by its msgid. The target is the channel\n" +
				"name. The msgid is the ID as can be found in the tags of that message.",
			Args: []ArgType{ArgTarget},
		},
		{
			Name: "EXPORT", Desc: "Export all messages for an account as JSON\n" +
				"Syntax: EXPORT <account>\n\n" +
				"EXPORT exports all messages sent by an account as JSON. This can be used at\n" +
				"the request of the account holder.",
			Args: []ArgType{ArgTarget},
		},
		{Name: "FORGET", Desc: "Delete all history messages from an account\n" +
			"Syntax: FORGET <account>\n\n" +
			"FORGET deletes all history messages sent by an account."},
	},
}

// serviceAliases maps short names to their canonical service name.
var serviceAliases = map[string]string{
	"NS": "NICKSERV",
	"CS": "CHANSERV",
}

// serviceNicks maps canonical service names to the IRC nick that sends responses.
var serviceNicks = map[string]string{
	"NICKSERV": "NickServ",
	"CHANSERV": "ChanServ",
	"HOSTSERV": "HostServ",
	"HISTSERV": "HistServ",
}

// serviceNickFor returns the expected IRC nick for a service command in the given raw text.
// Returns "" if the text is not a service command.
func serviceNickFor(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	canonical, ok := isServiceCommand(fields[0])
	if !ok {
		return ""
	}
	return serviceNicks[canonical]
}

// isServiceCommand checks whether name is a service command (or alias).
// Returns the canonical name and true if found.
func isServiceCommand(name string) (string, bool) {
	upper := strings.ToUpper(name)
	if _, ok := serviceSubcommands[upper]; ok {
		return upper, true
	}
	if canonical, ok := serviceAliases[upper]; ok {
		return canonical, true
	}
	return "", false
}

// findServiceSubcommand looks up a subcommand within a service's command list.
func findServiceSubcommand(service, subcmd string) (Command, bool) {
	cmds, ok := serviceSubcommands[service]
	if !ok {
		return Command{}, false
	}
	return findCommand(cmds, subcmd)
}
