package shorten

import (
	"reflect"
	"testing"
)

func TestParseStructMapping_TagHandling(t *testing.T) {
	type TestStruct struct {
		ID   int    `ino:"user_id"`
		Name string `ino:"-"`
		Age  int
	}

	index := parseStructMapping(reflect.TypeOf(TestStruct{}))

	if idx, ok := index["user_id"]; !ok {
		t.Error("expected 'user_id' from tag")
	} else if len(idx) != 1 {
		t.Errorf("expected flat index for top-level field, got %v", idx)
	}

	if _, ok := index["Name"]; ok {
		t.Error("Name should be ignored due to tag '-'")
	}

	if _, ok := index["Age"]; !ok {
		t.Error("expected 'Age'")
	}
}

func TestParseStructMapping_EmbeddedStruct(t *testing.T) {
	type Address struct {
		City string
	}

	type User struct {
		Address
		Name string
	}

	index := parseStructMapping(reflect.TypeOf(User{}))

	if _, ok := index["City"]; !ok {
		t.Error("expected 'City' from embedded struct")
	}

	if idx, ok := index["City"]; !ok {
		t.Fatal("City not found")
	} else if len(idx) != 2 {
		t.Errorf("expected index [1,0] for City, got %v", idx)
	}
}

func TestParseStructMapping_UnexportedField(t *testing.T) {
	type TestStruct struct {
		Public  string
		private string // unexported
	}

	index := parseStructMapping(reflect.TypeOf(TestStruct{}))

	if _, ok := index["private"]; ok {
		t.Error("unexported field should be ignored")
	}
}

func TestParseStructMapping_EmptyStruct(t *testing.T) {
	type Empty struct{}

	index := parseStructMapping(reflect.TypeOf(Empty{}))
	if len(index) != 0 {
		t.Errorf("empty struct should have empty index map, got %d fields", len(index))
	}
}
