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

package reflector

import (
	"errors"
	"github.com/hidevopsio/hiboot/pkg/utils/io"
	"github.com/hidevopsio/hiboot/pkg/utils/str"
	"reflect"
	"strings"
)

var (
	// ErrInvalidInput means that the input is invalid
	ErrInvalidInput = errors.New("input is invalid")

	// ErrInvalidMethod means that the method is invalid
	ErrInvalidMethod = errors.New("method is invalid")

	// ErrInvalidFunc means that the func is invalid
	ErrInvalidFunc = errors.New("func is invalid")

	// ErrFieldCanNotBeSet means that the field can not be set
	ErrFieldCanNotBeSet = errors.New("field can not be set")
)

func NewReflectType(st interface{}) interface{} {
	ct := reflect.TypeOf(st)
	co := reflect.New(ct)
	cp := co.Elem().Addr().Interface()
	return cp
}

func Validate(toValue interface{}) (*reflect.Value, error) {

	to := Indirect(reflect.ValueOf(toValue))

	// Return is from value is invalid
	if !to.IsValid() {
		return nil, errors.New("value is not valid")
	}

	if !to.CanAddr() {
		return nil, errors.New("value is unaddressable")
	}

	return &to, nil
}

func DeepFields(reflectType reflect.Type) []reflect.StructField {
	var fields []reflect.StructField

	if reflectType = IndirectType(reflectType); reflectType.Kind() == reflect.Struct {
		for i := 0; i < reflectType.NumField(); i++ {
			v := reflectType.Field(i)
			if v.Anonymous {
				fields = append(fields, DeepFields(v.Type)...)
			} else {
				fields = append(fields, v)
			}
		}
	}

	return fields
}

func Indirect(reflectValue reflect.Value) reflect.Value {
	for reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}
	return reflectValue
}

func IndirectType(reflectType reflect.Type) reflect.Type {
	for reflectType.Kind() == reflect.Ptr || reflectType.Kind() == reflect.Slice {
		reflectType = reflectType.Elem()
	}
	return reflectType
}

func GetFieldValue(f interface{}, name string) reflect.Value {
	r := reflect.ValueOf(f)
	fv := reflect.Indirect(r).FieldByName(name)

	return fv
}

func SetFieldValue(object interface{}, name string, value interface{}) error {

	obj := Indirect(reflect.ValueOf(object))

	if !obj.IsValid() {
		return ErrInvalidInput
	}

	if obj.Kind() != reflect.Struct {
		return ErrInvalidInput
	}

	fieldObj := obj.FieldByName(name)

	if !fieldObj.CanSet() {
		return ErrFieldCanNotBeSet
	}

	fov := reflect.ValueOf(value)
	fieldObj.Set(fov)

	//log.Debugf("Set %v.(%v) into %v.%v", value, fov.Type(), obj.Type(), name)
	return nil
}

func GetKind(kind reflect.Kind) reflect.Kind {

	// Check each condition until a case is true.
	switch {

	case kind >= reflect.Int && kind <= reflect.Int64:
		return reflect.Int

	case kind >= reflect.Uint && kind <= reflect.Uint64:
		return reflect.Uint

	case kind >= reflect.Float32 && kind <= reflect.Float64:
		return reflect.Float32

	default:
		return kind
	}
}

func GetKindByValue(val reflect.Value) reflect.Kind {
	return GetKind(val.Kind())
}

func GetKindByType(typ reflect.Type) reflect.Kind {
	return GetKind(typ.Kind())
}

func ValidateReflectType(obj interface{}, callback func(value *reflect.Value, reflectType reflect.Type, fieldSize int, isSlice bool) error) error {
	v, err := Validate(obj)
	if err != nil {
		return err
	}

	t := IndirectType(v.Type())

	isSlice := false
	fieldSize := 1
	if v.Kind() == reflect.Slice {
		isSlice = true
		fieldSize = v.Len()
	}

	if callback != nil {
		return callback(v, t, fieldSize, isSlice)
	}

	return err
}

func GetType(data interface{}) (typ reflect.Type, err error) {
	dv := Indirect(reflect.ValueOf(data))

	// Return is from value is invalid
	if !dv.IsValid() {
		err = ErrInvalidInput
		return
	}
	typ = dv.Type()

	//log.Debugf("%v %v %v %v %v", dv, typ, typ.String(), typ.Name(), typ.PkgPath())
	return
}

func GetName(data interface{}) (name string, err error) {

	typ, err := GetType(data)
	if err == nil {
		name = typ.Name()
	}
	return
}

func GetLowerCaseObjectName(data interface{}) (string, error) {
	name, err := GetName(data)
	name = strings.ToLower(name)
	return name, err
}

func HasField(object interface{}, name string) bool {
	r := reflect.ValueOf(object)
	fv := reflect.Indirect(r).FieldByName(name)

	return fv.IsValid()
}

func CallMethodByName(object interface{}, name string, args ...interface{}) (interface{}, error) {
	objVal := reflect.ValueOf(object)
	method, ok := objVal.Type().MethodByName(name)
	if ok {
		numIn := method.Type.NumIn()
		inputs := make([]reflect.Value, numIn)
		inputs[0] = objVal
		for i, arg := range args {
			inputs[i+1] = reflect.ValueOf(arg)
		}
		results := method.Func.Call(inputs)
		if len(results) != 0 {
			return results[0].Interface(), nil
		} else {
			return nil, nil
		}
	}
	return nil, ErrInvalidMethod
}

func CallFunc(object interface{}, args ...interface{}) (interface{}, error) {
	fn := reflect.ValueOf(object)
	if fn.Kind() == reflect.Func {
		numIn := fn.Type().NumIn()
		inputs := make([]reflect.Value, numIn)
		for i, arg := range args {
			inputs[i] = reflect.ValueOf(arg)
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

func HasEmbeddedField(object interface{}, name string) bool {
	//log.Debugf("HasEmbeddedField: %v", name)
	typ := IndirectType(reflect.TypeOf(object))
	if typ.Kind() != reflect.Struct {
		return false
	}
	field, ok := typ.FieldByName(name)
	return field.Anonymous && ok
}

func GetEmbeddedInterfaceFieldByType(typ reflect.Type) (field reflect.StructField) {
	if typ.Kind() == reflect.Struct {
		for i := 0; i < typ.NumField(); i++ {
			v := typ.Field(i)
			if v.Anonymous {
				if v.Type.Kind() == reflect.Interface {
					return v
				} else {
					return GetEmbeddedInterfaceFieldByType(v.Type)
				}
			}
		}
	}
	return
}

func GetEmbeddedInterfaceField(object interface{}) (field reflect.StructField) {
	if object == nil {
		return
	}
	typ := IndirectType(reflect.TypeOf(object))
	return GetEmbeddedInterfaceFieldByType(typ)
}

// ParseObjectName e.g. ExampleObject => example
func ParseObjectName(obj interface{}, eliminator string) string {
	name, err := GetName(obj)
	if err == nil {
		name = strings.Replace(name, eliminator, "", -1)
		name = str.LowerFirst(name)
	}
	return name
}

// ParseObjectName e.g. ExampleObject => example
func ParseObjectPkgName(obj interface{}) string {

	typ := IndirectType(reflect.TypeOf(obj))
	name := io.DirName(typ.PkgPath())

	return name
}

// GetPkgPath get the package patch
func GetPkgPath(object interface{}) string {
	objType := IndirectType(reflect.TypeOf(object))
	return objType.PkgPath()
}
