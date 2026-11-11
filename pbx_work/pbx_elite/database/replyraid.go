package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ReplyRaidDoc persists one active reply-raid target.
type ReplyRaidDoc struct {
	// Composite key: chatJID + "|" + senderJID
	ID       string `bson:"_id"`
	ChatJID  string `bson:"chat_jid"`
	UserJID  string `bson:"user_jid"`
	PoolName string `bson:"pool_name"`
}

func replyRaidCol() *mongo.Collection {
	return DB.Collection("reply_raids")
}

func replyRaidID(chatJID, userJID string) string {
	return chatJID + "|" + userJID
}

// SaveReplyRaid upserts a reply-raid target.
func SaveReplyRaid(chatJID, userJID, poolName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	doc := ReplyRaidDoc{
		ID:       replyRaidID(chatJID, userJID),
		ChatJID:  chatJID,
		UserJID:  userJID,
		PoolName: poolName,
	}
	_, err := replyRaidCol().UpdateOne(
		ctx,
		bson.M{"_id": doc.ID},
		bson.M{"$set": doc},
		options.Update().SetUpsert(true),
	)
	return err
}

// DeleteReplyRaid removes one reply-raid target.
func DeleteReplyRaid(chatJID, userJID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := replyRaidCol().DeleteMany(ctx, bson.M{"_id": replyRaidID(chatJID, userJID)})
	return err
}

// DeleteAllReplyRaidsForChat removes every target in a given chat.
func DeleteAllReplyRaidsForChat(chatJID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := replyRaidCol().DeleteMany(ctx, bson.M{"chat_jid": chatJID})
	return err
}

// LoadAllReplyRaids returns a nested map[chatJID]map[userJID]poolName
// that the replyraid module can use to repopulate its in-memory store on startup.
func LoadAllReplyRaids() map[string]map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out := make(map[string]map[string]string)

	cur, err := replyRaidCol().Find(ctx, bson.M{})
	if err != nil {
		return out
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var doc ReplyRaidDoc
		if err := cur.Decode(&doc); err != nil {
			continue
		}
		if _, ok := out[doc.ChatJID]; !ok {
			out[doc.ChatJID] = make(map[string]string)
		}
		out[doc.ChatJID][doc.UserJID] = doc.PoolName
	}
	return out
}

// CountReplyRaidsForChat returns how many targets are active in a chat.
func CountReplyRaidsForChat(chatJID string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return replyRaidCol().CountDocuments(ctx, bson.M{"chat_jid": chatJID})
}
