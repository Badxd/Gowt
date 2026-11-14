package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PMRecord stores the approval/block status of a user in PM Security.
type PMRecord struct {
	JID       string    `bson:"jid"`
	Approved  bool      `bson:"approved"`
	Blocked   bool      `bson:"blocked"`
	WarnCount int       `bson:"warn_count"`
	UpdatedAt time.Time `bson:"updated_at"`
}

func pmCol() *mongo.Collection {
	return DB.Collection("pm_security")
}

// UpsertPMRecord saves or updates a PM record for a JID.
func UpsertPMRecord(jid string, approved, blocked bool, warns int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := pmCol().UpdateOne(
		ctx,
		bson.M{"jid": jid},
		bson.M{"$set": PMRecord{
			JID:       jid,
			Approved:  approved,
			Blocked:   blocked,
			WarnCount: warns,
			UpdatedAt: time.Now(),
		}},
		options.Update().SetUpsert(true),
	)
	return err
}

// GetPMRecord fetches the PM record for a JID.
// Returns a zero-value PMRecord (not found = not approved, not blocked, 0 warns).
func GetPMRecord(jid string) PMRecord {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var rec PMRecord
	err := pmCol().FindOne(ctx, bson.M{"jid": jid}).Decode(&rec)
	if err != nil {
		return PMRecord{JID: jid}
	}
	return rec
}

// DeletePMRecord removes a PM record entirely (used on full reset).
func DeletePMRecord(jid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := pmCol().DeleteOne(ctx, bson.M{"jid": jid})
	return err
}

// ListApprovedPM returns all approved users.
func ListApprovedPM() ([]PMRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := pmCol().Find(ctx, bson.M{"approved": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []PMRecord
	return records, cursor.All(ctx, &records)
}

// ResetAllPMRecords deletes all PM warn/block records.
func ResetAllPMRecords() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := pmCol().DeleteMany(ctx, bson.M{})
	return err
}

// PMSettings stores global PM Security settings.
type PMSettings struct {
	ID        string `bson:"_id"`
	Enabled   bool   `bson:"enabled"`
	MaxWarns  int    `bson:"max_warns"`
	CustomMsg string `bson:"custom_msg"`
}

func pmSettingsCol() *mongo.Collection {
	return DB.Collection("pm_settings")
}

const pmSettingsID = "global"

// GetPMSettings loads global PM settings from DB.
// Returns safe defaults if not found.
func GetPMSettings() PMSettings {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var s PMSettings
	err := pmSettingsCol().FindOne(ctx, bson.M{"_id": pmSettingsID}).Decode(&s)
	if err != nil {
		return PMSettings{ID: pmSettingsID, Enabled: true, MaxWarns: 3}
	}
	return s
}

// SavePMSettings persists global PM settings to DB.
func SavePMSettings(enabled bool, maxWarns int, customMsg string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := pmSettingsCol().UpdateOne(
		ctx,
		bson.M{"_id": pmSettingsID},
		bson.M{"$set": PMSettings{
			ID:        pmSettingsID,
			Enabled:   enabled,
			MaxWarns:  maxWarns,
			CustomMsg: customMsg,
		}},
		options.Update().SetUpsert(true),
	)
	return err
}
