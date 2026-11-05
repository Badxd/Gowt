// Package mongostore is a MongoDB implementation of whatsmeow's store
// interfaces. It replaces sqlstore/SQLite completely, so the WhatsApp session
// (keys, sessions, prekeys, app-state, contacts ...) survives Heroku restarts.
package mongostore

import (
	"context"
	"errors"
	"regexp"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type bin struct {
	V []byte `bson:"v"`
}

func upsert(ctx context.Context, col *mongo.Collection, id string, set bson.M) error {
	_, err := col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": set}, options.Update().SetUpsert(true))
	return err
}

func upModel(id string, set bson.M) mongo.WriteModel {
	return mongo.NewUpdateOneModel().
		SetFilter(bson.M{"_id": id}).
		SetUpdate(bson.M{"$set": set}).
		SetUpsert(true)
}

func bulk(ctx context.Context, col *mongo.Collection, models []mongo.WriteModel) error {
	if len(models) == 0 {
		return nil
	}
	_, err := col.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
	return err
}

// findOne decodes the document with the given _id; (false, nil) if it does not exist.
func findOne(ctx context.Context, col *mongo.Collection, id string, out interface{}) (bool, error) {
	err := col.FindOne(ctx, bson.M{"_id": id}).Decode(out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func prefixRegex(prefix string) bson.M {
	return bson.M{"$regex": "^" + regexp.QuoteMeta(prefix)}
}

func upsertOpt() *options.UpdateOptions { return options.Update().SetUpsert(true) }
