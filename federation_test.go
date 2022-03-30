package gofed

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/graphql-go/graphql"
)

type testUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var testData = []testUser{
	{
		ID:   "1",
		Name: "Bilbo",
	},
	{
		ID:   "2",
		Name: "Frodo",
	},
	{
		ID:   "3",
		Name: "Gandalf",
	},
	{
		ID:   "4",
		Name: "Pippen",
	},
	{
		ID:   "5",
		Name: "Sam",
	},
}

func queryTestDatabase(rep *Representation) (interface{}, error) {

	// lookup in the "user" database if typeName is User
	if rep.TypeName == "User" {
		for _, v := range testData {
			obj := reflect.ValueOf(v)
			field := reflect.Indirect(obj).FieldByName((rep.KeyName))
			// handle field not existing on type
			if field.IsZero() {
				continue
			}

			// switch on keyvalue type
			switch t := rep.KeyValue.(type) {
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
		return nil, fmt.Errorf("entity not found in user database: %s=%s", rep.KeyName, rep.KeyValue)
	}

	return nil, fmt.Errorf("unknown database type: %s", rep.TypeName)
}

func buildUserObject() *graphql.Object {
	var userType = graphql.NewObject(
		graphql.ObjectConfig{
			Name:        "User",
			Description: "A user in the system",
			Fields: graphql.Fields{
				"id": &graphql.Field{
					Type:        graphql.NewNonNull(graphql.String),
					Description: "The database \"ID\".",
				},
				"name": &graphql.Field{
					Type: graphql.String,
				},
				"friends": &graphql.Field{
					Type:        graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.Int))),
					Description: "Friends of this user.",
				},
			},
			Extensions: map[string]interface{}{
				"directives": []*DirectiveValue{
					{
						Name: "key",
						Values: map[string]interface{}{
							"fields": "id",
						},
					},
				},
			},
		},
	)
	return userType
}

func buildSubgraphSchema() *Federation {

	var selfieType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Selfie",
			Fields: graphql.Fields{
				"id": &graphql.Field{
					Type: graphql.String,
				},
				"name": &graphql.Field{
					Type: graphql.String,
				},
			},
		},
	)

	actorInterface := graphql.NewInterface(graphql.InterfaceConfig{
		Name:        "Actor",
		Description: "An actor in the system",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type:        graphql.String,
				Description: "The name of the character.",
			},
		},
	})

	var userType = graphql.NewObject(
		graphql.ObjectConfig{
			Name:        "User",
			Description: "A user in the system",
			Fields: graphql.Fields{
				"id": &graphql.Field{
					Type:        graphql.NewNonNull(graphql.String),
					Description: "The database \"ID\".",
				},
				"name": &graphql.Field{
					Type: graphql.String,
				},
				"selfie": &graphql.Field{
					Type: selfieType,
				},
				"friends": &graphql.Field{
					Type:        graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.Int))),
					Description: "Friends of this user.",
				},
			},
			Extensions: map[string]interface{}{
				"directives": []*DirectiveValue{
					{
						Name: "key",
						Values: map[string]interface{}{
							"fields": "id",
						},
					},
				},
			},
			Interfaces: []*graphql.Interface{
				actorInterface,
			},
		},
	)

	var queryFields = graphql.Fields{
		"user": &graphql.Field{
			Type: userType,
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{
					Type: graphql.String,
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				for _, v := range testData {
					if v.ID == p.Args["id"].(string) {
						return v, nil
					}
				}
				return nil, nil
			},
		},
		"users": &graphql.Field{
			Type: graphql.NewList(userType),
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				return testData, nil
			},
		},
	}

	/*var blahDirective = graphql.NewDirective(graphql.DirectiveConfig{
		Name: "blah",
		Args: map[string]*graphql.ArgumentConfig{
			"fields": {
				Type: graphql.String,
			},
		},
		Locations: []string{
			graphql.DirectiveLocationObject,
			graphql.DirectiveLocationInterface,
		},
	})*/

	fed := NewFederation()
	fed.BuildSubgraphSchema(queryFields, nil)

	return fed
}

func compareStringLines(testA, testB string, t *testing.T) bool {
	areEqual := true

	testASplit := strings.Split(testA, "\n")
	testBSplit := strings.Split(testB, "\n")

	for i := range testASplit {
		if strings.TrimSpace(testASplit[i]) != strings.TrimSpace(testBSplit[i]) {
			t.Logf("Line %d does not match: \n%s\n%s\n", i, testASplit[i], testBSplit[i])
			areEqual = false
		}
	}
	return areEqual
}

func TestGetKey(t *testing.T) {

	obj := buildUserObject()
	keyFields, err := getKeyDirectives(obj)

	if err != nil {
		t.Errorf("error getting keys from object: %s", err)
	}

	if !reflect.DeepEqual(keyFields, []string{"id"}) {
		t.Error("key directives do not match")
	}

}

func TestFedSchema(t *testing.T) {

	fed := buildSubgraphSchema()

	schema := fed.PrintSDL()

	testSchema, err := ioutil.ReadFile("testdata/test_schema1.graphql")
	if err != nil {
		fmt.Println("File reading error", err)
		return
	}

	//fmt.Fprintln(os.Stdout, schema)

	if !compareStringLines(string(testSchema), schema, t) {
		fmt.Fprintln(os.Stdout, schema)
		t.Error("subgraph schemas do not match")
	}

}

func TestResolve(t *testing.T) {

	fed := buildSubgraphSchema()

	// Query
	query := `
		{
			user(id: "1") { id, name }
		}
	`
	params := graphql.Params{Schema: *fed.Schema(), RequestString: query}
	r := graphql.Do(params)
	if len(r.Errors) > 0 {
		t.Errorf("failed to execute graphql operation, errors: %+v", r.Errors)
	}

	rJSON, _ := json.Marshal(r)
	if string(rJSON) != `{"data":{"user":{"id":"1","name":"Bilbo"}}}` {
		t.Errorf("invalid query results")
	}

	// Query 2
	query = `
		{
			users { id, name }
		}
	`
	params = graphql.Params{Schema: *fed.Schema(), RequestString: query}
	r = graphql.Do(params)
	if len(r.Errors) > 0 {
		t.Errorf("failed to execute graphql operation, errors: %+v", r.Errors)
	}

	rJSON, _ = json.Marshal(r)
	if string(rJSON) != `{"data":{"users":[{"id":"1","name":"Bilbo"},{"id":"2","name":"Frodo"},{"id":"3","name":"Gandalf"},{"id":"4","name":"Pippen"},{"id":"5","name":"Sam"}]}}` {
		fmt.Fprintln(os.Stdout, string(rJSON))
		t.Errorf("invalid query results")
	}

}

func TestEntityResolve(t *testing.T) {

	fed := buildSubgraphSchema()

	// Query
	query := `
		query ($_representations: [_Any!]!) {
			_entities(representations: $_representations) { ... on User { id, name } }
		}
	`
	params := graphql.Params{
		Schema:        *fed.Schema(),
		RequestString: query,
		VariableValues: map[string]interface{}{
			"_representations": []map[string]interface{}{
				{"__typename": "User", "id": "1"},
			},
		},
	}
	r := graphql.Do(params)
	if len(r.Errors) > 0 {
		t.Errorf("failed to execute graphql operation, errors: %+v", r.Errors)
	}

	rJSON, _ := json.Marshal(r)
	if string(rJSON) != `{"data":{"user":{"id":"1","name":"Bilbo"}}}` {
		t.Errorf("invalid query results")
	}

}
