package main

import (
	"github.com/graphql-go/graphql"
	"github.com/jesse-apollo/gofed"
)

var userType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "User",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.String,
			},
			"name": &graphql.Field{
				Type: graphql.String,
			},
		},
		Extensions: map[string]interface{}{
			"directives": []*gofed.DirectiveValue{
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

var queryFields = graphql.Fields{
	"user": &graphql.Field{
		Type: userType,
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return databaseFind("User", "id", p.Args["id"])
		},
	},
}
