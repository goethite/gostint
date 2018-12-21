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

package state

import (
	"os"
	"os/signal"
	"sync"

	"github.com/gbevan/gostint/logmsg"
	. "github.com/visionmedia/go-debug" // nolint
)

var debug = Debug("state")

// State holds the gostint nodes health and state
type State struct {
	State string
	// db       *mgo.Database
	nodeUUID string
}

var (
	state      State
	stateMutex sync.Mutex
)

// Init initialises
func Init(nodeUUID string) {
	stateMutex.Lock()
	state = State{
		State: "active",
		// db:       db,
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
