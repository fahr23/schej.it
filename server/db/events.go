package db

import (
	"context"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"schej.it/server/logger"
	"schej.it/server/models"
)

// Returns an event based on its _id
func GetEventById(eventId string) *models.Event {
	objectId, err := primitive.ObjectIDFromHex(eventId)
	if err != nil {
		// eventId is malformatted
		return nil
	}
	result := EventsCollection.FindOne(context.Background(), bson.M{
		"$and": bson.A{
			bson.M{"_id": objectId},
			bson.M{
				"$or": bson.A{
					bson.M{"isDeleted": bson.M{"$exists": false}},
					bson.M{"isDeleted": bson.M{"$eq": false}},
				},
			},
		},
	})
	if result.Err() == mongo.ErrNoDocuments {
		// Event does not exist!
		return nil
	}

	// Decode result
	var event models.Event
	if err := result.Decode(&event); err != nil {
		logger.StdErr.Panicln(err)
	}

	return &event
}

// Returns an event based on its shortId
func GetEventByShortId(shortEventId string) *models.Event {
	result := EventsCollection.FindOne(context.Background(), bson.M{
		"$and": bson.A{
			bson.M{"shortId": shortEventId},
			bson.M{
				"$or": bson.A{
					bson.M{"isDeleted": bson.M{"$exists": false}},
					bson.M{"isDeleted": bson.M{"$eq": false}},
				},
			},
		},
	})
	if result.Err() == mongo.ErrNoDocuments {
		// Event does not exist!
		return nil
	}

	// Decode result
	var event models.Event
	if err := result.Decode(&event); err != nil {
		logger.StdErr.Panicln(err)
	}

	return &event
}

// Returns an event by either its _id or shortId
func GetEventByEitherId(id string) *models.Event {
	if len(id) <= 10 {
		return GetEventByShortId(id)
	}

	return GetEventById(id)
}

func GetEventResponses(eventId string) []models.EventResponse {
	objectId, err := primitive.ObjectIDFromHex(eventId)
	if err != nil {
		// eventId is malformatted
		return []models.EventResponse{}
	}

	result, err := EventResponsesCollection.Find(context.Background(), bson.M{
		"eventId": objectId,
	})
	if err != nil {
		logger.StdErr.Panicln(err)
	}
	if result.Err() == mongo.ErrNoDocuments {
		// Event responses do not exist!
		return []models.EventResponse{}
	}

	var eventResponses []models.EventResponse
	if err := result.All(context.Background(), &eventResponses); err != nil {
		logger.StdErr.Panicln(err)
	}

	return eventResponses
}

func GetAttendees(eventId string) []models.Attendee {
	objectId, err := primitive.ObjectIDFromHex(eventId)
	if err != nil {
		// eventId is malformatted
		return []models.Attendee{}
	}

	result, err := AttendeesCollection.Find(context.Background(), bson.M{
		"eventId": objectId,
	})
	if err != nil {
		logger.StdErr.Panicln(err)
	}
	if result.Err() == mongo.ErrNoDocuments {
		// Attendees do not exist!
		return []models.Attendee{}
	}

	var attendees []models.Attendee
	if err := result.All(context.Background(), &attendees); err != nil {
		logger.StdErr.Panicln(err)
	}

	return attendees
}

func GetEventsCreatedThisMonth(userId primitive.ObjectID) int {
	// Get the start of this month
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	result, err := EventsCollection.CountDocuments(context.Background(), bson.M{
		"ownerId": userId,
		"_id": bson.M{
			"$gte": primitive.NewObjectIDFromTimestamp(startOfMonth),
		},
	})
	if err != nil {
		logger.StdErr.Panicln(err)
	}

	return int(result)
}

// Returns a random unique short event id seeded by the actual event id
func GenerateShortEventId(eventId primitive.ObjectID) string {
	r := rand.New(rand.NewSource(eventId.Timestamp().Unix()))

	id := ""

	letters := "23456789ABCDEFabcdef"
	for i := 0; i < 5; i++ {
		index := r.Intn(len(letters))
		letter := letters[index : index+1]
		id += letter
	}

	i := 0
	event := GetEventByShortId(id)
	for event != nil && i < 5 {
		// Event exists, keep on adding letters until event doesn't exist anymore, max of 5 more letters
		index := r.Intn(len(letters))
		letter := letters[index : index+1]
		id += letter
		event = GetEventByShortId(id)
		i++
	}

	if event != nil {
		logger.StdErr.Panicln("Couldn't generate unique id")
	}

	return id
}

// Updates the name of a guest response
func UpdateGuestResponseName(eventId string, oldName string, newName string) {
	objectId, err := primitive.ObjectIDFromHex(eventId)
	if err != nil {
		// eventId is malformatted
		return
	}

	_, err = EventResponsesCollection.UpdateOne(context.Background(), bson.M{
		"eventId": objectId,
		"userId":  oldName,
	}, bson.M{
		"$set": bson.M{
			"userId":        newName,
			"response.name": newName,
		},
	})
	if err != nil {
		logger.StdErr.Panicln(err)
	}
}
