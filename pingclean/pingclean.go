package pingclean

import (
	"log"
	"math/rand"
	"time"

	"github.com/gbevan/goswim/jobqueues"
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
	//   status=unknown
	var ns []Node
	now := time.Now()
	threshold := now.Add(time.Duration(-1) * time.Minute)
	err = nodes.Find(bson.M{
		"last_seen": bson.M{"$lt": threshold},
	}).All(&ns)
	if err != nil {
		panic(err)
	}
	log.Printf("ns: %v", ns)

	ids := []string{}
	for _, n := range ns {
		ids = append(ids, n.ID)
	}
	log.Printf("ids: %v", ids)

	chg := mgo.Change{
		Update:    bson.M{"$set": bson.M{"status": "unknown"}},
		ReturnNew: true,
	}
	var jobs []jobqueues.Job
	_, err = queues.Find(bson.M{"node_uuid": bson.M{"$in": ids}}).Apply(chg, &jobs)
	if err != nil {
		if err.Error() != "not found" {
			panic(err)
		}
	}
	// log.Printf("jobs: %v", jobs)

	// clean up nodes
	nodes.RemoveAll(bson.M{"_id": bson.M{"$in": ids}})
}

func interval() {
	for {
		// wait 1 min + random upto another 1 min (splay)
		r := rand.Intn(int(time.Minute))
		time.Sleep(time.Duration(r)*time.Nanosecond + time.Minute)
		wakeup()
	}
}
