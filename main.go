package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/google/uuid"
	"github.com/thedevsaddam/renderer"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var rnd *renderer.Render
var db *mongo.Database

const (
	port = ":9000"
)

type Todo struct {
	ID        string    `json:"id" bson:"_id,omitempty"`
	Title     string    `json:"title" bson:"title"`
	Completed bool      `json:"completed" bson:"completed"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

func main() {
	rnd = renderer.New()
	// Database configuration
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("hostName")))
	checkErr(err)
	db = client.Database("todo_app")

	// routing
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", homeHandler)
	r.Mount("/todo", todoHandler())

	// server configuration
	server := &http.Server{
		Addr:         port,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Println("Listening on port" + port)
	if err := server.ListenAndServe(); err != nil {
		log.Printf("Server failed to listen and serve on port %v: %v", port, err)
	}
}

func todoHandler() http.Handler {
	rg := chi.NewRouter()
	rg.Group(func(r chi.Router) {
		r.Get("/", fetchHandler)
		r.Post("/", createHandler)
		r.Put("/{id}", updateHandler)
		r.Delete("/{id}", deleteHandler)
	})
	return rg
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	err := rnd.Template(w, http.StatusOK, []string{"static/index.tpl"}, nil)
	checkErr(err)
}

func fetchHandler(w http.ResponseWriter, r *http.Request) {
	var todo_md []Todo

	ctx := context.Background()
	tasksCollection := db.Collection("tasks")
	tasksCursor, err := tasksCollection.Find(ctx, bson.M{})
	if err != nil {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "failed to query through database",
			"error":   err,
		})
		return
	}
	if err := tasksCursor.All(ctx, &todo_md); err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "failed to decode queries",
			"error":   err,
		})
	}

	todoList := []Todo{}
	for _, t := range todo_md {
		todoList = append(todoList, Todo{
			ID:        t.ID,
			Title:     t.Title,
			Completed: t.Completed,
			CreatedAt: t.CreatedAt,
		})
	}

	rnd.JSON(w, http.StatusOK, renderer.M{
		"data": todoList,
	})
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	var received Todo

	if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
		rnd.JSON(w, http.StatusBadGateway, renderer.M{
			"message": "failed to decode user request",
			"error":   err,
		})
		return
	}

	if received.Title == "" {
		rnd.JSON(w, http.StatusBadGateway, renderer.M{
			"message": "title is required",
			"error":   "title is required",
		})
		return
	}

	received.ID = uuid.New().String()
	received.CreatedAt = time.Now()
	received.Completed = false

	tasksCollection := db.Collection("tasks")
	result, err := tasksCollection.InsertOne(context.Background(), received)
	if err != nil {
		rnd.JSON(w, http.StatusNotAcceptable, renderer.M{
			"message": "failed to insert new record to the database",
			"error":   err,
		})
		return
	}

	rnd.JSON(w, http.StatusCreated, renderer.M{
		"message": "new record created successfully",
		"todo_id": result.InsertedID,
	})
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))

	var received Todo

	if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "failed to decode user request",
			"error":   err,
		})
		return
	}

	if received.Title == "" {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "title is required",
		})
		return
	}

	tasksCollection := db.Collection("tasks")
	result, err := tasksCollection.UpdateOne(
		context.Background(),
		bson.M{
			"_id": id,
		},
		bson.M{
			"title":     received.Title,
			"completed": received.Completed,
		},
		nil,
	)
	if err != nil {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "failed to update your selected id",
			"error":   err,
		})
		return
	}

	rnd.JSON(w, http.StatusCreated, renderer.M{
		"message": strconv.FormatInt(result.MatchedCount, 10) + "record updated successfully",
	})
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))

	tasksCollection := db.Collection("tasks")
	result, err := tasksCollection.DeleteOne(
		context.Background(),
		bson.M{"_id": id},
		nil,
	)
	if err != nil {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "failed to remove selected record",
			"error":   err,
		})
		return
	}

	rnd.JSON(w, http.StatusOK, renderer.M{
		"message": strconv.FormatInt(result.DeletedCount, 10) + "records has been deleted successfully",
	})
}

func checkErr(err error) {
	if err != nil {
		log.Printf("Error accrued: %v", err)
	}
}
