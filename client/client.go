package client

import "go.mau.fi/whatsmeow"

var WA *whatsmeow.Client

func Set(c *whatsmeow.Client) { WA = c }
func Get() *whatsmeow.Client  { return WA }
