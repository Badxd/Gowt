package mongostore

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/util/keys"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ store.AllSessionSpecificStores = (*Store)(nil)

// ErrInvalidLength is returned when stored key material has the wrong size.
var ErrInvalidLength = errors.New("database returned byte array with illegal length")

// Store holds every per-device table (all keyed by the device JID).
type Store struct {
	db  *mongo.Database
	jid string
	lid *LIDMap
	log waLog.Logger

	preKeyLock sync.Mutex

	contactLock  sync.Mutex
	contactCache map[types.JID]*types.ContactInfo

	migMu    sync.Mutex
	migrated map[string]bool
}

func newStore(db *mongo.Database, jid types.JID, lid *LIDMap, log waLog.Logger) *Store {
	return &Store{
		db:           db,
		jid:          jid.String(),
		lid:          lid,
		log:          log,
		contactCache: map[types.JID]*types.ContactInfo{},
		migrated:     map[string]bool{},
	}
}

func (s *Store) c(name string) *mongo.Collection { return s.db.Collection("wm_" + name) }
func (s *Store) key(parts ...string) string       { return s.jid + "|" + strings.Join(parts, "|") }

// ── Identity ────────────────────────────────────────────────────────────────

func (s *Store) PutIdentity(ctx context.Context, address string, key [32]byte) error {
	return upsert(ctx, s.c("identity"), s.key(address), bson.M{"o": s.jid, "a": address, "v": key[:]})
}

func (s *Store) DeleteAllIdentities(ctx context.Context, phone string) error {
	_, err := s.c("identity").DeleteMany(ctx, bson.M{"o": s.jid, "a": prefixRegex(phone + ":")})
	return err
}

func (s *Store) DeleteIdentity(ctx context.Context, address string) error {
	_, err := s.c("identity").DeleteOne(ctx, bson.M{"_id": s.key(address)})
	return err
}

func (s *Store) IsTrustedIdentity(ctx context.Context, address string, key [32]byte) (bool, error) {
	var d bin
	ok, err := findOne(ctx, s.c("identity"), s.key(address), &d)
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil // unknown identities are trusted and saved later
	}
	if len(d.V) != 32 {
		return false, ErrInvalidLength
	}
	return *(*[32]byte)(d.V) == key, nil
}

// ── Sessions ────────────────────────────────────────────────────────────────

func (s *Store) GetSession(ctx context.Context, address string) ([]byte, error) {
	var d bin
	ok, err := findOne(ctx, s.c("session"), s.key(address), &d)
	if err != nil || !ok {
		return nil, err
	}
	return d.V, nil
}

func (s *Store) HasSession(ctx context.Context, address string) (bool, error) {
	var d bin
	return findOne(ctx, s.c("session"), s.key(address), &d)
}

func (s *Store) GetManySessions(ctx context.Context, addresses []string) (map[string][]byte, error) {
	out := make(map[string][]byte, len(addresses))
	if len(addresses) == 0 {
		return out, nil
	}
	cur, err := s.c("session").Find(ctx, bson.M{"o": s.jid, "a": bson.M{"$in": addresses}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var d struct {
			A string `bson:"a"`
			V []byte `bson:"v"`
		}
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		out[d.A] = d.V
	}
	return out, cur.Err()
}

func (s *Store) PutSession(ctx context.Context, address string, session []byte) error {
	return upsert(ctx, s.c("session"), s.key(address), bson.M{"o": s.jid, "a": address, "v": session})
}

func (s *Store) PutManySessions(ctx context.Context, sessions map[string][]byte) error {
	models := make([]mongo.WriteModel, 0, len(sessions))
	for addr, sess := range sessions {
		models = append(models, upModel(s.key(addr), bson.M{"o": s.jid, "a": addr, "v": sess}))
	}
	return bulk(ctx, s.c("session"), models)
}

func (s *Store) DeleteAllSessions(ctx context.Context, phone string) error {
	_, err := s.c("session").DeleteMany(ctx, bson.M{"o": s.jid, "a": prefixRegex(phone + ":")})
	return err
}

func (s *Store) DeleteSession(ctx context.Context, address string) error {
	_, err := s.c("session").DeleteOne(ctx, bson.M{"_id": s.key(address)})
	return err
}

// MigratePNToLID moves sessions / identity keys / sender keys from the phone-number
// address to the LID address.
func (s *Store) MigratePNToLID(ctx context.Context, pn, lid types.JID) error {
	pnSignal := pn.SignalAddressUser()
	s.migMu.Lock()
	if s.migrated[pnSignal] {
		s.migMu.Unlock()
		return nil
	}
	s.migrated[pnSignal] = true
	s.migMu.Unlock()

	lidSignal := lid.SignalAddressUser()
	for _, name := range []string{"session", "identity", "senderkey"} {
		col := s.c(name)
		cur, err := col.Find(ctx, bson.M{"o": s.jid, "a": prefixRegex(pnSignal + ":")})
		if err != nil {
			return fmt.Errorf("failed to migrate %s: %w", name, err)
		}
		var models []mongo.WriteModel
		for cur.Next(ctx) {
			var d struct {
				A string `bson:"a"`
				C string `bson:"c"`
				V []byte `bson:"v"`
			}
			if err := cur.Decode(&d); err != nil {
				_ = cur.Close(ctx)
				return err
			}
			newA := lidSignal + strings.TrimPrefix(d.A, pnSignal)
			set := bson.M{"o": s.jid, "a": newA, "v": d.V}
			var id string
			if name == "senderkey" {
				set["c"] = d.C
				id = s.key(d.C, newA)
			} else {
				id = s.key(newA)
			}
			models = append(models, upModel(id, set))
		}
		_ = cur.Close(ctx)
		if err := bulk(ctx, col, models); err != nil {
			return fmt.Errorf("failed to migrate %s: %w", name, err)
		}
		if _, err := col.DeleteMany(ctx, bson.M{"o": s.jid, "a": prefixRegex(pnSignal + ":")}); err != nil {
			return fmt.Errorf("failed to delete old %s: %w", name, err)
		}
	}
	return nil
}

// ── PreKeys ─────────────────────────────────────────────────────────────────

type preKeyDoc struct {
	KeyID int64  `bson:"kid"`
	Key   []byte `bson:"k"`
	Up    bool   `bson:"up"`
}

func (d preKeyDoc) toKey() (*keys.PreKey, error) {
	if len(d.Key) != 32 {
		return nil, ErrInvalidLength
	}
	return &keys.PreKey{
		KeyPair: *keys.NewKeyPairFromPrivateKey(*(*[32]byte)(d.Key)),
		KeyID:   uint32(d.KeyID),
	}, nil
}

func (s *Store) nextPreKeyID(ctx context.Context) (uint32, error) {
	var d preKeyDoc
	err := s.c("prekey").FindOne(ctx, bson.M{"o": s.jid},
		options.FindOne().SetSort(bson.D{{Key: "kid", Value: -1}})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return 1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to query next prekey ID: %w", err)
	}
	return uint32(d.KeyID) + 1, nil
}

func (s *Store) genOnePreKey(ctx context.Context, id uint32, uploaded bool) (*keys.PreKey, error) {
	key := keys.NewPreKey(id)
	err := upsert(ctx, s.c("prekey"), s.key(fmt.Sprint(id)), bson.M{
		"o": s.jid, "kid": int64(id), "k": key.Priv[:], "up": uploaded,
	})
	return key, err
}

func (s *Store) GenOnePreKey(ctx context.Context) (*keys.PreKey, error) {
	s.preKeyLock.Lock()
	defer s.preKeyLock.Unlock()
	next, err := s.nextPreKeyID(ctx)
	if err != nil {
		return nil, err
	}
	return s.genOnePreKey(ctx, next, true)
}

func (s *Store) GetOrGenPreKeys(ctx context.Context, count uint32) ([]*keys.PreKey, error) {
	s.preKeyLock.Lock()
	defer s.preKeyLock.Unlock()

	cur, err := s.c("prekey").Find(ctx, bson.M{"o": s.jid, "up": false},
		options.Find().SetSort(bson.D{{Key: "kid", Value: 1}}).SetLimit(int64(count)))
	if err != nil {
		return nil, fmt.Errorf("failed to query existing prekeys: %w", err)
	}
	defer cur.Close(ctx)

	var result []*keys.PreKey
	for cur.Next(ctx) {
		var d preKeyDoc
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		k, err := d.toKey()
		if err != nil {
			return nil, err
		}
		result = append(result, k)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}

	if uint32(len(result)) < count {
		next, err := s.nextPreKeyID(ctx)
		if err != nil {
			return nil, err
		}
		for uint32(len(result)) < count {
			k, err := s.genOnePreKey(ctx, next, false)
			if err != nil {
				return nil, fmt.Errorf("failed to generate prekey: %w", err)
			}
			result = append(result, k)
			next++
		}
	}
	return result, nil
}

func (s *Store) GetPreKey(ctx context.Context, id uint32) (*keys.PreKey, error) {
	var d preKeyDoc
	ok, err := findOne(ctx, s.c("prekey"), s.key(fmt.Sprint(id)), &d)
	if err != nil || !ok {
		return nil, err
	}
	return d.toKey()
}

func (s *Store) RemovePreKey(ctx context.Context, id uint32) error {
	_, err := s.c("prekey").DeleteOne(ctx, bson.M{"_id": s.key(fmt.Sprint(id))})
	return err
}

func (s *Store) MarkPreKeysAsUploaded(ctx context.Context, upToID uint32) error {
	_, err := s.c("prekey").UpdateMany(ctx,
		bson.M{"o": s.jid, "kid": bson.M{"$lte": int64(upToID)}},
		bson.M{"$set": bson.M{"up": true}})
	return err
}

func (s *Store) UploadedPreKeyCount(ctx context.Context) (int, error) {
	n, err := s.c("prekey").CountDocuments(ctx, bson.M{"o": s.jid, "up": true})
	return int(n), err
}

// ── Sender keys ─────────────────────────────────────────────────────────────

func (s *Store) PutSenderKey(ctx context.Context, group, user string, session []byte) error {
	return upsert(ctx, s.c("senderkey"), s.key(group, user), bson.M{"o": s.jid, "c": group, "a": user, "v": session})
}

func (s *Store) GetSenderKey(ctx context.Context, group, user string) ([]byte, error) {
	var d bin
	ok, err := findOne(ctx, s.c("senderkey"), s.key(group, user), &d)
	if err != nil || !ok {
		return nil, err
	}
	return d.V, nil
}

// ── App state sync keys ─────────────────────────────────────────────────────

type appKeyDoc struct {
	KID  []byte `bson:"kid"`
	Data []byte `bson:"d"`
	TS   int64  `bson:"ts"`
	FP   []byte `bson:"fp"`
}

func (d appKeyDoc) toKey() *store.AppStateSyncKey {
	return &store.AppStateSyncKey{Data: d.Data, Fingerprint: d.FP, Timestamp: d.TS}
}

func (s *Store) PutAppStateSyncKey(ctx context.Context, id []byte, key store.AppStateSyncKey) error {
	var existing appKeyDoc
	ok, err := findOne(ctx, s.c("appkey"), s.key(hex.EncodeToString(id)), &existing)
	if err != nil {
		return err
	}
	if ok && existing.TS >= key.Timestamp {
		return nil
	}
	return upsert(ctx, s.c("appkey"), s.key(hex.EncodeToString(id)), bson.M{
		"o": s.jid, "kid": id, "d": key.Data, "ts": key.Timestamp, "fp": key.Fingerprint,
	})
}

func (s *Store) GetAllAppStateSyncKeys(ctx context.Context) ([]*store.AppStateSyncKey, error) {
	cur, err := s.c("appkey").Find(ctx, bson.M{"o": s.jid},
		options.Find().SetSort(bson.D{{Key: "ts", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []*store.AppStateSyncKey
	for cur.Next(ctx) {
		var d appKeyDoc
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		out = append(out, d.toKey())
	}
	return out, cur.Err()
}

func (s *Store) GetAppStateSyncKey(ctx context.Context, id []byte) (*store.AppStateSyncKey, error) {
	var d appKeyDoc
	ok, err := findOne(ctx, s.c("appkey"), s.key(hex.EncodeToString(id)), &d)
	if err != nil || !ok {
		return nil, err
	}
	return d.toKey(), nil
}

func (s *Store) GetLatestAppStateSyncKeyID(ctx context.Context) ([]byte, error) {
	var d appKeyDoc
	err := s.c("appkey").FindOne(ctx, bson.M{"o": s.jid},
		options.FindOne().SetSort(bson.D{{Key: "ts", Value: -1}})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d.KID, nil
}

// ── App state versions / mutation MACs ──────────────────────────────────────

func (s *Store) PutAppStateVersion(ctx context.Context, name string, version uint64, hash [128]byte) error {
	return upsert(ctx, s.c("appver"), s.key(name), bson.M{"o": s.jid, "ver": int64(version), "h": hash[:]})
}

func (s *Store) GetAppStateVersion(ctx context.Context, name string) (uint64, [128]byte, error) {
	var hash [128]byte
	var d struct {
		Ver int64  `bson:"ver"`
		H   []byte `bson:"h"`
	}
	ok, err := findOne(ctx, s.c("appver"), s.key(name), &d)
	if err != nil {
		return 0, hash, err
	}
	if !ok {
		return 0, hash, nil // version 0 + empty hash is the correct initial state
	}
	if len(d.H) != 128 {
		return 0, hash, ErrInvalidLength
	}
	if d.Ver == 0 {
		return 0, hash, fmt.Errorf("invalid saved app state version 0 for name %s", name)
	}
	return uint64(d.Ver), *(*[128]byte)(d.H), nil
}

func (s *Store) DeleteAppStateVersion(ctx context.Context, name string) error {
	_, err := s.c("appver").DeleteOne(ctx, bson.M{"_id": s.key(name)})
	return err
}

func (s *Store) PutAppStateMutationMACs(ctx context.Context, name string, version uint64, mutations []store.AppStateMutationMAC) error {
	models := make([]mongo.WriteModel, 0, len(mutations))
	for _, m := range mutations {
		models = append(models, upModel(s.key(name, hex.EncodeToString(m.IndexMAC)), bson.M{
			"o": s.jid, "n": name, "ver": int64(version), "v": m.ValueMAC,
		}))
	}
	return bulk(ctx, s.c("appmac"), models)
}

func (s *Store) DeleteAppStateMutationMACs(ctx context.Context, name string, indexMACs [][]byte) error {
	if len(indexMACs) == 0 {
		return nil
	}
	ids := make([]string, len(indexMACs))
	for i, m := range indexMACs {
		ids[i] = s.key(name, hex.EncodeToString(m))
	}
	_, err := s.c("appmac").DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ids}})
	return err
}

func (s *Store) GetAppStateMutationMAC(ctx context.Context, name string, indexMAC []byte) ([]byte, error) {
	var d bin
	ok, err := findOne(ctx, s.c("appmac"), s.key(name, hex.EncodeToString(indexMAC)), &d)
	if err != nil || !ok {
		return nil, err
	}
	return d.V, nil
}

// ── Chat settings ───────────────────────────────────────────────────────────

func (s *Store) putChat(ctx context.Context, chat types.JID, field string, val interface{}) error {
	return upsert(ctx, s.c("chat"), s.key(chat.String()), bson.M{"o": s.jid, field: val})
}

func (s *Store) PutMutedUntil(ctx context.Context, chat types.JID, mutedUntil time.Time) error {
	var val int64
	if mutedUntil.Equal(store.MutedForever) {
		val = -1
	} else if !mutedUntil.IsZero() {
		val = mutedUntil.Unix()
	}
	return s.putChat(ctx, chat, "muted", val)
}

func (s *Store) PutPinned(ctx context.Context, chat types.JID, pinned bool) error {
	return s.putChat(ctx, chat, "pinned", pinned)
}

func (s *Store) PutArchived(ctx context.Context, chat types.JID, archived bool) error {
	return s.putChat(ctx, chat, "archived", archived)
}

func (s *Store) PutWASARootSecretID(ctx context.Context, chat types.JID, id types.MessageID) error {
	return s.putChat(ctx, chat, "wasa", string(id))
}

type chatDoc struct {
	Muted    int64  `bson:"muted"`
	Pinned   bool   `bson:"pinned"`
	Archived bool   `bson:"archived"`
	Wasa     string `bson:"wasa"`
}

func (s *Store) GetWASARootSecretID(ctx context.Context, chat types.JID) (types.MessageID, error) {
	var d chatDoc
	if _, err := findOne(ctx, s.c("chat"), s.key(chat.String()), &d); err != nil {
		return "", err
	}
	return types.MessageID(d.Wasa), nil
}

func (s *Store) GetChatSettings(ctx context.Context, chat types.JID) (types.LocalChatSettings, error) {
	var out types.LocalChatSettings
	var d chatDoc
	ok, err := findOne(ctx, s.c("chat"), s.key(chat.String()), &d)
	if err != nil || !ok {
		return out, err
	}
	out.Found = true
	out.Pinned = d.Pinned
	out.Archived = d.Archived
	if d.Muted < 0 {
		out.MutedUntil = store.MutedForever
	} else if d.Muted > 0 {
		out.MutedUntil = time.Unix(d.Muted, 0)
	}
	return out, nil
}
