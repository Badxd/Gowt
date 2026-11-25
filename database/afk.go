package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AFKState stores the bot owner's AFK status.
type AFKState struct {
	ID      string    `bson:"_id"`
	Enabled bool      `bson:"enabled"`
	Reason  string    `bson:"reason"`
	Since   time.Time `bson:"since"`
}

func afkCol() *mongo.Collection {
	return DB.Collection("afk_state")
}

const afkStateID = "owner"

// GetAFKState loads the AFK state from DB.
// Returns a safe default (disabled) if not found.
func GetAFKState() AFKState {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var s AFKState
	err := afkCol().FindOne(ctx, bson.M{"_id": afkStateID}).Decode(&s)
	if err != nil {
		return AFKState{ID: afkStateID, Enabled: false, Reason: "I am AFK right now."}
	}
	return s
}

// SetAFKState saves the AFK state to DB.
func SetAFKState(enabled bool, reason string, since time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := afkCol().UpdateOne(
		ctx,
		bson.M{"_id": afkStateID},
		bson.M{"$set": AFKState{
			ID:      afkStateID,
			Enabled: enabled,
			Reason:  reason,
			Since:   since,
		}},
		options.Update().SetUpsert(true),
	)
	return err
}

// AFKNotified tracks which users have already been notified during the current AFK session.
type AFKNotified struct {
	JID       string    `bson:"jid"`
	NotifiedAt time.Time `bson:"notified_at"`
}

func afkNotifiedCol() *mongo.Collection {
	return DB.Collection("afk_notified")
}

// IsAFKNotified returns true if this JID was already notified in the current AFK session.
func IsAFKNotified(jid string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	n, err := afkNotifiedCol().CountDocuments(ctx, bson.M{"jid": jid})
	return err == nil && n > 0
}

// MarkAFKNotified records that this JID was notified.
func MarkAFKNotified(jid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := afkNotifiedCol().UpdateOne(
		ctx,
		bson.M{"jid": jid},
		bson.M{"$setOnInsert": AFKNotified{JID: jid, NotifiedAt: time.Now()}},
		options.Update().SetUpsert(true),
	)
	return err
}

// ClearAFKNotified removes all notified records (called when AFK is toggled on/off).
func ClearAFKNotified() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := afkNotifiedCol().DeleteMany(ctx, bson.M{})
	return err
}
