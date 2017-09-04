// Copyright 2017 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package main

import (
	"flag"
	"os"
	"os/signal"
	"testing"

	logt "github.com/mendersoftware/go-lib-micro/log/testing"
)

var runAcceptanceTests bool

func init() {

	logt.MaybeDiscardLogs()

	flag.BoolVar(&runAcceptanceTests, "acceptance-tests", false, "set flag when running acceptance tests")
	flag.Parse()
}

func TestRunMain(t *testing.T) {
	if !runAcceptanceTests {
		t.Skip()
	}

	go main()

	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	<-stopChan
}
