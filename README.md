# Apollo Federation for graphql-go

This is a WIP implementation of Federation using graphql-go.

## Basic usage

`go get github.com/jesse-apollo/gofed`

Create your graphql-go fields map:

``` golang
// declare userType above here
var queryFields = graphql.Fields{
	"user": &graphql.Field{
		Type: userType,
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			// ...
			return nil, nil
		},
	},
}
```

Then generate the schema with:

``` golang
fed := gofed.NewFederation()
schema := fed.BuildSubgraphSchema(queryFields, nil)

http.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
    result := executeQuery(r.URL.Query().Get("query"), schema)
    json.NewEncoder(w).Encode(result)
})
```