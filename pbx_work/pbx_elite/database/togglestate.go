package database

// togglestate.go — generic per-feature toggle persistence.
//
// Used by: autosave (single global bool), antilink (per-chat bool),
// antibot (per-chat bool).
//
// Collection layout:
//   toggle_states  { _id: "<feature>:<scope>", feature: string, scope: string, enabled: bool }
//
// scope is "global" for single-bool features, or chatJID for per-chat toggles.

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ToggleState struct {
	ID      string `bson:"_id"`
	Feature string `bson:"feature"`
	Scope   string `bson:"scope"`
	Enabled bool   `bson:"enabled"`
}

func toggleCol() *mongo.Collection { return DB.Collection("toggle_states") }

func toggleID(feature, scope string) string { return feature + ":" + scope }

// SetToggle upserts a boolean toggle for feature+scope.
func SetToggle(feature, scope string, enabled bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id := toggleID(feature, scope)
	_, err := toggleCol().UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": ToggleState{ID: id, Feature: feature, Scope: scope, Enabled: enabled}},
		options.Update().SetUpsert(true),
	)
	return err
}

// GetToggle retrieves the bool for feature+scope; returns defaultVal if not found.
func GetToggle(feature, scope string, defaultVal bool) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var t ToggleState
	if err := toggleCol().FindOne(ctx, bson.M{"_id": toggleID(feature, scope)}).Decode(&t); err != nil {
		return defaultVal
	}
	return t.Enabled
}

// LoadAllToggles returns all scope→enabled pairs for a given feature.
// Useful for per-chat toggles that need to populate a map on startup.
func LoadAllToggles(feature string) map[string]bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out := make(map[string]bool)
	cur, err := toggleCol().Find(ctx, bson.M{"feature": feature})
	if err != nil {
		return out
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var t ToggleState
		if err := cur.Decode(&t); err == nil {
			out[t.Scope] = t.Enabled
		}
	}
	return out
}

// DeleteToggle removes a toggle entry (e.g. when disabling a per-chat feature).
func DeleteToggle(feature, scope string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := toggleCol().DeleteMany(ctx, bson.M{"_id": toggleID(feature, scope)})
	return err
}
