package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SudoUser represents a user with elevated bot privileges.
type SudoUser struct {
	JID     string    `bson:"jid"`
	AddedBy string    `bson:"added_by"`
	AddedAt time.Time `bson:"added_at"`
}

func sudoCol() *mongo.Collection {
	return DB.Collection("sudo_users")
}

// AddSudo grants sudo access to the given JID.
// Uses upsert so duplicate inserts are safe.
func AddSudo(jid, addedBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := sudoCol().UpdateOne(
		ctx,
		bson.M{"jid": jid},
		bson.M{"$setOnInsert": SudoUser{
			JID:     jid,
			AddedBy: addedBy,
			AddedAt: time.Now(),
		}},
		options.Update().SetUpsert(true),
	)
	return err
}

// RemoveSudo revokes sudo access from the given JID.
func RemoveSudo(jid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := sudoCol().DeleteOne(ctx, bson.M{"jid": jid})
	return err
}

// IsSudo reports whether the given JID has sudo access.
func IsSudo(jid string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	n, err := sudoCol().CountDocuments(ctx, bson.M{"jid": jid})
	return err == nil && n > 0
}

// ListSudo returns all current sudo users.
func ListSudo() ([]SudoUser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := sudoCol().Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []SudoUser
	return users, cursor.All(ctx, &users)
}
