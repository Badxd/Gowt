package mongostore

import (
	"context"
	"fmt"
	"sync"

	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var _ store.LIDStore = (*LIDMap)(nil)

// LIDMap stores the global LID <-> phone-number mapping. The whole table is
// small, so it is kept in memory and written through to MongoDB.
type LIDMap struct {
	col *mongo.Collection

	mu      sync.RWMutex
	pnToLID map[string]string
	lidToPN map[string]string
}

func newLIDMap(ctx context.Context, db *mongo.Database) (*LIDMap, error) {
	m := &LIDMap{
		col:     db.Collection("wm_lid"),
		pnToLID: map[string]string{},
		lidToPN: map[string]string{},
	}
	cur, err := m.col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var d struct {
			PN  string `bson:"_id"`
			LID string `bson:"lid"`
		}
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		m.pnToLID[d.PN] = d.LID
		m.lidToPN[d.LID] = d.PN
	}
	return m, cur.Err()
}

func (m *LIDMap) GetLIDForPN(ctx context.Context, pn types.JID) (types.JID, error) {
	if pn.Server != types.DefaultUserServer {
		return types.JID{}, fmt.Errorf("invalid GetLIDForPN call with non-PN JID %s", pn)
	}
	m.mu.RLock()
	lid, ok := m.pnToLID[pn.User]
	m.mu.RUnlock()
	if !ok || lid == "" {
		return types.JID{}, nil
	}
	return types.JID{User: lid, Device: pn.Device, Server: types.HiddenUserServer}, nil
}

func (m *LIDMap) GetPNForLID(ctx context.Context, lid types.JID) (types.JID, error) {
	if lid.Server != types.HiddenUserServer {
		return types.JID{}, fmt.Errorf("invalid GetPNForLID call with non-LID JID %s", lid)
	}
	m.mu.RLock()
	pn, ok := m.lidToPN[lid.User]
	m.mu.RUnlock()
	if !ok || pn == "" {
		return types.JID{}, nil
	}
	return types.JID{User: pn, Device: lid.Device, Server: types.DefaultUserServer}, nil
}

func (m *LIDMap) GetManyLIDsForPNs(ctx context.Context, pns []types.JID) (map[types.JID]types.JID, error) {
	out := make(map[types.JID]types.JID, len(pns))
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, pn := range pns {
		if pn.Server != types.DefaultUserServer {
			continue
		}
		if lid, ok := m.pnToLID[pn.User]; ok && lid != "" {
			out[pn] = types.JID{User: lid, Device: pn.Device, Server: types.HiddenUserServer}
		}
	}
	return out, nil
}

// putLocked writes one mapping; caller holds m.mu (write).
func (m *LIDMap) putLocked(ctx context.Context, lid, pn string) error {
	if old, ok := m.pnToLID[pn]; ok && old == lid {
		return nil
	}
	// drop any stale pair that shares the LID or the PN
	if oldPN, ok := m.lidToPN[lid]; ok && oldPN != pn {
		delete(m.pnToLID, oldPN)
		_, _ = m.col.DeleteOne(ctx, bson.M{"_id": oldPN})
	}
	if oldLID, ok := m.pnToLID[pn]; ok && oldLID != lid {
		delete(m.lidToPN, oldLID)
	}
	_, err := m.col.UpdateOne(ctx, bson.M{"_id": pn}, bson.M{"$set": bson.M{"lid": lid}}, upsertOpt())
	if err != nil {
		return err
	}
	m.pnToLID[pn] = lid
	m.lidToPN[lid] = pn
	return nil
}

func (m *LIDMap) PutLIDMapping(ctx context.Context, lid, pn types.JID) error {
	if lid.Server != types.HiddenUserServer || pn.Server != types.DefaultUserServer {
		return fmt.Errorf("invalid PutLIDMapping call %s/%s", lid, pn)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.putLocked(ctx, lid.User, pn.User)
}

func (m *LIDMap) PutManyLIDMappings(ctx context.Context, mappings []store.LIDMapping) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, mp := range mappings {
		if mp.LID.Server != types.HiddenUserServer || mp.PN.Server != types.DefaultUserServer {
			continue
		}
		if err := m.putLocked(ctx, mp.LID.User, mp.PN.User); err != nil {
			return err
		}
	}
	return nil
}
