package modules

import (
	"fmt"
	"sort"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

// .help          — list all modules
// .help <module> — list commands in that module
// .cmds          — flat list of all commands

func helpHandler(v *events.Message, args []string) error {
	if len(args) > 0 {
		query := strings.ToLower(strings.Join(args, " "))
		for _, mod := range allModules {
			if strings.ToLower(mod.Name) == query || strings.ToLower(mod.Description) == query {
				return showModuleHelp(v, mod)
			}
		}
		// fuzzy: partial name match
		for _, mod := range allModules {
			if strings.Contains(strings.ToLower(mod.Name), query) {
				return showModuleHelp(v, mod)
			}
		}
		return ReplyErr(v, fmt.Sprintf("❌ No module found matching: *%s*\nTry `.help` for the full list.", query))
	}

	// list all modules
	var lines []string
	lines = append(lines, "📦 *PBX Elite — Module List*")
	lines = append(lines, "")

	// sort by name
	mods := make([]ModuleInfo, len(allModules))
	copy(mods, allModules)
	sort.Slice(mods, func(i, j int) bool {
		return strings.ToLower(mods[i].Name) < strings.ToLower(mods[j].Name)
	})

	for i, mod := range mods {
		lines = append(lines, fmt.Sprintf("%d. *%s* — %s", i+1, mod.Name, mod.Description))
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Total: *%d modules*", len(mods)))
	lines = append(lines, "Use `.help <module name>` for commands.")

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func showModuleHelp(v *events.Message, mod ModuleInfo) error {
	var lines []string
	lines = append(lines, fmt.Sprintf("📦 *%s*", mod.Name))
	lines = append(lines, mod.Description)
	lines = append(lines, "")
	lines = append(lines, "*Commands:*")
	for _, cmd := range mod.Commands {
		sudo := ""
		if cmd.Sudo {
			sudo = " *(sudo)*"
		}
		lines = append(lines, fmt.Sprintf("• `.%s`%s", cmd.Pattern, sudo))
	}
	return ReplyErr(v, strings.Join(lines, "\n"))
}

func cmdsHandler(v *events.Message, args []string) error {
	var cmds []string
	for _, mod := range allModules {
		for _, cmd := range mod.Commands {
			cmds = append(cmds, "."+cmd.Pattern)
		}
	}
	sort.Strings(cmds)

	var lines []string
	lines = append(lines, fmt.Sprintf("⚡ *All Commands (%d)*", len(cmds)))
	lines = append(lines, "")

	// group into columns of 3
	for i := 0; i < len(cmds); i += 3 {
		end := i + 3
		if end > len(cmds) {
			end = len(cmds)
		}
		lines = append(lines, strings.Join(cmds[i:end], "  |  "))
	}

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func init() {
	Register(ModuleInfo{
		Name:        "Help",
		Description: "List all modules and commands",
		Commands: []CommandInfo{
			{Pattern: "help", Handler: helpHandler, Sudo: true},
			{Pattern: "cmds", Handler: cmdsHandler, Sudo: true},
		},
	})
}
