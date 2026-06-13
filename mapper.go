package shorten

import (
	"fmt"
	"reflect"
	"sync"
)

var structFieldMapping sync.Map

func getStructMapping(structType reflect.Type) map[string][]int {
	if structType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("mapper: expects type %s to be struct", structType))
	}

	if idx, found := structFieldMapping.Load(structType); found {
		return idx.(map[string][]int)
	}
	index := parseStructMapping(structType)
	structFieldMapping.Store(structType, index)
	return index
}

func parseStructMapping(structType reflect.Type) map[string][]int {
	fields := make(map[string][]int)
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		name := field.Name
		if tagName := field.Tag.Get("ino"); len(tagName) != 0 {
			name = tagName
		}

		switch {
		case name == "-", len(field.PkgPath) != 0 && !field.Anonymous:
			continue
		case field.Anonymous:
			if field.Type.Kind() == reflect.Struct {
				for k, idx := range parseStructMapping(field.Type) {
					fields[k] = append(field.Index, idx...)
				}
			}
		default:
			fields[name] = field.Index
		}
	}
	return fields
}
