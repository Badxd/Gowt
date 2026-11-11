package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GBannedDoc persists one gbanned (group-banned) user entry.
type GBannedDoc struct {
	// Composite key: groupJID + "|" + userJID
	ID       string `bson:"_id"`
	GroupJID string `bson:"group_jid"`
	UserJID  string `bson:"user_jid"`
}

func gbannedCol() *mongo.Collection { return DB.Collection("gbanned") }

func gbannedID(groupJID, userJID string) string { return groupJID + "|" + userJID }

// SaveGBanned upserts a gbanned entry.
func SaveGBanned(groupJID, userJID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	doc := GBannedDoc{
		ID:       gbannedID(groupJID, userJID),
		GroupJID: groupJID,
		UserJID:  userJID,
	}
	_, err := gbannedCol().UpdateOne(
		ctx,
		bson.M{"_id": doc.ID},
		bson.M{"$setOnInsert": doc},
		options.Update().SetUpsert(true),
	)
	return err
}

// DeleteGBanned removes a gbanned entry.
func DeleteGBanned(groupJID, userJID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := gbannedCol().DeleteMany(ctx, bson.M{"_id": gbannedID(groupJID, userJID)})
	return err
}

// LoadAllGBanned returns map[groupJID]map[userJID]true for in-memory repopulation.
func LoadAllGBanned() map[string]map[string]bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out := make(map[string]map[string]bool)
	cur, err := gbannedCol().Find(ctx, bson.M{})
	if err != nil {
		return out
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var doc GBannedDoc
		if err := cur.Decode(&doc); err != nil {
			continue
		}
		if out[doc.GroupJID] == nil {
			out[doc.GroupJID] = make(map[string]bool)
		}
		out[doc.GroupJID][doc.UserJID] = true
	}
	return out
}
