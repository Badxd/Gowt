// Package shortsession lets you export/import just the essential WhatsApp
// device identity (noise key, identity key, signed prekey, registration id,
// adv secret, account signature, JID) instead of the entire whatsmeow SQLite
// database.
//
// The full SQLite dump keeps growing forever because it also stores every
// Signal session with every chat, prekeys, sender keys, app-state sync keys,
// etc. This file's blob only holds the fixed-size identity fields, so it
// stays roughly 700-1000 base64 characters no matter how long the bot runs.
// Everything else (sessions, prekeys, app state) is renegotiated
// automatically by whatsmeow / the WhatsApp servers after reconnecting.
package session

import (
	"PbxGb/mongostore"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"go.mau.fi/whatsmeow/proto/waAdv"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/util/keys"
)

// blob is the on-the-wire (JSON) shape of the short session string.
type blob struct {
	ID             string `json:"id"`
	LID            string `json:"lid,omitempty"`
	RegistrationID uint32 `json:"reg"`

	NoiseKeyPriv    string `json:"nk"`
	IdentityKeyPriv string `json:"ik"`

	SignedPreKeyPriv string `json:"spk"`
	SignedPreKeyID   uint32 `json:"spkid"`
	SignedPreKeySig  string `json:"spksig"`

	AdvSecretKey string `json:"adv"`

	AccountDetails string `json:"acd"`
	AccountSigKey  string `json:"acsk"`
	AccountSig     string `json:"acs"`
	DeviceSig      string `json:"ds"`

	Platform     string `json:"pl,omitempty"`
	BusinessName string `json:"bn,omitempty"`
	PushName     string `json:"pn,omitempty"`
}

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func unb64(s string) ([]byte, error) { return base64.StdEncoding.DecodeString(s) }

// Encode takes a logged-in whatsmeow device (e.g. client.Store right after a
// successful pair) and returns a short base64 session string.
func Encode(device *store.Device) (string, error) {
	if device.ID == nil {
		return "", fmt.Errorf("device is not logged in yet (ID is nil)")
	}
	if device.Account == nil {
		return "", fmt.Errorf("device has no account identity yet (Account is nil) — wait a few seconds after pairing and try again")
	}

	b := blob{
		ID:             device.ID.String(),
		RegistrationID: device.RegistrationID,

		NoiseKeyPriv:    b64(device.NoiseKey.Priv[:]),
		IdentityKeyPriv: b64(device.IdentityKey.Priv[:]),

		SignedPreKeyPriv: b64(device.SignedPreKey.Priv[:]),
		SignedPreKeyID:   device.SignedPreKey.KeyID,
		SignedPreKeySig:  b64(device.SignedPreKey.Signature[:]),

		AdvSecretKey: b64(device.AdvSecretKey),

		AccountDetails: b64(device.Account.Details),
		AccountSigKey:  b64(device.Account.AccountSignatureKey),
		AccountSig:     b64(device.Account.AccountSignature),
		DeviceSig:      b64(device.Account.DeviceSignature),

		Platform:     device.Platform,
		BusinessName: device.BusinessName,
		PushName:     device.PushName,
	}
	if !device.LID.IsEmpty() {
		b.LID = device.LID.String()
	}

	raw, err := json.Marshal(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

// Decode restores a device into the given (normally freshly created, empty)
// Mongo store container from a short session string produced by Encode, and
// persists it so the container's usual stores (sessions, prekeys, app
// state, ...) are wired up and ready to use.
func Decode(ctx context.Context, container *mongostore.Container, encoded string) (*store.Device, error) {
	outer, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("outer base64 decode: %w", err)
	}

	var b blob
	if err := json.Unmarshal(outer, &b); err != nil {
		return nil, fmt.Errorf("json decode: %w (is this a short session string, or an old full SESSION dump?)", err)
	}

	jid, err := types.ParseJID(b.ID)
	if err != nil {
		return nil, fmt.Errorf("parse jid: %w", err)
	}

	device := container.NewDevice()
	device.ID = &jid
	device.RegistrationID = b.RegistrationID

	noisePriv, err := unb64(b.NoiseKeyPriv)
	if err != nil {
		return nil, fmt.Errorf("noise key: %w", err)
	}
	identityPriv, err := unb64(b.IdentityKeyPriv)
	if err != nil {
		return nil, fmt.Errorf("identity key: %w", err)
	}
	spkPriv, err := unb64(b.SignedPreKeyPriv)
	if err != nil {
		return nil, fmt.Errorf("signed prekey: %w", err)
	}
	spkSig, err := unb64(b.SignedPreKeySig)
	if err != nil {
		return nil, fmt.Errorf("signed prekey sig: %w", err)
	}
	advSecret, err := unb64(b.AdvSecretKey)
	if err != nil {
		return nil, fmt.Errorf("adv secret: %w", err)
	}
	if len(noisePriv) != 32 || len(identityPriv) != 32 || len(spkPriv) != 32 || len(spkSig) != 64 {
		return nil, fmt.Errorf("corrupted session string (unexpected key length)")
	}

	device.NoiseKey = keys.NewKeyPairFromPrivateKey(*(*[32]byte)(noisePriv))
	device.IdentityKey = keys.NewKeyPairFromPrivateKey(*(*[32]byte)(identityPriv))
	device.SignedPreKey = &keys.PreKey{
		KeyPair:   *keys.NewKeyPairFromPrivateKey(*(*[32]byte)(spkPriv)),
		KeyID:     b.SignedPreKeyID,
		Signature: (*[64]byte)(spkSig),
	}
	device.AdvSecretKey = advSecret

	accDetails, _ := unb64(b.AccountDetails)
	accSigKey, _ := unb64(b.AccountSigKey)
	accSig, _ := unb64(b.AccountSig)
	devSig, _ := unb64(b.DeviceSig)
	device.Account = &waAdv.ADVSignedDeviceIdentity{
		Details:             accDetails,
		AccountSignatureKey: accSigKey,
		AccountSignature:    accSig,
		DeviceSignature:     devSig,
	}

	if b.LID != "" {
		if lid, err := types.ParseJID(b.LID); err == nil {
			device.LID = lid
		}
	}
	device.Platform = b.Platform
	device.BusinessName = b.BusinessName
	device.PushName = b.PushName

	if err := device.Save(ctx); err != nil {
		return nil, fmt.Errorf("save device: %w", err)
	}
	return device, nil
}
