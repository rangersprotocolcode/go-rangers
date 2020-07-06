// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package cli

import (
	"os"
	"os/signal"
	"syscall"
)

func signals() <-chan bool {
	quit := make(chan bool)

	go func() {
		signals := make(chan os.Signal)
		defer close(signals)

		signal.Notify(signals, syscall.SIGQUIT, syscall.SIGTERM, os.Interrupt)
		defer signalStop(signals)

		<-signals
		quit <- true
	}()

	return quit
}

func signalStop(c chan<- os.Signal) {
	signal.Stop(c)
}
