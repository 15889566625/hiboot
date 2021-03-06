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

package locale

import (
	"github.com/hidevopsio/hiboot/pkg/app"
	"github.com/hidevopsio/hiboot/pkg/log"
	"github.com/hidevopsio/hiboot/pkg/utils/io"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/middleware/i18n"
	"os"
	"path/filepath"
	"strings"
)

type configuration struct {
	app.Configuration
	Properties         Properties `mapstructure:"locale"`
	applicationContext app.ApplicationContext
}

func newConfiguration(applicationContext app.ApplicationContext) *configuration {
	return &configuration{
		applicationContext: applicationContext,
	}
}

func init() {
	app.AutoConfiguration(newConfiguration)
}

func (c *configuration) LocaleHandler() context.Handler {
	// TODO: localePath should be configurable in application.yml
	// locale:
	//   en-US: ./config/i18n/en-US.ini
	//   cn-ZH: ./config/i18n/cn-ZH.ini
	// TODO: or
	// locale:
	//   path: ./config/i18n/
	localePath := c.Properties.LocalePath
	if io.IsPathNotExist(localePath) {
		return nil
	}

	// parse language files
	languages := make(map[string]string)
	err := filepath.Walk(localePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		//*files = append(*files, path)
		lng := strings.Replace(path, localePath, "", 1)
		lng = io.BaseDir(lng)
		lng = io.Basename(lng)

		if lng != "" && path != localePath+lng {
			//languages[lng] = path
			if languages[lng] == "" {
				languages[lng] = path
			} else {
				languages[lng] = languages[lng] + ", " + path
			}
			//log.Debugf("%v, %v", lng, languages[lng])
		}
		return nil
	})
	if err != nil {
		return nil
	}

	globalLocale := i18n.New(i18n.Config{
		Default:      c.Properties.Default,
		URLParameter: c.Properties.URLParameter,
		Languages:    languages,
	})

	c.applicationContext.Use(globalLocale)

	return globalLocale
}
