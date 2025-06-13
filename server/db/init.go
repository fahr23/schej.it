package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"schej.it/server/logger"
)

var Client *mongo.Client
var Db *mongo.Database
var EventsCollection *mongo.Collection
var UsersCollection *mongo.Collection
var DailyUserLogCollection *mongo.Collection
var FriendRequestsCollection *mongo.Collection
var EventResponsesCollection *mongo.Collection
var AttendeesCollection *mongo.Collection
var FoldersCollection *mongo.Collection
var FolderEventsCollection *mongo.Collection

func Init() func() {
	// Establish mongodb connection
	var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	Client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost"))
	if err != nil {
		logger.StdErr.Panicln(err)
	}

	// Define mongodb database + collections
	Db = Client.Database("schej-it")
	EventsCollection = Db.Collection("events")
	UsersCollection = Db.Collection("users")
	DailyUserLogCollection = Db.Collection("dailyuserlogs")
	FriendRequestsCollection = Db.Collection("friendrequests")
	EventResponsesCollection = Db.Collection("eventResponses")
	AttendeesCollection = Db.Collection("attendees")
	FoldersCollection = Db.Collection("folders")
	FolderEventsCollection = Db.Collection("folderEvents")

	// Return a function to close the connection
	return func() {
		Client.Disconnect(ctx)
	}
}

// MongoDB backup / restore commands

// Backup
// mongodump --uri="mongodb://localhost:27017" --db=schej-it

// Restore
// mongorestore --uri="mongodb://localhost:27017" --drop --db=schej-it ./dump
