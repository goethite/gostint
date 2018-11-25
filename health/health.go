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
	"fmt"
	"strconv"

	"github.com/gbevan/gostint/jobqueues"
	"github.com/gbevan/gostint/logmsg"
	"github.com/gbevan/gostint/state"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	. "github.com/visionmedia/go-debug" // nolint
)

var debug = Debug("health")

// Health holds props
type Health struct {
	db *mgo.Database
}

var (
	health Health
	// stateMutex sync.Mutex
)

// Init the health module
func Init(db *mgo.Database) {
	health = Health{
		db: db,
	}
}

// GetHealth Returns the gostint health status
func GetHealth() (*map[string]string, error) {
	m := make(map[string]string)
	m["state"] = state.GetState()

	db := health.db
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

	// Docker Info
	clientAPIVer, dockerInfo, err := jobqueues.GetDockerInfo()
	if err == nil {
		logmsg.Info("client ver %v", clientAPIVer)
	}

	m["containers"] = fmt.Sprintf("%d", dockerInfo.Containers)
	m["containers_running"] = fmt.Sprintf("%d", dockerInfo.ContainersRunning)
	m["containers_paused"] = fmt.Sprintf("%d", dockerInfo.ContainersPaused)
	m["containers_stopped"] = fmt.Sprintf("%d", dockerInfo.ContainersStopped)
	m["images"] = fmt.Sprintf("%d", dockerInfo.Images)
	m["mem_total"] = fmt.Sprintf("%d", dockerInfo.MemTotal)
	m["architecture"] = fmt.Sprintf("%s", dockerInfo.Architecture)
	m["operating_system"] = fmt.Sprintf("%s", dockerInfo.OperatingSystem)
	// m["runtimes"] = fmt.Sprintf("%v", dockerInfo.Runtimes)

	return &m, nil
}
