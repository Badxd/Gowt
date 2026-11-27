package modules

// chatlock.go — stub: command removed, IsChatLocked retained for module.go gate
// .chatlock and .messagelock commands removed (not supported in this version)

// IsChatLocked always returns false (feature disabled)
func IsChatLocked(_ string) bool {
	return false
}
