package modules

import (
	"fmt"
	"strings"

	"PbxGb/config"
	"PbxGb/database"

	"go.mau.fi/whatsmeow/types/events"
)

// PBX Elite — Sudo Module

// ADD SUDO
func addSudoHandler(
	v *events.Message,
	args []string,
) error {

	target := resolveTarget(
		v,
		args,
	)

	if target == "" {

		return ReplyErr(
			v,
			"❌ Reply user or use .addsudo 9198xxxxxx",
		)
	}

	if target == config.C.OwnerJID {

		return ReplyErr(
			v,
			"ℹ️ Owner already has full access",
		)
	}

	err := database.AddSudo(
		target,
		SenderJID(v),
	)

	if err != nil {

		return ReplyErr(
			v,
			fmt.Sprintf(
				"❌ Database error: %v",
				err,
			),
		)
	}

	return ReplyErr(
		v,
		fmt.Sprintf(
			"✅ Added sudo: %s",
			target,
		),
	)
}

// REMOVE SUDO
func removeSudoHandler(
	v *events.Message,
	args []string,
) error {

	target := resolveTarget(
		v,
		args,
	)

	if target == "" {

		return ReplyErr(
			v,
			"❌ Reply user or use .removesudo 9198xxxxxx",
		)
	}

	if target == config.C.OwnerJID {

		return ReplyErr(
			v,
			"❌ Cannot remove owner",
		)
	}

	err := database.RemoveSudo(
		target,
	)

	if err != nil {

		return ReplyErr(
			v,
			fmt.Sprintf(
				"❌ Database error: %v",
				err,
			),
		)
	}

	return ReplyErr(
		v,
		fmt.Sprintf(
			"✅ Removed sudo: %s",
			target,
		),
	)
}

// SUDO LIST
func sudoListHandler(
	v *events.Message,
	args []string,
) error {

	users, err := database.ListSudo()

	if err != nil {

		return ReplyErr(
			v,
			fmt.Sprintf(
				"❌ Database error: %v",
				err,
			),
		)
	}

	if len(users) == 0 {

		return ReplyErr(
			v,
			"ℹ️ No sudo users",
		)
	}

	lines := []string{
		"╔══════════════════════╗",
		"║     🛡️ SUDO LIST     ║",
		"╚══════════════════════╝",
		"",
	}

	for i, u := range users {

		lines = append(
			lines,
			fmt.Sprintf(
				"%d. %s\nAddedBy: %s\nDate: %s",
				i+1,
				u.JID,
				u.AddedBy,
				u.AddedAt.Format("02 Jan 2006"),
			),
		)
	}

	return ReplyErr(
		v,
		strings.Join(lines, "\n\n"),
	)
}

// RESOLVE TARGET
func resolveTarget(
	v *events.Message,
	args []string,
) string {

	// REPLY TARGET
	if IsReply(v) {

		ctx := v.Message.
			ExtendedTextMessage.
			ContextInfo

		if ctx != nil &&
			ctx.Participant != nil {

			return *ctx.Participant
		}
	}

	// ARG TARGET
	if len(args) > 0 {

		jid := args[0]

		if !strings.Contains(
			jid,
			"@",
		) {

			jid += "@s.whatsapp.net"
		}

		return jid
	}

	return ""
}

// REGISTER
func init() {

	Register(ModuleInfo{
		Name:        "Sudo",
		Description: "Sudo manager",

		Commands: []CommandInfo{

			{
				Pattern: "addsudo",
				Handler: addSudoHandler,
				Sudo:   true,
			},

			{
				Pattern: "removesudo",
				Handler: removeSudoHandler,
				Sudo:   true,
			},

			{
				Pattern: "sudolist",
				Handler: sudoListHandler,
				Sudo:   true,
			},
		},
	})
}
