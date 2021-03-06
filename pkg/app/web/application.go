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

package web

import (
	"errors"
	"fmt"
	"github.com/hidevopsio/hiboot/pkg/app"
	"github.com/hidevopsio/hiboot/pkg/inject"
	"github.com/hidevopsio/hiboot/pkg/log"
	"github.com/hidevopsio/hiboot/pkg/utils/io"
	"github.com/hidevopsio/hiboot/pkg/utils/reflector"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"os"
	"reflect"
	"regexp"
)

const (
	pathSep = "/"

	beforeMethod = "Before"
	afterMethod  = "After"
)

// Application is the struct of web Application
type application struct {
	app.BaseApplication
	webApp     *iris.Application
	jwtEnabled bool
	//anonControllers []interface{}
	//jwtControllers  []interface{}
	controllers   []interface{}
	dispatcher    dispatcher
	controllerMap map[string][]interface{}
}

var (
	// controllers global controllers container
	registeredControllers []interface{}
	compiledRegExp        = regexp.MustCompile(`\{(.*?)\}`)

	// ErrControllersNotFound controller not found
	ErrControllersNotFound = errors.New("[app] controllers not found")

	// ErrInvalidController invalid controller
	ErrInvalidController = errors.New("[app] invalid controller")
)

// SetProperty set application property
func (a *application) SetProperty(name string, value interface{}) app.Application {
	a.BaseApplication.SetProperty(name, value)
	return a
}

// Initialize init application
func (a *application) Initialize() error {
	return a.BaseApplication.Initialize()
}

// Run run web application
func (a *application) Run() (err error) {
	serverPort := ":8080"
	conf := a.SystemConfig()
	if conf != nil && conf.Server.Port != "" {
		serverPort = fmt.Sprintf(":%v", conf.Server.Port)
	}

	err = a.build()
	if err != nil {
		return
	}

	err = a.webApp.Run(iris.Addr(fmt.Sprintf(serverPort)), iris.WithConfiguration(defaultConfiguration()))
	return
}

func (a *application) add(controllers ...interface{}) (err error) {
	for _, controller := range controllers {
		ctrl := controller
		// TODO: should run it before register
		if reflect.TypeOf(controller).Kind() == reflect.Func {
			ctrl, err = inject.IntoFunc(controller)
			if err != nil {
				return
			}
		}
		ifcField := reflector.GetEmbeddedInterfaceField(ctrl)
		if ifcField.Anonymous {
			ctrlTypeName := ifcField.Name
			ctrls := a.controllerMap[ctrlTypeName]
			a.controllerMap[ctrlTypeName] = append(ctrls, ctrl)
		}
	}
	return
}

// Init init web application
func (a *application) build(controllers ...interface{}) (err error) {
	a.PrintStartupMessages()

	systemConfig := a.SystemConfig()
	if systemConfig != nil {
		log.SetLevel(systemConfig.Logging.Level)
		log.Infof("Starting Hiboot web application %v on localhost with PID %v", systemConfig.App.Name, os.Getpid())
		log.Infof("Working directory: %v", a.WorkDir)
		log.Infof("The following profiles are active: %v, %v", systemConfig.App.Profiles.Active, systemConfig.App.Profiles.Include)
	}

	f := a.ConfigurableFactory()
	f.SetInstance("applicationContext", a)

	// build auto configurations
	a.BuildConfigurations()

	// The only one Required:
	// here is how you define how your own context will
	// be created and acquired from the iris' generic context pool.
	a.webApp.ContextPool.Attach(func() context.Context {
		return &Context{
			// Optional Part 3:
			Context: context.NewContext(a.webApp),
		}
	})

	// categorize controllers
	a.controllerMap = make(map[string][]interface{})
	err = a.add(a.controllers...)
	if err != nil {
		return
	}

	// first register anon controllers
	a.RegisterController(new(AnonController))

	// call AfterInitialization with factory interface
	a.AfterInitialization()
	return err
}

// RegisterController register controller, e.g. web.Controller, jwt.Controller, or other customized controller
func (a *application) RegisterController(controller interface{}) error {
	// get from controller map
	// parse controller type
	controllerInterfaceName, err := reflector.GetName(controller)
	if err != nil {
		return ErrInvalidController
	}
	controllers, ok := a.controllerMap[controllerInterfaceName]
	if ok {
		return a.dispatcher.register(a.webApp, controllers)
	}
	return ErrControllersNotFound
}

// Use apply middleware
func (a *application) Use(handlers ...context.Handler) {
	// pass user's instances
	for _, hdl := range handlers {
		a.webApp.Use(hdl)
	}
}

func (a *application) initialize(controllers ...interface{}) (err error) {
	io.EnsureWorkDir(3, "config/application.yml")

	// new iris app
	a.webApp = iris.New()

	err = a.Initialize()
	if err == nil {
		if len(controllers) == 0 {
			a.controllers = registeredControllers
		} else {
			a.controllers = controllers
		}
	}
	return
}

// Add add controller to controllers container
func RestController(controllers ...interface{}) {
	registeredControllers = append(registeredControllers, controllers...)
}

// NewApplication create new web application instance and init it
func NewApplication(controllers ...interface{}) app.Application {
	log.SetLevel("error") // set debug level to error first
	a := new(application)
	err := a.initialize(controllers...)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	return a
}
