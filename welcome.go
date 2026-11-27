package modules

// welcome.go — stub: welcome/goodbye commands removed (not supported in this version)
// HandleGroupJoin and HandleGroupLeave are no-ops to satisfy module.go references.

import "go.mau.fi/whatsmeow/types/events"

func HandleGroupJoin(_ *events.GroupInfo)  {}
func HandleGroupLeave(_ *events.GroupInfo) {}
