// Copyright 2018 John Deng (hi.devops.io@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gotest

import (
	"flag"
	"github.com/hidevopsio/hiboot/pkg/utils/str"
	"os"
	"strings"
)

func IsRunning() bool {

	args := os.Args

	//log.Println("args: ", args)
	//log.Println("args[0]: ", args[0])

	if str.InSlice("-test.v", args) ||
		strings.Contains(args[0], ".test") {
		return true
	}

	return false
}

func ParseArgs(args []string) {

	a := os.Args[1:]
	if args != nil {
		a = args
	}

	flag.CommandLine.Parse(a)
}
