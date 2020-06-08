// Copyright 2020 TiKV Project Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"math/rand"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/tikv/tikv-operator/cmd/tikv-controller-manager/app"
	"k8s.io/component-base/logs"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	command := app.NewControllerManagerCommand()

	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
