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

package cmd

import (
	"github.com/hidevopsio/hiboot/pkg/app/cli"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBarCommands(t *testing.T) {

	testApp := cli.NewTestApplication(t, new(barCommand))

	t.Run("should run bar command", func(t *testing.T) {
		_, err := testApp.RunTest()
		assert.Equal(t, nil, err)
	})

	t.Run("should run baz command", func(t *testing.T) {
		_, err := testApp.RunTest("baz")
		assert.Equal(t, nil, err)
	})

	t.Run("should run buz command", func(t *testing.T) {
		_, err := testApp.RunTest("buz")
		assert.Equal(t, nil, err)
	})
}
