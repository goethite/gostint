/*
Copyright 2018 Graham Lee Bevan <graham.bevan@ntlworld.com>

This file is part of gostint.

gostint is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

gostint is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with gostint.  If not, see <https://www.gnu.org/licenses/>.
*/

package health

import (
	"os"
	"os/signal"
	"strconv"
	"sync"

	"github.com/gbevan/gostint/logmsg"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	. "github.com/visionmedia/go-debug" // nolint
)

var debug = Debug("health")

// State holds the gostint nodes health and state
type State struct {
	State    string
	db       *mgo.Database
	nodeUUID string
}

var (
	state      State
	stateMutex sync.Mutex
)

// Init initialises
func Init(db *mgo.Database, nodeUUID string) {
	stateMutex.Lock()
	state = State{
		State:    "active",
		db:       db,
		nodeUUID: nodeUUID,
	}
	stateMutex.Unlock()

	// SIGINT Handler to drain the node for shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		for {
			sig := <-sigs
			switch sig {
			case os.Interrupt:
				logmsg.Info("SIGINT received, draining node...")
				SetState("draining")
			}
		}
	}()
}

// SetState sets the node's State
func SetState(s string) {
	stateMutex.Lock()
	state.State = s
	stateMutex.Unlock()
}

// GetState Returns the gostint node's state
func GetState() string {
	stateMutex.Lock()
	s := state.State
	stateMutex.Unlock()
	return s
}

// GetHealth Returns the gostint health status
func GetHealth() (*map[string]string, error) {
	m := make(map[string]string)
	m["state"] = GetState()

	db := state.db
	c := db.C("queues")

	num, err := c.Count()
	if err != nil {
		return nil, err
	}
	m["all_jobs"] = strconv.Itoa(num)

	// TODO: replace all below with consolidated MapReduce
	num, err = c.Find(bson.M{
		"status": "queued",
	}).Count()
	if err != nil {
		return nil, err
	}
	m["queued_jobs"] = strconv.Itoa(num)

	num, err = c.Find(bson.M{
		"status": "running",
	}).Count()
	if err != nil {
		return nil, err
	}
	m["running_jobs"] = strconv.Itoa(num)

	num, err = c.Find(bson.M{
		"status": "notauthorised",
	}).Count()
	if err != nil {
		return nil, err
	}
	m["notauthorised_jobs"] = strconv.Itoa(num)

	num, err = c.Find(bson.M{
		"status": "stopping",
	}).Count()
	if err != nil {
		return nil, err
	}
	m["stopping_jobs"] = strconv.Itoa(num)

	num, err = c.Find(bson.M{
		"status": "success",
	}).Count()
	if err != nil {
		return nil, err
	}
	m["success_jobs"] = strconv.Itoa(num)

	num, err = c.Find(bson.M{
		"status": "failed",
	}).Count()
	if err != nil {
		return nil, err
	}
	m["failed_jobs"] = strconv.Itoa(num)

	num, err = c.Find(bson.M{
		"status": "unknown",
	}).Count()
	if err != nil {
		return nil, err
	}
	m["unknown_jobs"] = strconv.Itoa(num)

	return &m, nil
}
