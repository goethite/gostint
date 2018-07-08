/*
Copyright 2018 Graham Lee Bevan <graham.bevan@ntlworld.com>

This file is part of goswim.

goswim is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

goswim is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Foobar.  If not, see <https://www.gnu.org/licenses/>.
*/

package pingclean

import (
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

	ids := []string{}
	for _, n := range ns {
		ids = append(ids, n.ID)
	}

	chg := mgo.Change{
		Update:    bson.M{"$set": bson.M{"status": "unknown"}},
		ReturnNew: true,
	}
	var jobs []jobqueues.Job
	_, err = queues.Find(bson.M{
		"node_uuid": bson.M{"$in": ids},
		"status":    "running",
	}).Apply(chg, &jobs)
	if err != nil {
		if err.Error() != "not found" {
			panic(err)
		}
	}

	// clean up nodes
	nodes.RemoveAll(bson.M{"_id": bson.M{"$in": ids}})

	// clean up any queues with ended datetime > 6 hours ago
	now = time.Now()
	threshold = now.Add(time.Duration(-6) * time.Hour)
	queues.RemoveAll(bson.M{
		"ended": bson.M{"$lt": threshold},
	})
}

func interval() {
	for {
		// wait 1 min + random upto another 1 min (splay)
		r := rand.Intn(int(time.Minute))
		time.Sleep(time.Duration(r)*time.Nanosecond + time.Minute)
		wakeup()
	}
}
