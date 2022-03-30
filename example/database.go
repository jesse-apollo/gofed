package main

import (
	"fmt"
	"reflect"
)

type user struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var data = map[string]user{
	"1": {
		ID:   "1",
		Name: "Bilbo",
	},
	"2": {
		ID:   "2",
		Name: "Frodo",
	},
	"3": {
		ID:   "3",
		Name: "Gandalf",
	},
	"4": {
		ID:   "4",
		Name: "Pippen",
	},
	"5": {
		ID:   "5",
		Name: "Sam",
	},
}

// databaseFind - query the database for the entity specified by typeName and keyName/keyValue
func databaseFind(typeName, keyName string, keyValue interface{}) (interface{}, error) {

	// lookup in the "user" database if typeName is User
	if typeName == "User" {
		for _, v := range data {
			obj := reflect.ValueOf(v)
			field := reflect.Indirect(obj).FieldByName((keyName))
			// handle field not existing on type
			if field.IsZero() {
				continue
			}

			// switch on keyvalue type
			switch t := keyValue.(type) {
			case string:
				if field.String() == t {
					return v, nil
				}
			case int64:
				if field.Int() == t {
					return v, nil
				}
			case float64:
				if field.Float() == t {
					return v, nil
				}
			default:
				return nil, fmt.Errorf("cannot determin type of key value: %s", t)
			}

		}
		return nil, fmt.Errorf("entity not found in user database: %s=%s", keyName, keyValue)
	}

	return nil, fmt.Errorf("unknown database type: %s", typeName)
}
