package mongostore

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ── Contacts ────────────────────────────────────────────────────────────────

type contactDoc struct {
	JID   string `bson:"j"`
	First string `bson:"fn"`
	Full  string `bson:"full"`
	Push  string `bson:"push"`
	Biz   string `bson:"biz"`
	Red   string `bson:"red"`
}

func (d contactDoc) info() *types.ContactInfo {
	return &types.ContactInfo{
		Found:         true,
		FirstName:     d.First,
		FullName:      d.Full,
		PushName:      d.Push,
		BusinessName:  d.Biz,
		RedactedPhone: d.Red,
	}
}

// getContact must be called with contactLock held.
func (s *Store) getContact(ctx context.Context, user types.JID) (*types.ContactInfo, error) {
	if cached, ok := s.contactCache[user]; ok {
		return cached, nil
	}
	var d contactDoc
	ok, err := findOne(ctx, s.c("contact"), s.key(user.String()), &d)
	if err != nil {
		return nil, err
	}
	info := &types.ContactInfo{}
	if ok {
		info = d.info()
	}
	s.contactCache[user] = info
	return info, nil
}

func (s *Store) putContactField(ctx context.Context, user types.JID, set bson.M) error {
	set["o"] = s.jid
	set["j"] = user.String()
	return upsert(ctx, s.c("contact"), s.key(user.String()), set)
}

func (s *Store) PutPushName(ctx context.Context, user types.JID, pushName string) (bool, string, error) {
	s.contactLock.Lock()
	defer s.contactLock.Unlock()
	cached, err := s.getContact(ctx, user)
	if err != nil {
		return false, "", err
	}
	if cached.PushName == pushName {
		return false, "", nil
	}
	if err := s.putContactField(ctx, user, bson.M{"push": pushName}); err != nil {
		return false, "", err
	}
	prev := cached.PushName
	cached.PushName = pushName
	cached.Found = true
	return true, prev, nil
}

func (s *Store) PutBusinessName(ctx context.Context, user types.JID, businessName string) (bool, string, error) {
	s.contactLock.Lock()
	defer s.contactLock.Unlock()
	cached, err := s.getContact(ctx, user)
	if err != nil {
		return false, "", err
	}
	if cached.BusinessName == businessName {
		return false, "", nil
	}
	if err := s.putContactField(ctx, user, bson.M{"biz": businessName}); err != nil {
		return false, "", err
	}
	prev := cached.BusinessName
	cached.BusinessName = businessName
	cached.Found = true
	return true, prev, nil
}

// PutContactName — argument order matches how whatsmeow calls it (first, full).
func (s *Store) PutContactName(ctx context.Context, user types.JID, firstName, fullName string) error {
	s.contactLock.Lock()
	defer s.contactLock.Unlock()
	cached, err := s.getContact(ctx, user)
	if err != nil {
		return err
	}
	if cached.FirstName == firstName && cached.FullName == fullName {
		return nil
	}
	if err := s.putContactField(ctx, user, bson.M{"fn": firstName, "full": fullName}); err != nil {
		return err
	}
	cached.FirstName = firstName
	cached.FullName = fullName
	cached.Found = true
	return nil
}

func (s *Store) PutAllContactNames(ctx context.Context, contacts []store.ContactEntry) error {
	if len(contacts) == 0 {
		return nil
	}
	models := make([]mongo.WriteModel, 0, len(contacts))
	for _, c := range contacts {
		models = append(models, upModel(s.key(c.JID.String()), bson.M{
			"o": s.jid, "j": c.JID.String(), "fn": c.FirstName, "full": c.FullName,
		}))
	}
	if err := bulk(ctx, s.c("contact"), models); err != nil {
		return err
	}
	s.contactLock.Lock()
	s.contactCache = map[types.JID]*types.ContactInfo{}
	s.contactLock.Unlock()
	return nil
}

func (s *Store) PutManyRedactedPhones(ctx context.Context, entries []store.RedactedPhoneEntry) error {
	if len(entries) == 0 {
		return nil
	}
	models := make([]mongo.WriteModel, 0, len(entries))
	for _, e := range entries {
		models = append(models, upModel(s.key(e.JID.String()), bson.M{
			"o": s.jid, "j": e.JID.String(), "red": e.RedactedPhone,
		}))
	}
	if err := bulk(ctx, s.c("contact"), models); err != nil {
		return err
	}
	s.contactLock.Lock()
	for _, e := range entries {
		delete(s.contactCache, e.JID)
	}
	s.contactLock.Unlock()
	return nil
}

func (s *Store) GetContact(ctx context.Context, user types.JID) (types.ContactInfo, error) {
	s.contactLock.Lock()
	defer s.contactLock.Unlock()
	info, err := s.getContact(ctx, user)
	if err != nil {
		return types.ContactInfo{}, err
	}
	return *info, nil
}

func (s *Store) GetAllContacts(ctx context.Context) (map[types.JID]types.ContactInfo, error) {
	s.contactLock.Lock()
	defer s.contactLock.Unlock()
	cur, err := s.c("contact").Find(ctx, bson.M{"o": s.jid})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := map[types.JID]types.ContactInfo{}
	for cur.Next(ctx) {
		var d contactDoc
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		jid, err := types.ParseJID(d.JID)
		if err != nil {
			continue
		}
		info := d.info()
		out[jid] = *info
		s.contactCache[jid] = info
	}
	return out, cur.Err()
}

// ── Message secrets ─────────────────────────────────────────────────────────

type secretDoc struct {
	Secret []byte `bson:"v"`
	Sender string `bson:"s"`
}

func (s *Store) secretID(chat, sender types.JID, id types.MessageID) string {
	return s.key(chat.ToNonAD().String(), sender.ToNonAD().String(), string(id))
}

func (s *Store) PutMessageSecrets(ctx context.Context, inserts []store.MessageSecretInsert) error {
	for _, in := range inserts {
		if in.Chat.IsEmpty() || in.Sender.IsEmpty() || in.ID == "" || len(in.Secret) == 0 {
			continue
		}
		if err := s.PutMessageSecret(ctx, in.Chat, in.Sender, in.ID, in.Secret); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) PutMessageSecret(ctx context.Context, chat, sender types.JID, id types.MessageID, secret []byte) error {
	// "do nothing on conflict": only set when the document does not exist yet
	_, err := s.c("secret").UpdateOne(ctx,
		bson.M{"_id": s.secretID(chat, sender, id)},
		bson.M{"$setOnInsert": bson.M{"o": s.jid, "v": secret, "s": sender.ToNonAD().String()}},
		upsertOpt(),
	)
	return err
}

func (s *Store) altJIDs(ctx context.Context, jid types.JID) []types.JID {
	jid = jid.ToNonAD()
	out := []types.JID{jid}
	if s.lid == nil {
		return out
	}
	switch jid.Server {
	case types.HiddenUserServer:
		if pn, err := s.lid.GetPNForLID(ctx, jid); err == nil && !pn.IsEmpty() {
			out = append(out, pn.ToNonAD())
		}
	case types.DefaultUserServer:
		if lid, err := s.lid.GetLIDForPN(ctx, jid); err == nil && !lid.IsEmpty() {
			out = append(out, lid.ToNonAD())
		}
	}
	return out
}

func (s *Store) GetMessageSecret(ctx context.Context, chat, sender types.JID, id types.MessageID) ([]byte, types.JID, error) {
	for _, c := range s.altJIDs(ctx, chat) {
		for _, snd := range s.altJIDs(ctx, sender) {
			var d secretDoc
			ok, err := findOne(ctx, s.c("secret"), s.key(c.String(), snd.String(), string(id)), &d)
			if err != nil {
				return nil, types.JID{}, err
			}
			if ok {
				real, _ := types.ParseJID(d.Sender)
				return d.Secret, real, nil
			}
		}
	}
	return nil, types.JID{}, nil
}

// ── Privacy tokens / NCT salt ───────────────────────────────────────────────

type tokenDoc struct {
	Token []byte `bson:"v"`
	TS    int64  `bson:"ts"`
	STS   int64  `bson:"sts"`
}

func (s *Store) PutPrivacyTokens(ctx context.Context, tokens ...store.PrivacyToken) error {
	for _, t := range tokens {
		user := t.User.ToNonAD()
		id := s.key(user.String())
		var existing tokenDoc
		ok, err := findOne(ctx, s.c("privtoken"), id, &existing)
		if err != nil {
			return err
		}
		if ok && t.Timestamp.Unix() < existing.TS {
			continue
		}
		sts := existing.STS
		if !t.SenderTimestamp.IsZero() {
			sts = t.SenderTimestamp.Unix()
		}
		if err := upsert(ctx, s.c("privtoken"), id, bson.M{
			"o": s.jid, "v": t.Token, "ts": t.Timestamp.Unix(), "sts": sts,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetPrivacyToken(ctx context.Context, user types.JID) (*store.PrivacyToken, error) {
	var best *store.PrivacyToken
	for _, j := range s.altJIDs(ctx, user) {
		var d tokenDoc
		ok, err := findOne(ctx, s.c("privtoken"), s.key(j.String()), &d)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if best == nil || time.Unix(d.TS, 0).After(best.Timestamp) {
			t := &store.PrivacyToken{User: user.ToNonAD(), Token: d.Token, Timestamp: time.Unix(d.TS, 0)}
			if d.STS > 0 {
				t.SenderTimestamp = time.Unix(d.STS, 0)
			}
			best = t
		}
	}
	return best, nil
}

func (s *Store) DeleteExpiredPrivacyTokens(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.c("privtoken").DeleteMany(ctx, bson.M{"o": s.jid, "ts": bson.M{"$lt": cutoff.Unix()}})
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}

func (s *Store) PutNCTSalt(ctx context.Context, salt []byte) error {
	return upsert(ctx, s.c("nct"), s.key("nct"), bson.M{"o": s.jid, "v": salt})
}

func (s *Store) GetNCTSalt(ctx context.Context) ([]byte, error) {
	var d bin
	ok, err := findOne(ctx, s.c("nct"), s.key("nct"), &d)
	if err != nil || !ok {
		return nil, err
	}
	return d.V, nil
}

func (s *Store) DeleteNCTSalt(ctx context.Context) error {
	_, err := s.c("nct").DeleteOne(ctx, bson.M{"_id": s.key("nct")})
	return err
}

// ── Event buffer / outgoing events ──────────────────────────────────────────

func (s *Store) GetBufferedEvent(ctx context.Context, ciphertextHash [32]byte) (*store.BufferedEvent, error) {
	var d struct {
		PT []byte `bson:"pt"`
		ST int64  `bson:"st"`
		IT int64  `bson:"it"`
	}
	ok, err := findOne(ctx, s.c("evbuf"), s.key(hex.EncodeToString(ciphertextHash[:])), &d)
	if err != nil || !ok {
		return nil, err
	}
	return &store.BufferedEvent{
		Plaintext:  d.PT,
		ServerTime: time.Unix(d.ST, 0),
		InsertTime: time.UnixMilli(d.IT),
	}, nil
}

func (s *Store) PutBufferedEvent(ctx context.Context, ciphertextHash [32]byte, plaintext []byte, serverTimestamp time.Time) error {
	return upsert(ctx, s.c("evbuf"), s.key(hex.EncodeToString(ciphertextHash[:])), bson.M{
		"o": s.jid, "pt": plaintext, "st": serverTimestamp.Unix(), "it": time.Now().UnixMilli(),
	})
}

// DoDecryptionTxn: MongoDB standalone has no multi-document transactions, so the
// callback just runs directly (every store call is individually atomic).
func (s *Store) DoDecryptionTxn(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (s *Store) ClearBufferedEventPlaintext(ctx context.Context, ciphertextHash [32]byte) error {
	_, err := s.c("evbuf").UpdateOne(ctx,
		bson.M{"_id": s.key(hex.EncodeToString(ciphertextHash[:]))},
		bson.M{"$set": bson.M{"pt": nil}})
	return err
}

func (s *Store) DeleteOldBufferedHashes(ctx context.Context) error {
	// WhatsApp only buffers events for 14 days
	_, err := s.c("evbuf").DeleteMany(ctx, bson.M{
		"o": s.jid, "it": bson.M{"$lt": time.Now().Add(-14 * 24 * time.Hour).UnixMilli()},
	})
	return err
}

func (s *Store) GetOutgoingEvent(ctx context.Context, chatJID, altChatJID types.JID, id types.MessageID) (string, []byte, error) {
	for _, c := range []types.JID{chatJID, altChatJID} {
		if c.IsEmpty() {
			continue
		}
		var d struct {
			Format string `bson:"f"`
			PT     []byte `bson:"pt"`
		}
		ok, err := findOne(ctx, s.c("outevt"), s.key(c.String(), string(id)), &d)
		if err != nil {
			return "", nil, err
		}
		if ok {
			return d.Format, d.PT, nil
		}
	}
	return "", nil, mongo.ErrNoDocuments
}

func (s *Store) AddOutgoingEvent(ctx context.Context, chatJID types.JID, id types.MessageID, format string, plaintext []byte) error {
	return upsert(ctx, s.c("outevt"), s.key(chatJID.String(), string(id)), bson.M{
		"o": s.jid, "f": format, "pt": plaintext, "ts": time.Now().UnixMilli(),
	})
}

func (s *Store) DeleteOldOutgoingEvents(ctx context.Context) error {
	_, err := s.c("outevt").DeleteMany(ctx, bson.M{
		"o": s.jid, "ts": bson.M{"$lt": time.Now().Add(-7 * 24 * time.Hour).UnixMilli()},
	})
	return err
}

var _ = errors.New
