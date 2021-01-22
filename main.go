package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/xid"
	"go.mongodb.org/mongo-driver/bson"
	// "go.mongodb.org/mongo-driver/bson/primitive"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// script parameters variables
const (
	helpInfo = `

                         -> mongo version: MongoDB shell version v3.6.21
--db-path=DBPATH         ->  database file path.
--help                   ->  open help menu.
                         ->  if there is no parameter, script tyr to use current database.
	 `
	db_p       = "--db-path"
	help       = "--help"
	emailRegEx = `(([a-zA-Z0-9]+)@([a-zA-Z0-9]+)\.([a-z]{3}|[a-z]{2}))`
)

// error message variables
const (
	wrongParameter = "! wrong parameter. Please look at --help"
	dbFileNotExist = "! database file not found."
	dumpLoaded     = "Dumped backup loaded."
	dumpNotLoaded  = "Dumped backup not loaded, bacuse --db-path parameter not given."
)

const (
	pending    = "Pending"
	inProgress = "In Progress"
	denied     = "Denied"
	accept     = "Accepted"
)

const (
	merketing   = "Marketing"
	design      = "Design"
	development = "Development"
)

const ceo = "Zafer"

type DB struct {
	param  string
	path   string
	client *mongo.Client
}

var db DB

type Candidate struct {
	Id               string    `json:"_id" bson:"_id"`
	First_name       string    `json:"first_name" bson:"first_name"`
	Last_name        string    `json:"last_name" bson:"last_name"`
	Email            string    `json:"email" bson:"email"`
	Department       string    `json:"department" bson:"department"`
	University       string    `json:"university" bson:"university"`
	Assignee         string    `json:"assignee" bson:"assignee"`
	Experience       bool      `json:"experience" bson:"experience"`
	Meeting_count    int       `json:"meeting_count" bson:"meeting_count"`
	Next_meeting     time.Time `json:"next_meeting" bson:"next_meeting"`
	Status           string    `json:"status" bson:"status"`
	Application_date time.Time `json:"application_date" bson:application_date`
}

type Assignee struct {
	Id         string `json:"_id" bson:"_id"`
	Name       string `json:"name" bson:"name"`
	Department string `json:"department" bson:"department"`
}

type Id struct {
	Id string `json:"_id" bson:"_id"`
}

type Meeting struct {
	Id                string    `json:"_id" bson:"_id"`
	Next_meeting_time time.Time `json:"nextMeetingTime" bson:"nextMeetingTime"`
}

// some assistant funcs end at line: 233
func RunCommand(command, input string) string {
	cmd := exec.Command(command)
	cmd.Stdin = strings.NewReader(input)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	return out.String()
}

func FillDBParams(scriptArgs []string) {
	if len(scriptArgs) > 1 {
		args := scriptArgs[1:]

		if args[0] == help {
			log.Fatal(helpInfo)
		}

		parameter := strings.Split(args[0], "=")
		if len(parameter) > 1 {
			isfileExist := RunCommand("/bin/bash", "[ -f "+parameter[1]+" ] && echo \"1\" || echo \"0\"")
			if isfileExist != "1\n" {
				log.Fatal(dbFileNotExist)
			}
			db.path = parameter[1]
			if db_p == parameter[0] {
				// return parameter[0]
				db.param = parameter[0]
			} else {
				log.Fatal(wrongParameter)
			}
		} else {
			log.Fatal(wrongParameter)
		}
	}
}

func CreateDb() {
	if db.param != "" {
		RunCommand("mongorestore", "--host localhost:27017 --gzip --archive="+db.path)
		log.Println(dumpLoaded)

	} else {
		log.Println(dumpNotLoaded)
	}
}

func ConnectDatabase() {
	log.Println("Database connecting...")
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")

	client, err := mongo.Connect(context.TODO(), clientOptions)
	db.client = client
	if err != nil {
		log.Fatal(err)
	}

	err = db.client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Database Connected.")
}

func LogRouterPath(r *http.Request) {
	path, err := filepath.Abs(r.URL.Path)
	_ = err
	log.Println(path)
}

func RouterHomePage(w http.ResponseWriter, r *http.Request) {
	LogRouterPath(r)
	fmt.Fprintf(w, `Welcome to the HomePage!

		api routers:
		  /crateCandidate
		  /readCandidate
		  /deleteCandidate
		  /arrangeMeeting
		  /completeMeeting
		  /acceptCandidate/{id}
		  /denyCandidate/{id}
		  /findAssigneeIDByName/{name}
		`)
}

func ParseBodyId(r *http.Request) (Id, error) {
	var obj Id
	err := json.NewDecoder(r.Body).Decode(&obj)
	return obj, err
}

func ParseBodyMeeting(r *http.Request) (Meeting, error) {
	var obj Meeting
	err := json.NewDecoder(r.Body).Decode(&obj)
	return obj, err
}

func UpdateRecord(filter map[string]interface{}, update map[string]interface{}, errMsg string) error {
	collection := db.client.Database("Otsimo").Collection("Candidates")
	result, err := collection.UpdateOne(
		context.Background(),
		filter,
		update,
	)
	if result.MatchedCount == 0 {
		err := errors.New(errMsg)
		return err
	}
	return err
}

func isCandidateNotExist(c Candidate) bool {
	collection := db.client.Database("Otsimo").Collection("Candidates")
	filter := bson.M{"email": c.Email}
	var result Candidate
	err := collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil && err.Error() == "mongo: no documents in result" {
		return true
	}
	return false
}

func InsertCandidate(c Candidate) error {
	collection := db.client.Database("Otsimo").Collection("Candidates")
	c.Id = xid.New().String()
	insertResult, err := collection.InsertOne(context.TODO(), c)
	_ = insertResult
	return err
}
// db funcs start here end at line :
func CreateCandidate(c Candidate) (Candidate, error) {
	if isCandidateNotExist(c) {
		if c.Email != "" {
			re := regexp.MustCompile(emailRegEx)
			result := re.FindAllString(c.Email, -1)
			if len(result[0]) == len(c.Email) { // if there are undefined characters at the end control it with compare length
				if c.Assignee != "" && c.Department != "" {
					err := InsertCandidate(c)
					return c, err
				}
				return c, errors.New("Assignee or Department is empty!")
			} else {
				err := errors.New("Invalid email!")
				return c, err
			}
		} else {
			err := errors.New("Empty email!")
			return c, err
		}
	} else {
		err := errors.New("Candidate already exist!")
		return c, err
	}
}

func HandleCreateCandidate(w http.ResponseWriter, r *http.Request) {
	LogRouterPath(r)
	var candidate Candidate
	candidate.Status = pending
	dt := time.Now()
	candidate.Application_date = dt
	err := json.NewDecoder(r.Body).Decode(&candidate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		c, _err := CreateCandidate(candidate)
		if _err != nil {
			http.Error(w, _err.Error(), http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, "User insert!\nfirst_name:"+c.First_name+"\nlast_name:"+c.Last_name+"\nemail:"+c.Email)
		}
	}
}

func ReadCandidate(id string) (Candidate, error) {
	collection := db.client.Database("Otsimo").Collection("Candidates")
	filter := bson.M{"_id": id}
	var result Candidate
	err := collection.FindOne(context.TODO(), filter).Decode(&result)
	return result, err
}

func HandleReadCandidate(w http.ResponseWriter, r *http.Request) {
	LogRouterPath(r)
	obj, err := ParseBodyId(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		c, _err := ReadCandidate(obj.Id)
		if _err != nil {
			http.Error(w, _err.Error(), http.StatusBadRequest)
		} else {
			out, err := json.Marshal(c)
			if err != nil {
				http.Error(w, _err.Error(), http.StatusBadRequest)
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, string(out))
		}
	}
}

func DeleteCandidate(id string) error {
	collection := db.client.Database("Otsimo").Collection("Candidates")
	res, err := collection.DeleteOne(context.TODO(), bson.M{"_id": id})
	_ = res
	return err
}

func HandleDeleteCandidate(w http.ResponseWriter, r *http.Request) {
	LogRouterPath(r)
	obj, err := ParseBodyId(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		err := DeleteCandidate(obj.Id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

func ArrangeMeeting(id string, nextMeetingTime time.Time) error {
	filter := bson.M{"_id": id, "meeting_count": bson.M{"$lt": 4}}
	update := bson.M{"$set": bson.M{"next_meeting": nextMeetingTime, "status": inProgress}}
	err := UpdateRecord(filter, update, "Candidate not suitable or meeting count limit is already 4.")
	return err
}

func HandleArrangeMeeting(w http.ResponseWriter, r *http.Request) {
	LogRouterPath(r)
	obj, err := ParseBodyMeeting(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		err := ArrangeMeeting(obj.Id, obj.Next_meeting_time)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}

func CompleteMeeting(id string) error {
	filter := bson.M{"_id": id, "meeting_count": bson.M{"$lt": 4}}
	update := bson.M{"$inc": bson.M{"meeting_count": 1}} // it did not work
	err := UpdateRecord(filter, update, "Can't Complete Meeting.")

	if err == nil {
		c, _err := ReadCandidate(id)

		if c.Meeting_count == 3 {
			ceoId := FindAssigneeIDByName(ceo)
			filter := bson.M{"_id": id, "meeting_count": bson.M{"$eq": 3}}
			update := bson.M{"$set": bson.M{"assignee": ceoId}}
			err := UpdateRecord(filter, update, "Can't Complete Meeting.")
			return err
		}
		return _err
	}
	return err
}

func HandleCompleteMeeting(w http.ResponseWriter, r *http.Request) {
	LogRouterPath(r)
	obj, err := ParseBodyId(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		err := CompleteMeeting(obj.Id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}

func AcceptCandidate(id string) error {
	filter := bson.M{"_id": id, "meeting_count": bson.M{"$eq": 4}}
	update := bson.M{"$set": bson.M{"status": accept}}
	err := UpdateRecord(filter, update, "Candidate can't find or meeting_count less than 4.")
	return err
}

func HandleAcceptCandidate(w http.ResponseWriter, r *http.Request) {
	LogRouterPath(r)
	vars := mux.Vars(r)
	id := vars["id"]
	err := AcceptCandidate(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func DenyCandidate(id string) error {
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"status": denied}}
	err := UpdateRecord(filter, update, "Candidate can't find.")
	return err
}

func HandleDenyCandidate(w http.ResponseWriter, r *http.Request) {
	LogRouterPath(r)
	vars := mux.Vars(r)
	id := vars["id"]
	err := DenyCandidate(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func FindAssigneeIDByName(name string) string {

	collection := db.client.Database("Otsimo").Collection("Assignees")
	filter := bson.M{"name": name}
	var result Assignee
	err := collection.FindOne(context.TODO(), filter).Decode(&result)
	_ = err
	return result.Id
}

func HandleFindAssigneeIDByName(w http.ResponseWriter, r *http.Request) {
	LogRouterPath(r)
	vars := mux.Vars(r)
	name := vars["name"]
	strName := FindAssigneeIDByName(name)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(strName))
}
// db funcs end.
func HandleRequests() {
	r := mux.NewRouter().StrictSlash(true)
	hf := r.HandleFunc

	hf("/", RouterHomePage)

	hf("/crateCandidate", HandleCreateCandidate)
	hf("/readCandidate", HandleReadCandidate)
	hf("/deleteCandidate", HandleDeleteCandidate)

	hf("/arrangeMeeting", HandleArrangeMeeting)
	hf("/completeMeeting", HandleCompleteMeeting)

	hf("/acceptCandidate/{id}", HandleAcceptCandidate)
	hf("/denyCandidate/{id}", HandleDenyCandidate)

	hf("/findAssigneeIDByName/{name}", HandleFindAssigneeIDByName)

	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func main() {

	log.SetPrefix("otsimo_app: ")
	log.SetFlags(0)

	FillDBParams(os.Args)
	CreateDb()
	ConnectDatabase()
	HandleRequests()

}
