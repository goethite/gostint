package pingclean

import (
	"log"
	"math/rand"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	uuid "github.com/satori/go.uuid"
)

type PingClean struct {
	Uuid string
	Db   *mgo.Database
}

var pingClean PingClean

func Init(db *mgo.Database) string {
	pingClean.Db = db
	// Assign this node a uuid
	pingClean.Uuid = uuid.NewV4().String()
	wakeup()

	go interval()

	return pingClean.Uuid
}

type Node struct {
	ID       string    `json:"_id"        bson:"_id"`
	LastSeen time.Time `json:"last_seen"  bson:"last_seen"`
}

func wakeup() {
	log.Println("wakeup")
	// ping db 'nodes' collection using uuid as clean, with current time stamp
	db := pingClean.Db
	nodes := db.C("nodes")
	queues := db.C("queues")

	_, err := nodes.UpsertId(pingClean.Uuid, Node{
		ID:       pingClean.Uuid,
		LastSeen: time.Now(),
	})
	if err != nil {
		panic(err)
	}

	// scan nodes for stale node (no longer pinging)
	//   if stale, set all running jobs in queues for that node's uuid to have
	//   status=unknown, rest in queue to aborted
	var ns []Node
	err = nodes.Find(bson.M{
		"last_seen": bson.M{"$lt": time.Now().Sub(10 * time.Minute)},
	}).All(&ns)

	// var jobs []jobqueues.Job
	// err = queues.Find(bson.M{
	// 	"status": "running",
	// }).All(&jobs)
	// if err != nil {
	// 	log.Printf("Find stale queues error: %s", err)
	// }
	// log.Printf("Running: %v", jobs)
}

func interval() {
	// wait 1 min + random upto another 1 min (splay)
	r := rand.Intn(int(time.Minute))
	time.Sleep(time.Duration(r)*time.Nanosecond + time.Minute)
	wakeup()
}
