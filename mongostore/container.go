package mongostore

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	mathRand "math/rand/v2"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/proto/waAdv"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/util/keys"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ store.DeviceContainer = (*Container)(nil)

// ErrDeviceIDMustBeSet — a device can only be saved after pairing gave it a JID.
var ErrDeviceIDMustBeSet = errors.New("device JID must be known before accessing database")

// Container is the MongoDB counterpart of sqlstore.Container.
type Container struct {
	db  *mongo.Database
	log waLog.Logger
	LID *LIDMap
}

// New creates the container, the indexes and loads the LID map.
func New(ctx context.Context, db *mongo.Database, log waLog.Logger) (*Container, error) {
	if log == nil {
		log = waLog.Noop
	}
	lid, err := newLIDMap(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to load LID map: %w", err)
	}
	c := &Container{db: db, log: log, LID: lid}
	c.ensureIndexes(ctx)
	return c, nil
}

func (c *Container) ensureIndexes(ctx context.Context) {
	idx := map[string][]bson.D{
		"wm_session":   {{{Key: "o", Value: 1}, {Key: "a", Value: 1}}},
		"wm_identity":  {{{Key: "o", Value: 1}, {Key: "a", Value: 1}}},
		"wm_senderkey": {{{Key: "o", Value: 1}, {Key: "a", Value: 1}}},
		"wm_prekey":    {{{Key: "o", Value: 1}, {Key: "kid", Value: -1}}, {{Key: "o", Value: 1}, {Key: "up", Value: 1}}},
		"wm_appkey":    {{{Key: "o", Value: 1}, {Key: "ts", Value: -1}}},
		"wm_contact":   {{{Key: "o", Value: 1}}},
		"wm_evbuf":     {{{Key: "o", Value: 1}, {Key: "it", Value: 1}}},
		"wm_outevt":    {{{Key: "o", Value: 1}, {Key: "ts", Value: 1}}},
		"wm_privtoken": {{{Key: "o", Value: 1}, {Key: "ts", Value: 1}}},
	}
	for col, list := range idx {
		for _, k := range list {
			_, _ = c.db.Collection(col).Indexes().CreateOne(ctx, mongo.IndexModel{Keys: k})
		}
	}
}

type deviceDoc struct {
	JID        string `bson:"_id"`
	LID        string `bson:"lid"`
	RegID      int64  `bson:"reg"`
	Noise      []byte `bson:"noise"`
	Identity   []byte `bson:"ident"`
	SPK        []byte `bson:"spk"`
	SPKID      int64  `bson:"spkid"`
	SPKSig     []byte `bson:"spksig"`
	AdvKey     []byte `bson:"advkey"`
	AdvDetails []byte `bson:"advdet"`
	AdvAccSig  []byte `bson:"advacc"`
	AdvAccKey  []byte `bson:"advacckey"`
	AdvDevSig  []byte `bson:"advdev"`
	Platform   string `bson:"platform"`
	Business   string `bson:"biz"`
	Push       string `bson:"push"`
	FBUUID     string `bson:"fb"`
	LIDMigTS   int64  `bson:"lidts"`
	Nonce      string `bson:"nonce"`
}

func (c *Container) toDevice(d deviceDoc) (*store.Device, error) {
	if len(d.Noise) != 32 || len(d.Identity) != 32 || len(d.SPK) != 32 || len(d.SPKSig) != 64 {
		return nil, ErrInvalidLength
	}
	jid, err := types.ParseJID(d.JID)
	if err != nil {
		return nil, fmt.Errorf("bad device jid %q: %w", d.JID, err)
	}
	dev := &store.Device{Log: c.log}
	dev.ID = &jid
	if d.LID != "" {
		if lid, err := types.ParseJID(d.LID); err == nil {
			dev.LID = lid
		}
	}
	dev.RegistrationID = uint32(d.RegID)
	dev.NoiseKey = keys.NewKeyPairFromPrivateKey(*(*[32]byte)(d.Noise))
	dev.IdentityKey = keys.NewKeyPairFromPrivateKey(*(*[32]byte)(d.Identity))
	dev.SignedPreKey = &keys.PreKey{
		KeyPair:   *keys.NewKeyPairFromPrivateKey(*(*[32]byte)(d.SPK)),
		KeyID:     uint32(d.SPKID),
		Signature: (*[64]byte)(d.SPKSig),
	}
	dev.AdvSecretKey = d.AdvKey
	dev.Account = &waAdv.ADVSignedDeviceIdentity{
		Details:             d.AdvDetails,
		AccountSignature:    d.AdvAccSig,
		AccountSignatureKey: d.AdvAccKey,
		DeviceSignature:     d.AdvDevSig,
	}
	dev.Platform = d.Platform
	dev.BusinessName = d.Business
	dev.PushName = d.Push
	if d.FBUUID != "" {
		if u, err := uuid.Parse(d.FBUUID); err == nil {
			dev.FacebookUUID = u
		}
	}
	dev.LIDMigrationTimestamp = d.LIDMigTS
	dev.CompanionMetaNonce = d.Nonce
	c.initializeDevice(dev)
	return dev, nil
}

// GetAllDevices returns every stored device.
func (c *Container) GetAllDevices(ctx context.Context) ([]*store.Device, error) {
	cur, err := c.db.Collection("wm_device").Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []*store.Device
	for cur.Next(ctx) {
		var d deviceDoc
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		dev, err := c.toDevice(d)
		if err != nil {
			return nil, err
		}
		out = append(out, dev)
	}
	return out, cur.Err()
}

// GetFirstDevice returns the stored device or a blank unsaved one.
func (c *Container) GetFirstDevice(ctx context.Context) (*store.Device, error) {
	devs, err := c.GetAllDevices(ctx)
	if err != nil {
		return nil, err
	}
	if len(devs) == 0 {
		return c.NewDevice(), nil
	}
	return devs[0], nil
}

// GetDevice returns the device with that JID (nil if unknown).
func (c *Container) GetDevice(ctx context.Context, jid types.JID) (*store.Device, error) {
	var d deviceDoc
	err := c.db.Collection("wm_device").FindOne(ctx, bson.M{"_id": jid.String()}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c.toDevice(d)
}

// NewDevice makes a fresh, unpaired device with new keys.
func (c *Container) NewDevice() *store.Device {
	adv := make([]byte, 32)
	_, _ = rand.Read(adv)
	dev := &store.Device{
		Log:            c.log,
		Container:      c,
		NoiseKey:       keys.NewKeyPair(),
		IdentityKey:    keys.NewKeyPair(),
		RegistrationID: mathRand.Uint32(),
		AdvSecretKey:   adv,
	}
	dev.SignedPreKey = dev.IdentityKey.CreateSignedPreKey(1)
	return dev
}

func (c *Container) initializeDevice(dev *store.Device) {
	dev.SetAllStores(newStore(c.db, *dev.ID, c.LID, c.log))
	dev.LIDs = c.LID
	dev.Container = c
	dev.Initialized = true
}

// PutDevice saves the device (called by whatsmeow after pairing and on changes).
func (c *Container) PutDevice(ctx context.Context, dev *store.Device) error {
	if dev.ID == nil {
		return ErrDeviceIDMustBeSet
	}
	acc := dev.Account
	if acc == nil {
		acc = &waAdv.ADVSignedDeviceIdentity{}
	}
	fb := ""
	if dev.FacebookUUID != uuid.Nil {
		fb = dev.FacebookUUID.String()
	}
	lid := ""
	if !dev.LID.IsEmpty() {
		lid = dev.LID.String()
	}
	set := bson.M{
		"lid": lid, "reg": int64(dev.RegistrationID),
		"noise": dev.NoiseKey.Priv[:], "ident": dev.IdentityKey.Priv[:],
		"spk": dev.SignedPreKey.Priv[:], "spkid": int64(dev.SignedPreKey.KeyID), "spksig": dev.SignedPreKey.Signature[:],
		"advkey": dev.AdvSecretKey, "advdet": acc.Details, "advacc": acc.AccountSignature,
		"advacckey": acc.AccountSignatureKey, "advdev": acc.DeviceSignature,
		"platform": dev.Platform, "biz": dev.BusinessName, "push": dev.PushName,
		"fb": fb, "lidts": dev.LIDMigrationTimestamp, "nonce": dev.CompanionMetaNonce,
	}
	_, err := c.db.Collection("wm_device").UpdateOne(ctx,
		bson.M{"_id": dev.ID.String()}, bson.M{"$set": set}, options.Update().SetUpsert(true))
	if err != nil {
		return err
	}
	if !dev.Initialized {
		c.initializeDevice(dev)
	}
	return nil
}

// DeleteDevice removes the device and ALL of its data.
func (c *Container) DeleteDevice(ctx context.Context, dev *store.Device) error {
	if dev.ID == nil {
		return ErrDeviceIDMustBeSet
	}
	jid := dev.ID.String()
	if _, err := c.db.Collection("wm_device").DeleteOne(ctx, bson.M{"_id": jid}); err != nil {
		return err
	}
	for _, name := range []string{"identity", "session", "prekey", "senderkey", "appkey", "appver",
		"appmac", "contact", "chat", "secret", "privtoken", "nct", "evbuf", "outevt"} {
		_, _ = c.db.Collection("wm_"+name).DeleteMany(ctx, bson.M{"o": jid})
	}
	return nil
}
