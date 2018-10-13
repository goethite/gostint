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
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	. "github.com/visionmedia/go-debug" // nolint
)

var debug = Debug("health")

// State holds the gostint nodes health and state
type State struct {
	State        string `json:"state"`
	AllJobs      int    `json:"all_jobs"`
	QueuedJobs   int    `json:"queued_jobs"`
	RunningJobs  int    `json:"running_jobs"`
	NotAuthJobs  int    `json:"notauthorised_jobs"`
	StoppingJobs int    `json:"stopping_jobs"`
	SuccessJobs  int    `json:"success_jobs"`
	FailedJobs   int    `json:"failed_jobs"`
	UnknownJobs  int    `json:"unknown_jobs"`
	db           *mgo.Database
	nodeUUID     string
}

var state State

// Init initialises
func Init(db *mgo.Database, nodeUUID string) {
	state = State{
		State:    "active",
		db:       db,
		nodeUUID: nodeUUID,
	}
}

// GetState Returns the gostint health status
func GetState() (*State, error) {
	db := state.db
	c := db.C("queues")

	num, err := c.Count()
	if err != nil {
		return nil, err
	}
	state.AllJobs = num

	num, err = c.Find(bson.M{
		"status": "queued",
	}).Count()
	if err != nil {
		return nil, err
	}
	state.QueuedJobs = num

	num, err = c.Find(bson.M{
		"status": "running",
	}).Count()
	if err != nil {
		return nil, err
	}
	state.RunningJobs = num

	num, err = c.Find(bson.M{
		"status": "notauthorised",
	}).Count()
	if err != nil {
		return nil, err
	}
	state.NotAuthJobs = num

	num, err = c.Find(bson.M{
		"status": "stopping",
	}).Count()
	if err != nil {
		return nil, err
	}
	state.StoppingJobs = num

	num, err = c.Find(bson.M{
		"status": "success",
	}).Count()
	if err != nil {
		return nil, err
	}
	state.SuccessJobs = num

	num, err = c.Find(bson.M{
		"status": "failed",
	}).Count()
	if err != nil {
		return nil, err
	}
	state.FailedJobs = num

	num, err = c.Find(bson.M{
		"status": "unknown",
	}).Count()
	if err != nil {
		return nil, err
	}
	state.UnknownJobs = num

	return &state, nil
}

// SetState sets the node's State
func SetState(s string) {
	state.State = s
}
