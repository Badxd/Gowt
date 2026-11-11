package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AutoReplySettings holds the global on/off flag for auto-reply.
type AutoReplySettings struct {
	ID      string `bson:"_id"`
	Enabled bool   `bson:"enabled"`
}

// AutoReplyRule is one trigger→reply pair stored in the DB.
type AutoReplyRule struct {
	Trigger string `bson:"trigger"`
	Reply   string `bson:"reply"`
}

func arSettingsCol() *mongo.Collection { return DB.Collection("autoreply_settings") }
func arRulesCol() *mongo.Collection    { return DB.Collection("autoreply_rules") }

const arSettingsID = "global"

// GetAutoReplySettings loads the global enabled flag (defaults to true).
func GetAutoReplySettings() AutoReplySettings {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var s AutoReplySettings
	if err := arSettingsCol().FindOne(ctx, bson.M{"_id": arSettingsID}).Decode(&s); err != nil {
		return AutoReplySettings{ID: arSettingsID, Enabled: true}
	}
	return s
}

// SaveAutoReplyEnabled persists the enabled flag.
func SaveAutoReplyEnabled(enabled bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := arSettingsCol().UpdateOne(
		ctx,
		bson.M{"_id": arSettingsID},
		bson.M{"$set": AutoReplySettings{ID: arSettingsID, Enabled: enabled}},
		options.Update().SetUpsert(true),
	)
	return err
}

// LoadAutoReplyRules returns all saved trigger→reply pairs.
func LoadAutoReplyRules() []AutoReplyRule {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cur, err := arRulesCol().Find(ctx, bson.M{})
	if err != nil {
		return nil
	}
	defer cur.Close(ctx)

	var rules []AutoReplyRule
	for cur.Next(ctx) {
		var r AutoReplyRule
		if err := cur.Decode(&r); err == nil {
			rules = append(rules, r)
		}
	}
	return rules
}

// SaveAutoReplyRule upserts a trigger→reply pair (trigger is the doc key).
func SaveAutoReplyRule(trigger, reply string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := arRulesCol().UpdateOne(
		ctx,
		bson.M{"trigger": trigger},
		bson.M{"$set": AutoReplyRule{Trigger: trigger, Reply: reply}},
		options.Update().SetUpsert(true),
	)
	return err
}

// DeleteAutoReplyRule removes the rule for the given trigger.
func DeleteAutoReplyRule(trigger string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := arRulesCol().DeleteMany(ctx, bson.M{"trigger": trigger})
	return err
}

// ClearAutoReplyRules removes all rules.
func ClearAutoReplyRules() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := arRulesCol().DeleteMany(ctx, bson.M{})
	return err
}
