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

package inject

import (
	"errors"
	"fmt"
	"github.com/hidevopsio/hiboot/pkg/factory"
	"github.com/hidevopsio/hiboot/pkg/log"
	"github.com/hidevopsio/hiboot/pkg/system"
	"github.com/hidevopsio/hiboot/pkg/utils/cmap"
	"github.com/hidevopsio/hiboot/pkg/utils/io"
	"github.com/hidevopsio/hiboot/pkg/utils/reflector"
	"github.com/hidevopsio/hiboot/pkg/utils/str"
	"reflect"
)

const (
	initMethodName = "Init"
)

var (
	// ErrNotImplemented: the interface is not implemented
	ErrNotImplemented = errors.New("[inject] interface is not implemented")

	// ErrInvalidObject: the object is invalid
	ErrInvalidObject = errors.New("[inject] invalid object")

	// ErrInvalidTagName the tag name is invalid
	ErrInvalidTagName = errors.New("[inject] invalid tag name, e.g. exampleTag")

	// ErrSystemConfiguration system is not configured
	ErrSystemConfiguration = errors.New("[inject] system is not configured")

	// ErrInvalidFunc the function is invalid
	ErrInvalidFunc = errors.New("[inject] invalid func")

	// ErrFactoryIsNil factory is invalid
	ErrFactoryIsNil = errors.New("[inject] factory is nil")

	tagsContainer []Tag

	//instancesMap cmap.ConcurrentMap
	appFactory factory.ConfigurableFactory
)

// SetFactory set factory from app
func SetFactory(f factory.ConfigurableFactory) {
	//if fct == nil {
	appFactory = f
	//}
}

// AddTag add new tag
func AddTag(tag Tag) {
	tagsContainer = append(tagsContainer, tag)
}

func getInstanceByName(name string, instType reflect.Type) (inst interface{}) {
	name = str.ToLowerCamel(name)
	if appFactory != nil {
		inst = appFactory.GetInstance(name)
	}
	return
}

func saveInstance(name string, inst interface{}) error {
	name = str.LowerFirst(name)
	if appFactory == nil {
		return ErrFactoryIsNil
	}
	return appFactory.SetInstance(name, inst)
}

// DefaultValue injects instance into the tagged field with `inject:"instanceName"`
func DefaultValue(object interface{}) error {
	return IntoObjectValue(reflect.ValueOf(object), new(defaultTag))
}

// IntoObject injects instance into the tagged field with `inject:"instanceName"`
func IntoObject(object interface{}) error {
	return IntoObjectValue(reflect.ValueOf(object))
}

// IntoObjectValue injects instance into the tagged field with `inject:"instanceName"`
func IntoObjectValue(object reflect.Value, tags ...Tag) error {
	var err error

	// TODO refactor IntoObject
	if appFactory == nil {
		return ErrSystemConfiguration
	}

	obj := reflector.Indirect(object)
	if obj.Kind() != reflect.Struct {
		log.Errorf("[inject] object: %v, kind: %v", object, obj.Kind())
		return ErrInvalidObject
	}

	var targetTags []Tag
	if len(tags) != 0 {
		targetTags = tags
	} else {
		targetTags = tagsContainer
	}
	sc := appFactory.GetInstance("systemConfiguration")
	if sc == nil {
		return ErrSystemConfiguration
	}
	systemConfig := sc.(*system.Configuration)

	cs := appFactory.GetInstance("configurations")
	if cs == nil {
		return ErrSystemConfiguration
	}
	configurations := cs.(cmap.ConcurrentMap)

	// field injection
	for _, f := range reflector.DeepFields(object.Type()) {
		//log.Debugf("parent: %v, name: %v, type: %v, tag: %v", obj.Type(), f.Name, f.Type, f.Tag)
		// check if object has value field to be injected
		var injectedObject interface{}

		ft := f.Type
		if f.Type.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}

		// set field object
		var fieldObj reflect.Value
		if obj.IsValid() && obj.Kind() == reflect.Struct {
			fieldObj = obj.FieldByName(f.Name)
		}

		// TODO: assume that the f.Name of value and inject tag is not the same
		injectedObject = getInstanceByName(f.Name, f.Type)
		if injectedObject == nil {
			for _, tagImpl := range targetTags {
				tagName := reflector.ParseObjectName(tagImpl, "Tag")
				if tagName == "" {
					return ErrInvalidTagName
				}
				tag, ok := f.Tag.Lookup(tagName)
				if ok {
					tagImpl.Init(systemConfig, configurations)
					injectedObject = tagImpl.Decode(object, f, tag)
					if injectedObject != nil {
						if tagImpl.IsSingleton() {
							err := saveInstance(f.Name, injectedObject)
							if err != nil {
								log.Warnf("instance %v is already exist", f.Name)
							}
						}
						// ONLY one tag should be used for dependency injection
						break
					}
				}
			}
		}

		if injectedObject != nil && fieldObj.CanSet() {
			fov := reflect.ValueOf(injectedObject)
			fieldObj.Set(fov)
			log.Debugf("Injected %v.(%v) into %v.%v", injectedObject, fov.Type(), obj.Type(), f.Name)
		}

		//log.Debugf("- kind: %v, %v, %v, %v", obj.Kind(), object.Type(), fieldObj.Type(), f.Name)
		//log.Debugf("isValid: %v, canSet: %v", fieldObj.IsValid(), fieldObj.CanSet())
		filedObject := reflect.Indirect(fieldObj)
		filedKind := filedObject.Kind()
		canNested := filedKind == reflect.Struct
		if canNested && fieldObj.IsValid() && fieldObj.CanSet() && filedObject.Type() != obj.Type() {
			err = IntoObjectValue(fieldObj, tags...)
		}
	}

	// method injection
	// Init, Setter
	method, ok := object.Type().MethodByName(initMethodName)
	if ok {
		numIn := method.Type.NumIn()
		inputs := make([]reflect.Value, numIn)
		inputs[0] = obj.Addr()
		var val reflect.Value
		for i := 1; i < numIn; i++ {
			val, ok = parseMethodInput(method.Type.In(i))
			if ok {
				inputs[i] = val
				//log.Debugf("inType: %v, name: %v, instance: %v", inType, inTypeName, inst)
				//log.Debugf("kind: %v == %v, %v, %v ", obj.Kind(), reflect.Struct, paramValue.IsValid(), paramValue.CanSet())
				paramObject := reflect.Indirect(val)
				if val.IsValid() && paramObject.IsValid() && paramObject.Type() != obj.Type() && paramObject.Kind() == reflect.Struct {
					err = IntoObjectValue(val, tags...)
				}
			} else {
				break
			}
		}
		// finally call Init method to inject
		if ok {
			method.Func.Call(inputs)
		}
	}

	return err
}

func parseMethodInput(inType reflect.Type) (paramValue reflect.Value, ok bool) {
	inType = reflector.IndirectType(inType)
	inTypeName := inType.Name()
	pkgName := io.DirName(inType.PkgPath())
	//log.Debugf("pkg: %v", pkgName)
	inst := getInstanceByName(inTypeName, inType)
	if inst == nil {
		alternativeName := pkgName + inTypeName
		inst = getInstanceByName(alternativeName, inType)
	}
	ok = true
	if inst == nil {
		//log.Debug(inType.Kind())
		switch inType.Kind() {
		// interface and slice creation is not supported
		case reflect.Interface, reflect.Slice:
			ok = false
			break
		default:
			paramValue = reflect.New(inType)
			inst = paramValue.Interface()
			err := saveInstance(inTypeName, inst)
			if err != nil {
				log.Warnf("instance %v is already exist", inTypeName)
			}
		}
	}

	if inst != nil {
		paramValue = reflect.ValueOf(inst)
	}
	return
}

// IntoFunc inject object into func and return instance
func IntoFunc(object interface{}) (retVal interface{}, err error) {
	fn := reflect.ValueOf(object)
	if fn.Kind() == reflect.Func {
		numIn := fn.Type().NumIn()
		inputs := make([]reflect.Value, numIn)
		for i := 0; i < numIn; i++ {
			fnInType := fn.Type().In(i)
			val, ok := parseMethodInput(fnInType)
			if ok {
				inputs[i] = val
			} else {
				return nil, fmt.Errorf("%v is not injected", fnInType.Name())
			}

			paramObject := reflect.Indirect(val)
			if val.IsValid() && paramObject.IsValid() && paramObject.Kind() == reflect.Struct {
				err = IntoObjectValue(val)
			}
		}
		results := fn.Call(inputs)
		if len(results) != 0 {
			return results[0].Interface(), nil
		} else {
			return nil, nil
		}
	}
	return nil, ErrInvalidFunc
}
