package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Warning tracks how many times a user has been warned in a specific group.
type Warning struct {
	GroupJID  string    `bson:"group_jid"`
	UserJID   string    `bson:"user_jid"`
	Count     int       `bson:"count"`
	Reasons   []string  `bson:"reasons"`
	UpdatedAt time.Time `bson:"updated_at"`
}

func warnCol() *mongo.Collection {
	return DB.Collection("warnings")
}

// AddWarning increments the warning count for a user in a group (creating
// the record if needed) and returns the new total count.
func AddWarning(groupJID, userJID, reason string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	after := options.After
	res := warnCol().FindOneAndUpdate(
		ctx,
		bson.M{"group_jid": groupJID, "user_jid": userJID},
		bson.M{
			"$inc":  bson.M{"count": 1},
			"$push": bson.M{"reasons": reason},
			"$set":  bson.M{"updated_at": time.Now()},
		},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(after),
	)

	var w Warning
	if err := res.Decode(&w); err != nil {
		return 0, err
	}
	return w.Count, nil
}

// GetWarningCount returns the current warning count for a user in a group
// (0 if the user has never been warned there).
func GetWarningCount(groupJID, userJID string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var w Warning
	err := warnCol().FindOne(ctx, bson.M{"group_jid": groupJID, "user_jid": userJID}).Decode(&w)
	if err != nil {
		return 0
	}
	return w.Count
}

// ClearWarnings removes all warnings for a user in a group.
func ClearWarnings(groupJID, userJID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := warnCol().DeleteOne(ctx, bson.M{"group_jid": groupJID, "user_jid": userJID})
	return err
}
