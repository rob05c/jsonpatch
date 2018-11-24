package jsonpatch

import (
	"reflect"
	"testing"
)

func TestApply(t *testing.T) {
	type B struct {
		C int `json:"c"`
		D int `json:"d"`
	}
	type A struct {
		B B `json:"b"`
	}
	type TestObj struct {
		A A `json:"a"`
	}

	patch := JSONPatch{
		JSONPatchOp{
			Op:    OpTypeAdd,
			Path:  "/a/b/c",
			Value: 42,
		},
		JSONPatchOp{
			Op:    OpTypeReplace,
			Path:  "/a/b/d",
			Value: 24,
		},
	}

	obj := &TestObj{
		A: A{
			B: B{
				C: 1,
				D: 2,
			},
		},
	}

	if err := Apply(patch, obj); err != nil {
		t.Fatalf("%+v", err)
	}

	if obj.A.B.C != 42 {
		t.Errorf("Apply obj.A.B.C expected %+v actual %+v", 42, obj.A.B.C)
	}
	if obj.A.B.D != 24 {
		t.Errorf("Apply obj.A.B.D expected %+v actual %+v", 24, obj.A.B.D)
	}
}

func TestApplyObj(t *testing.T) {
	type B struct {
		C int `json:"c"`
		D int `json:"d"`
	}
	type A struct {
		B B `json:"b"`
	}
	type TestObj struct {
		A A `json:"a"`
	}

	patch := JSONPatch{
		JSONPatchOp{
			Op:    OpTypeReplace,
			Path:  "/a/b",
			Value: B{C: 23, D: 29},
		},
	}

	obj := &TestObj{
		A: A{
			B: B{
				C: 1,
				D: 2,
			},
		},
	}

	if err := Apply(patch, obj); err != nil {
		t.Fatalf("%+v", err)
	}

	if obj.A.B.C != 23 {
		t.Errorf("Apply obj.A.B.C expected %+v actual %+v", 23, obj.A.B.C)
	}
	if obj.A.B.D != 29 {
		t.Errorf("Apply obj.A.B.D expected %+v actual %+v", 29, obj.A.B.D)
	}
}

func TestApplyArr(t *testing.T) {
	type ArrA struct {
		B []string `json:"b"`
	}

	type TestArr struct {
		A ArrA `json:"a"`
	}

	expected := []string{"apricot", "blackberry", "cherry"}
	patch := JSONPatch{
		JSONPatchOp{
			Op:    OpTypeReplace,
			Path:  "/a/b",
			Value: expected,
		},
	}

	obj := &TestArr{
		A: ArrA{
			B: []string{"foo", "bar", "baz"},
		},
	}

	if err := Apply(patch, obj); err != nil {
		t.Fatalf("%+v", err)
	}

	if !reflect.DeepEqual(obj.A.B, expected) {
		t.Errorf("Apply obj.A.B expected %+v actual %+v", expected, obj.A.B)
	}
}

func TestApplyArrIndexObj(t *testing.T) {
	type ArrA struct {
		B []string `json:"b"`
	}

	type TestArr struct {
		A ArrA `json:"a"`
	}

	patch := JSONPatch{
		JSONPatchOp{
			Op:    OpTypeReplace,
			Path:  "/a/b/2",
			Value: "crabapple",
		},
	}

	obj := &TestArr{
		A: ArrA{
			B: []string{"avacado", "blueberry", "currant", "durian"},
		},
	}

	expected := []string{"avacado", "blueberry", "crabapple", "durian"}

	if err := Apply(patch, obj); err != nil {
		t.Fatalf("%+v", err)
	}

	if !reflect.DeepEqual(obj.A.B, expected) {
		t.Errorf("Apply obj.A.B expected %+v actual %+v", expected, obj.A.B)
	}
}

func TestApplyArrIndex(t *testing.T) {
	type ArrB struct {
		C int `json:"c"`
	}
	type ArrA struct {
		B []ArrB `json:"b"`
	}
	type TestArr struct {
		A ArrA `json:"a"`
	}

	expected := 42

	patch := JSONPatch{
		JSONPatchOp{
			Op:    OpTypeReplace,
			Path:  "/a/b/2/c",
			Value: expected,
		},
	}

	obj := &TestArr{
		A: ArrA{
			B: []ArrB{ArrB{19}, ArrB{23}, ArrB{29}, ArrB{31}},
		},
	}

	if err := Apply(patch, obj); err != nil {
		t.Fatalf("%+v", err)
	}

	if actual := obj.A.B[2].C; expected != actual {
		t.Errorf("Apply obj.A.B[2].C expected %+v actual %+v", expected, actual)
	}
}

// TestApplyObjAddNil tests an 'add' op to an object that's a nil pointer at the add location
func TestApplyObjAddPtr(t *testing.T) {
	type B struct {
		C *int `json:"c"`
	}
	type A struct {
		B B `json:"b"`
	}
	type TestObj struct {
		A A `json:"a"`
	}

	expected := 42

	patch := JSONPatch{
		JSONPatchOp{
			Op:    OpTypeAdd,
			Path:  "/a/b/c",
			Value: expected,
		},
	}

	v := 19
	obj := &TestObj{
		A: A{
			B: B{
				C: &v,
			},
		},
	}

	if err := Apply(patch, obj); err != nil {
		t.Fatalf("%+v", err)
	}

	if obj.A.B.C == nil {
		t.Fatalf("Apply obj.A.B.C expected *%+v actual %+v", expected, nil)
	}
	if actual := *obj.A.B.C; expected != actual {
		t.Errorf("Apply obj.A.B.C expected *%+v actual *%+v", expected, actual)
	}
}

// TestApplyObjAddNil tests an 'add' op to an object that's a nil pointer at the add location
func TestApplyObjReplacePtr(t *testing.T) {
	type B struct {
		C *int `json:"c"`
	}
	type A struct {
		B B `json:"b"`
	}
	type TestObj struct {
		A A `json:"a"`
	}

	expected := 42

	patch := JSONPatch{
		JSONPatchOp{
			Op:    OpTypeAdd,
			Path:  "/a/b/c",
			Value: expected,
		},
	}

	v := 19
	obj := &TestObj{
		A: A{
			B: B{
				C: &v,
			},
		},
	}

	if err := Apply(patch, obj); err != nil {
		t.Fatalf("%+v", err)
	}

	if obj.A.B.C == nil {
		t.Fatalf("Apply obj.A.B.C expected *%+v actual %+v", expected, nil)
	}
	if actual := *obj.A.B.C; expected != actual {
		t.Errorf("Apply obj.A.B.C expected *%+v actual *%+v", expected, actual)
	}
}

// TestApplyObjAddNil tests an 'add' op to an object that's a nil pointer at the add location
func TestApplyObjAddNil(t *testing.T) {
	type B struct {
		C *int `json:"c"`
	}
	type A struct {
		B B `json:"b"`
	}
	type TestObj struct {
		A A `json:"a"`
	}

	expected := 42

	patch := JSONPatch{
		JSONPatchOp{
			Op:    OpTypeAdd,
			Path:  "/a/b/c",
			Value: expected,
		},
	}

	obj := &TestObj{
		A: A{
			B: B{
				C: nil,
			},
		},
	}

	if err := Apply(patch, obj); err != nil {
		t.Fatalf("%+v", err)
	}

	if obj.A.B.C == nil {
		t.Fatalf("Apply obj.A.B.C expected *%+v actual %+v", expected, nil)
	}
	if actual := *obj.A.B.C; expected != actual {
		t.Errorf("Apply obj.A.B.C expected *%+v actual *%+v", expected, actual)
	}
}

// TestApplyObjAddNil tests an 'add' op to an object that's a nil pointer at the add location
func TestApplyObjReplaceNil(t *testing.T) {
	type B struct {
		C *int `json:"c"`
	}
	type A struct {
		B B `json:"b"`
	}
	type TestObj struct {
		A A `json:"a"`
	}

	expected := 42

	patch := JSONPatch{
		JSONPatchOp{
			Op:    OpTypeAdd,
			Path:  "/a/b/c",
			Value: expected,
		},
	}

	obj := &TestObj{
		A: A{
			B: B{
				C: nil,
			},
		},
	}

	if err := Apply(patch, obj); err != nil {
		t.Fatalf("%+v", err)
	}

	if obj.A.B.C == nil {
		t.Fatalf("Apply obj.A.B.C expected *%+v actual %+v", expected, nil)
	}
	if actual := *obj.A.B.C; expected != actual {
		t.Errorf("Apply obj.A.B.C expected *%+v actual *%+v", expected, actual)
	}
}

// TestRemoveObj tests a 'remove' op on a value.
func TestRemoveObj(t *testing.T) {
	type B struct {
		C int `json:"c"`
		D int `json:"d"`
	}
	type A struct {
		B B `json:"b"`
	}
	type TestObj struct {
		A A `json:"a"`
	}

	patch := JSONPatch{
		JSONPatchOp{
			Op:   OpTypeRemove,
			Path: "/a/b/c",
		},
	}

	obj := &TestObj{
		A: A{
			B: B{
				C: 1,
				D: 2,
			},
		},
	}

	if err := Apply(patch, obj); err != nil {
		t.Fatalf("%+v", err)
	}

	if obj.A.B.C != 0 {
		t.Errorf("Apply remove obj.A.B.C expected %+v actual %+v", 0, obj.A.B.C)
	}
	if obj.A.B.D != 2 {
		t.Errorf("Apply obj.A.B.D expected %+v actual %+v", 2, obj.A.B.D)
	}
}

// TestApplyObjRemoveNil tests an 'remove' op on a pointer.
func TestApplyObjRemoveNil(t *testing.T) {
	type B struct {
		C *int `json:"c"`
	}
	type A struct {
		B B `json:"b"`
	}
	type TestObj struct {
		A A `json:"a"`
	}

	expected := 42

	patch := JSONPatch{
		JSONPatchOp{
			Op:   OpTypeRemove,
			Path: "/a/b/c",
		},
	}

	v := expected

	obj := &TestObj{
		A: A{
			B: B{
				C: &v,
			},
		},
	}

	if err := Apply(patch, obj); err != nil {
		t.Fatalf("%+v", err)
	}

	if obj.A.B.C != nil {
		t.Fatalf("Apply obj.A.B.C expected %+v actual *%+v", nil, *obj.A.B.C)
	}
}
