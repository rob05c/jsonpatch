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

func Apply(patch JSONPatch, realObj interface{}) error {
	obj := reflect.ValueOf(realObj)
	if obj.Kind() != reflect.Ptr {
		return errors.New("object must be a pointer")
	}
	if obj.IsNil() {
		return errors.New("object must not be nil")
	}
	obj = reflect.Indirect(obj)
	for _, patchOp := range patch {
		if err := applyOp(obj, patchOp); err != nil {
			return err
		}
	}
	return nil
}

func applyOp(obj reflect.Value, patchOp JSONPatchOp) error {
	// fmt.Printf("DEBUG Apply oPath.Type().Name() '%+v'\n", oPath.Type().Name())

	switch patchOp.Op {
	case OpTypeAdd:
		if err := applyAdd(obj, patchOp.Path, patchOp.Value); err != nil {
			return err
		}
	case OpTypeRemove:
		if err := applyRemove(obj, patchOp.Path); err != nil {
			return err
		}
	case OpTypeReplace:
		if err := applyReplace(obj, patchOp.Path, patchOp.Value); err != nil {
			return err
		}
	case OpTypeMove:
		if err := applyMove(obj, patchOp.Path, patchOp.From); err != nil {
			return err
		}
	case OpTypeCopy:
		return errors.New("not implemented")
	case OpTypeTest:
		return errors.New("not implemented")
	default:
		return errors.New("unknown op type")
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
	// fmt.Printf("DEBUG getValBefore pathParts: %+v\n", pathParts)

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
		return reflect.Value{}, errors.New("interfaces aren't supported yet")

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
		return reflect.Value{}, errors.New("interfaces aren't supported yet")

	case reflect.Map:
		keyVal, err := ConvertKeyToType(key, obj.Type().Key())
		if err != nil {
			return reflect.Value{}, err
		}
		if !keyVal.CanSet() {
			return reflect.Value{}, fmt.Errorf("can't set value at new key reflect.Value kind %+v", keyVal.Kind().String()) // TODO remove? Necessary?
		}
		mapVal := obj.MapIndex(keyVal)
		zeroValue := reflect.Value{}
		if mapVal != zeroValue {
			// fmt.Println("DEBUG mapVal not zeroValue: returning")

			if !mapVal.CanSet() {
				return reflect.Value{}, fmt.Errorf("can't set map value at '%+v'", keyVal.Interface())
			}
			return mapVal, nil
		}

		if add {
			mapVal = reflect.Zero(obj.Type().Elem())
			obj.SetMapIndex(keyVal, mapVal)
		} else {
			return reflect.Value{}, errors.New("map has no key '" + key + "'")
		}
	}
	return reflect.Value{}, fmt.Errorf("obj has no object or slice at '%+v'", key)
}

// ConvertKeyToType converts a path part of the JSON Pointer op path, to a reflect.Value of a map's key type.
// Returns an error, if the key type is not supported as a JSON Patch map type.
// Supported types are: strings, and integers.
func ConvertKeyToType(key string, keyType reflect.Type) (reflect.Value, error) {
	keyVal := reflect.Indirect(reflect.New(keyType)) // TODO determine if there's a faster way
	switch keyType.Kind() {
	case reflect.String:
		keyVal.SetString(key)
		return keyVal, nil
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		keyI, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			return reflect.Value{}, errors.New("object at key is a map[int], but " + key + " is not an integer")
		}
		keyVal.SetInt(keyI)
		return keyVal, nil
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		fallthrough
	case reflect.Uintptr:
		keyI, err := strconv.ParseUint(key, 10, 64)
		if err != nil {
			return reflect.Value{}, errors.New("object at key is a map[uint], but " + key + " is not a positive integer")
		}
		keyVal.SetUint(keyI)
		return keyVal, nil
	default:
		// TODO support encoding.TextMarshaler
		// if obj.Key().Implements(reflect.TypeOf(encoding.TextMarshaler)) {
		// 	mapKeys := obj.MapKeys()
		// 	for _, mapKey := range mapKeys {
		// 		if mapKey.CanInterface() {
		// 			mapKeyI := mapKey.Interface()
		// 			mapKeyTextMarshaler := mapKey.(encoding.TextMarshaler)
		// 			mapKeyTextBts, err := mapKeyTextMarshaler.MarshalText()
		// 			if err != nil {
		// 				continue // TODO error?
		// 			}
		// 			if mapVal
		// 		}
		// 	}
		// }
		return reflect.Value{}, errors.New("map key type " + keyType.Kind().String() + " not supported; map keys must be strings, integers")
	}
}

// applyAdd performs a JSON Patch add op to obj at pathToken with patchValue.
func applyAdd(obj reflect.Value, path string, patchVal interface{}) error {
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 1 {
		return fmt.Errorf("malformed patch op path: %+v", pathParts)
	}
	lastPathPart := pathParts[len(pathParts)-1]

	obj, err := getValBefore(path, obj)
	if err != nil {
		return errors.New("getValBefore: " + err.Error())
	}

	pathToken := lastPathPart

	if obj.Kind() == reflect.Map {
		return applyAddMap(obj, pathToken, patchVal)
	} else {
		return applyAddGeneric(obj, pathToken, patchVal)
	}
}

// applyAddMap performs a JSON Patch add op to obj at pathToken with patchValue.
// Map values aren't addressable, so they need special logic
func applyAddMap(obj reflect.Value, pathToken string, patchValue interface{}) error {
	if !obj.CanSet() {
		return errors.New("can't set value of map at path " + pathToken)
	}
	objKey, err := ConvertKeyToType(pathToken, obj.Type().Key())
	if err != nil {
		return err
	}
	obj.SetMapIndex(objKey, reflect.ValueOf(patchValue))
	return nil
}

// applyAddMap performs a JSON Patch add op to obj at pathToken with patchValue.
// This func applies to all objects, except maps, which should use applyAddMap
func applyAddGeneric(obj reflect.Value, pathToken string, patchVal interface{}) error {
	objVal, err := getNextVal(pathToken, obj, true)
	if err != nil {
		return errors.New("getting or creating last value in add op: " + err.Error())
	}
	// fmt.Printf("DEBUG Apply reflect.TypeOf(patchOp.Value) %+v\n", reflect.TypeOf(patchOp.Value))
	// fmt.Printf("DEBUG Apply objVal.Type().Name() '%+v'\n", objVal.Type().Name())
	// fmt.Printf("DEBUG Apply objVal.Kind() '%+v'\n", objVal.Kind())
	if objVal.Kind() == reflect.Ptr {
		// fmt.Printf("DEBUG Apply objVal.Type().Elem().Kind() '%+v'\n", objVal.Type().Elem().Kind())
		if objVal.IsNil() {
			objVal.Set(reflect.New(objVal.Type().Elem()))
		}
		objVal = reflect.Indirect(objVal)
	}

	if !objVal.CanSet() {
		return errors.New("can't set value at path " + pathToken)
	}
	if objVal.Type() != reflect.TypeOf(patchVal) {
		// TODO add interface support
		return fmt.Errorf("can't set object field '%+v' to patch value type %T\n", objVal.Type().Name(), patchVal)
	}
	objVal.Set(reflect.ValueOf(patchVal))
	return nil
}

// applyRemoveMap applies a JSON Patch remove op to the given object at the given path token.
func applyRemove(obj reflect.Value, path string) error {
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 1 {
		return fmt.Errorf("malformed patch op path: %+v", pathParts)
	}
	lastPathPart := pathParts[len(pathParts)-1]

	obj, err := getValBefore(path, obj)
	if err != nil {
		return errors.New("getValBefore: " + err.Error())
	}

	pathToken := lastPathPart

	// TODO add slice/array remove
	if obj.Kind() == reflect.Map {
		return applyRemoveMap(obj, pathToken)
	} else {
		return applyRemoveGeneric(obj, pathToken)
	}
}

// applyRemoveMap applies a JSON Patch remove op to the given object at the given path token.
// Map values aren't addressable, so they need special logic
func applyRemoveMap(obj reflect.Value, pathToken string) error {
	objKey, err := ConvertKeyToType(pathToken, obj.Type().Key())
	if err != nil {
		return err
	}
	// TODO error if key doesn't exist, per RFC6902§4.2
	obj.SetMapIndex(objKey, reflect.Value{}) // deletes the key
	return nil
}

// applyRemoveMap applies a JSON Patch remove op to the given object at the given path token.
// Applies to all types except maps, which must call applyRemoveMap because they need special logic.
func applyRemoveGeneric(obj reflect.Value, pathToken string) error {
	objVal, err := getNextVal(pathToken, obj, true)
	if err != nil {
		return errors.New("getting or creating last value in remove op: " + err.Error())
	}
	if !objVal.CanSet() {
		return errors.New("can't set value at path " + pathToken)
	}
	objVal.Set(reflect.Zero(objVal.Type()))
	return nil
}

// applyAdd performs a JSON Patch add op to obj at pathToken with patchValue.
func applyReplace(obj reflect.Value, path string, patchVal interface{}) error {
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 1 {
		return fmt.Errorf("malformed patch op path: %+v", pathParts)
	}
	lastPathPart := pathParts[len(pathParts)-1]

	obj, err := getValBefore(path, obj)
	if err != nil {
		return errors.New("getValBefore: " + err.Error())
	}

	pathToken := lastPathPart

	if obj.Kind() == reflect.Map {
		return applyReplaceMap(obj, pathToken, patchVal)
	} else {
		return applyReplaceGeneric(obj, pathToken, patchVal)
	}

}

func applyReplaceMap(obj reflect.Value, pathToken string, patchVal interface{}) error {
	// map values aren't addressable, so they need special logic
	if !obj.CanSet() {
		return errors.New("can't set value of map at path " + pathToken)
	}
	objKey, err := ConvertKeyToType(pathToken, obj.Type().Key())
	if err != nil {
		return err
	}
	if obj.MapIndex(objKey) == (reflect.Value{}) {
		return errors.New("no value to replace at path " + pathToken)
	}
	obj.SetMapIndex(objKey, reflect.ValueOf(patchVal))
	return nil
}

func applyReplaceGeneric(obj reflect.Value, pathToken string, patchVal interface{}) error {
	obj, err := getNextVal(pathToken, obj, false)
	if err != nil {
		return errors.New("getting last value in add op: " + err.Error())
	}
	if obj.Kind() == reflect.Ptr { // TODO: for loop? Allow multiple pointers?
		obj = reflect.Indirect(obj)
	}

	if !obj.CanSet() {
		return errors.New("can't set value at path " + pathToken)
	}
	if obj.Type() != reflect.TypeOf(patchVal) {
		// fmt.Printf("DEBUG Apply reflect.TypeOf(patchVal) %+v\n", reflect.TypeOf(patchVal))
		// fmt.Printf("DEBUG Apply obj.Type().Name() '%+v'\n", obj.Type().Name())
		// TODO add interface support
		return fmt.Errorf("can't set object field '%+v' to patch value type %T\n", obj.Type().Name(), patchVal)
	}
	obj.Set(reflect.ValueOf(patchVal))
	return nil
}

func applyMove(obj reflect.Value, path string, fromPath string) error {
	if strings.HasPrefix(path, fromPath) {
		if path == fromPath {
			return nil // proper prefixes are allowed, per RFC RFC6902§4.4, and moving to the same place is a no-op.
		}
		return errors.New("move op 'from' cannot be a proper prefix of the 'path' to move into.")
	}

	fromPathParts := strings.Split(fromPath, "/")
	if len(fromPathParts) < 1 {
		return fmt.Errorf("malformed patch op from path: %+v", fromPathParts)
	}
	lastFromPathPart := fromPathParts[len(fromPathParts)-1]

	fromObjBefore, err := getValBefore(fromPath, obj)
	if err != nil {
		return errors.New("getValBefore from: " + err.Error())
	}

	fromPathToken := lastFromPathPart

	fromObj, err := getNextVal(fromPathToken, fromObjBefore, false)
	if err != nil {
		return errors.New("getting last from value in move op: " + err.Error())
	}

	// if obj.Kind() == reflect.Ptr { // TODO: for loop? Allow multiple pointers?
	// 	obj = reflect.Indirect(obj)
	// }

	// if !fromObj.CanSet() {
	// 	return errors.New("can't set value at from path " + fromPathToken)
	// }

	pathParts := strings.Split(path, "/")
	if len(pathParts) < 1 {
		return fmt.Errorf("malformed patch op path: %+v", pathParts)
	}
	lastPathPart := pathParts[len(pathParts)-1]

	obj, err = getValBefore(path, obj)
	if err != nil {
		return errors.New("getValBefore: " + err.Error())
	}

	pathToken := lastPathPart

	obj, err = getNextVal(pathToken, obj, true)
	if err != nil {
		return errors.New("getting last from value in move op: " + err.Error())
	}

	if !obj.CanSet() {
		return errors.New("move can't set value at path " + pathToken)
	}

	// if the 'from' is a pointer and the 'path' isn't, or vica-versa, make the 'from' match the 'path'.
	if fromObj.Type().Kind() == reflect.Ptr && obj.Type().Kind() != reflect.Ptr {
		fromObj = reflect.Indirect(fromObj)
	} else if obj.Type().Kind() == reflect.Ptr && fromObj.Type().Kind() != reflect.Ptr {
		// make a new pointer
		// TODO test, with a map entry (which isn't addressable)
		newFrom := reflect.New(fromObj.Type())
		newFromVal := reflect.Indirect(newFrom)
		newFromVal.Set(fromObj)
		fromObj = newFromVal.Addr()
	}

	if fromObj.Type() != obj.Type() {
		// TODO add interface support
		return fmt.Errorf("can't set path '%+v' to from '%+v'\n", obj.Type().Name(), fromObj.Type().Name())
	}
	obj.Set(fromObj)

	err = applyRemove(fromObjBefore, fromPathToken)

	return nil
}
