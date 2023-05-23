package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/thedevsaddam/renderer"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var rnd *renderer.Render
var db *mongo.Database

const (
	connectionString = "mongodb://localhost:27017"
	dbName           = "todo_db"
	collectionName   = "todo"
	port             = ":9000"
)

type (
	todoMD struct {
		ID        primitive.ObjectID `bson:"_id,omitempty"`
		Title     string             `bson:"title"`
		Completed bool               `bson:"completed"`
		CreatedAt time.Time          `bson:"created_at"`
	}

	todoJson struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		Completed bool      `json:"completed"`
		CreatedAt time.Time `json:"created_at"`
	}
)

func init() {
	rnd = renderer.New()
	client, err := mongo.NewClient(options.Client().ApplyURI(connectionString))
	checkErr(err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	checkErr(err)

	db = client.Database(dbName)
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", homeHandler)
	r.Mount("/todo", todoHandler())

	server := &http.Server{
		Addr:         port,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Println("Listening on port" + port)
		if err := server.ListenAndServe(); err != nil {
			log.Printf("Server failed to listen and serve on port %v: %v", port, err)
		}
	}()
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
	var todo_md []todoMD

	if err := db.C(collectionName).Find(bson.M{}).All(&todo_md); err != nil {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "failed to query through database",
			"error":   err,
		})
		return
	}

	todoList := []todoJson{}
	for _, t := range todo_md {
		todoList = append(todoList, todoJson{
			ID:        t.ID.Hex(),
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
	var received todoJson

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

	new_record := todoMD{
		ID:        bson.NewObjectId(),
		Title:     received.Title,
		Completed: false,
		CreatedAt: time.Now(),
	}

	if err := db.C(collectionName).Insert(); err != nil {
		rnd.JSON(w, http.StatusNotAcceptable, renderer.M{
			"message": "failed to insert new record to the database",
			"error":   err,
		})
		return
	}

	rnd.JSON(w, http.StatusCreated, renderer.M{
		"message": "new record created successfully",
		"todo_id": new_record.ID.Hex(),
	})
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))

	// check if id is valid
	if !bson.IsObjectIdHex(id) {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "given id is not valid",
		})
		return
	}

	var received todoJson

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

	if err := db.C(collectionName).Update(
		bson.M{
			"_id": bson.ObjectIdHex(id),
		},
		bson.M{
			"title":     received.Title,
			"completed": received.Completed,
		},
	); err != nil {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "failed to update your selected id",
			"error":   err,
		})
		return
	}

	rnd.JSON(w, http.StatusCreated, renderer.M{
		"message": "selected record updated successfully",
	})
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	fmt.Printf("$$$$$$$$$$$$$$$$$$$$$$$\n|%v|\n$$$$$$$$$$$$$$$$$$$$$$$", chi.URLParam(r, "id"))

	if !bson.IsObjectIdHex(id) {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "the given id is not valid",
		})
		return
	}

	if err := db.C(collectionName).RemoveId(
		bson.M{"_id": bson.ObjectIdHex(id)},
	); err != nil {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "failed to remove selected record",
			"error":   err,
		})
		return
	}

	rnd.JSON(w, http.StatusOK, renderer.M{
		"message": "selected record has been deleted successfully",
	})
}

func checkErr(err error) {
	if err != nil {
		log.Printf("Error accrued: %v", err)
	}
}
