package gofed

import (
	"testing"

	"github.com/graphql-go/graphql"
)

func buildTestSchema() (*graphql.Schema, *graphql.Union) {

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

	var entityType = graphql.NewUnion(graphql.UnionConfig{
		Name: "_Entity",
		Types: []*graphql.Object{
			userType,
		},
		ResolveType: func(p graphql.ResolveTypeParams) *graphql.Object {
			return nil
		},
	})

	var queryType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"user": &graphql.Field{
					Type: userType,
					Args: graphql.FieldConfigArgument{
						"id": &graphql.ArgumentConfig{
							Type: graphql.String,
						},
					},
				},
				//"_service":  serviceField,
				//"_entities": entityField,
			},
		})

	var blahDirective = graphql.NewDirective(graphql.DirectiveConfig{
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
	})

	var schema, _ = graphql.NewSchema(
		graphql.SchemaConfig{
			Query:      queryType,
			Directives: []*graphql.Directive{blahDirective},
		},
	)

	return &schema, entityType
}

func TestSDLPrint(t *testing.T) {

	schema, entityType := buildTestSchema()

	if len(entityType.Types()) != 1 {
		t.Errorf(("entity type is invalid"))
	}

	//sdl, err := printSDL(schema, enentityType)
	printSDL(schema, entityType)

	//fmt.Fprintln(os.Stdout, result)
	//if !reflect.DeepEqual(result.Data, expected) {
	//	t.Errorf("wrong result, query: %v, graphql result diff: %v", query, testutil.Diff(expected, result))
	//}
}
