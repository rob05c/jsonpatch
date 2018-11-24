package jsonpatch

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	// "encoding/json"
)

type OpType string

const (
	OpTypeAdd     = "add"
	OpTypeRemove  = "remove"
	OpTypeReplace = "replace"
	OpTypeMove    = "move"
	OpTypeCopy    = "copy"
	OpTypeTest    = "test"
)

type JSONPatchOp struct {
	Op    OpType
	Path  string
	Value interface{}
	From  string
}

type JSONPatch []JSONPatchOp

/*
TODO:
  interfaces
  pointers
  remove (pointers only?)
  move, copy, test
  benchmark, performance optimizations
*/

func Apply(patch JSONPatch, obj interface{}) error {
	oVal := reflect.ValueOf(obj)

	if oVal.Kind() != reflect.Ptr {
		return errors.New("object must be a pointer")
	}
	if oVal.IsNil() {
		return errors.New("object must not be nil")
	}

	oVal = reflect.Indirect(oVal)

	for _, patchOp := range patch {
		// oPath, err := getValAt(patchOp.Path, oVal)
		// if err != nil {
		// 	return err
		// }

		pathParts := strings.Split(patchOp.Path, "/")
		if len(pathParts) < 1 {
			return fmt.Errorf("malformed patch op path: %+v", pathParts)
		}
		lastPathPart := pathParts[len(pathParts)-1]

		oPath, err := getValBefore(patchOp.Path, oVal)
		if err != nil {
			return errors.New("getValBefore: " + err.Error())
		}
		fmt.Printf("DEBUG Apply oPath.Type().Name() '%+v'\n", oPath.Type().Name())

		switch patchOp.Op {
		case OpTypeAdd:
			oPath, err = getNextVal(lastPathPart, oPath, true)
			if err != nil {
				return errors.New("getting or creating last value in add op: " + err.Error())
			}

			fmt.Printf("DEBUG Apply reflect.TypeOf(patchOp.Value) %+v\n", reflect.TypeOf(patchOp.Value))
			fmt.Printf("DEBUG Apply oPath.Type().Name() '%+v'\n", oPath.Type().Name())
			fmt.Printf("DEBUG Apply oPath.Kind() '%+v'\n", oPath.Kind())
			if oPath.Kind() == reflect.Ptr {
				fmt.Printf("DEBUG Apply oPath.Type().Elem().Kind() '%+v'\n", oPath.Type().Elem().Kind())
				if oPath.IsNil() {
					oPath.Set(reflect.New(oPath.Type().Elem()))
				}
				oPath = reflect.Indirect(oPath)
			}

			if !oPath.CanSet() {
				return errors.New("can't set value at path " + patchOp.Path)
			}
			if oPath.Type() != reflect.TypeOf(patchOp.Value) {
				// TODO add interface support
				return fmt.Errorf("can't set object field '%+v' to patch value type %T\n", oPath.Type().Name(), patchOp.Value)
			}
			oPath.Set(reflect.ValueOf(patchOp.Value))
		case OpTypeRemove:
			oPath, err = getNextVal(lastPathPart, oPath, true)
			if err != nil {
				return errors.New("getting or creating last value in add op: " + err.Error())
			}
			if !oPath.CanSet() {
				return errors.New("can't set value at path " + patchOp.Path)
			}
			oPath.Set(reflect.Zero(oPath.Type()))
		case OpTypeReplace:
			oPath, err = getNextVal(lastPathPart, oPath, false)
			if err != nil {
				return errors.New("getting last value in add op: " + err.Error())
			}
			if oPath.Kind() == reflect.Ptr { // TODO: for loop? Allow multiple pointers?
				oPath = reflect.Indirect(oPath)
			}

			if !oPath.CanSet() {
				return errors.New("can't set value at path " + patchOp.Path)
			}
			if oPath.Type() != reflect.TypeOf(patchOp.Value) {
				fmt.Printf("DEBUG Apply reflect.TypeOf(patchOp.Value) %+v\n", reflect.TypeOf(patchOp.Value))
				fmt.Printf("DEBUG Apply oPath.Type().Name() '%+v'\n", oPath.Type().Name())
				// TODO add interface support
				return fmt.Errorf("can't set object field '%+v' to patch value type %T\n", oPath.Type().Name(), patchOp.Value)
			}
			oPath.Set(reflect.ValueOf(patchOp.Value))
		case OpTypeMove:
			return errors.New("not implemented")
		case OpTypeCopy:
			return errors.New("not implemented")
		case OpTypeTest:
			return errors.New("not implemented")
		}
	}

	return nil
}

// getValAt returns the reflect.Value for the field at the given path of the object.
// If add is true, nil pointers in the object are constructed; otherwise, an error is returned if a member is nil.
func getValAt(path string, obj reflect.Value) (reflect.Value, error) {
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 1 {
		return reflect.Value{}, fmt.Errorf("malformed patch op path: %+v", pathParts)
	}
	pathParts = pathParts[1:]
	err := error(nil)
	for _, part := range pathParts {
		obj, err = getNextVal(part, obj, false)
		if err != nil {
			return reflect.Value{}, err
		}
	}
	return obj, nil
}

// getValBefore gets the reflect.Value immediately preceding the last path. For example, path `/a/b/c` returns `obj.A.B`.
// If add is true, nil pointers in the object are constructed; otherwise, an error is returned if a member is nil.
func getValBefore(path string, obj reflect.Value) (reflect.Value, error) {
	// TODO move split outside these calls, to avoid splitting twice, for this and the following getNextVal for the last val
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 1 {
		return reflect.Value{}, fmt.Errorf("malformed patch op path: %+v", pathParts)
	}
	if len(pathParts) < 2 {
		// path is "/foo", so the object preceding the last is the root object.
		return obj, nil
	}
	pathParts = pathParts[1 : len(pathParts)-1]
	fmt.Printf("DEBUG getValBefore pathParts: %+v\n", pathParts)

	err := error(nil)
	for _, part := range pathParts {
		obj, err = getNextVal(part, obj, false)
		if err != nil {
			return reflect.Value{}, err
		}
	}
	return obj, nil
}

func getNextVal(key string, obj reflect.Value, add bool) (reflect.Value, error) {
	switch obj.Kind() {
	case reflect.Interface:
		// TODO implement; get real value
	case reflect.Struct:
		oType := obj.Type()
		// TODO get field by toLower(name) if no tag exists, to match encoding/json pkg.

		/*
			To unmarshal JSON into a struct, Unmarshal matches incoming object keys to the keys used by Marshal (either the struct field name or its tag), preferring an exact match but also accepting a case-insensitive match. By default, object keys which don't have a corresponding struct field are ignored (see Decoder.DisallowUnknownFields for an alternative).
		*/
		for i := 0; i < obj.NumField(); i++ {
			field := oType.Field(i)
			tag := field.Tag
			if tag.Get("json") == key {
				return obj.Field(i), nil
			}
		}
		return reflect.Value{}, fmt.Errorf("object has no json tag '%+v' (only tags are supported, this library doesn't use field names like encoding/json!)", key)
	case reflect.Slice:
		partI, err := strconv.Atoi(key)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("object at path is an array, but path element is not a number: %+v", key)
		}
		if obj.Len() < partI {
			return reflect.Value{}, fmt.Errorf("object is only %+v long, but path references element %+v", obj.Len(), partI)
		}
		return obj.Index(partI), nil
	case reflect.Array:
		// TODO implement
	}
	return reflect.Value{}, fmt.Errorf("obj has no object or slice at '%+v'", key)
}
