package gofed

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

type DirectiveValue struct {
	Name   string
	Values map[string]interface{}
}

type Representation struct {
	TypeName string
	KeyName  string
	KeyValue interface{}
}

type EntityResolverFn func(rep *Representation) (interface{}, error)
type EntityResolverBatchFn func(reps []*Representation) ([]interface{}, error)

type Federation struct {
	entityResolver      EntityResolverFn
	batchEntityResolver EntityResolverBatchFn
	schema              *graphql.Schema
	entityType          *graphql.Union
	objects             map[string]*graphql.Object
	interfaces          map[string]*graphql.Interface
}

func NewFederation() *Federation {
	return &Federation{}
}

func (f *Federation) resolveEntity(p graphql.ResolveParams) (interface{}, error) {
	_, isOK := p.Args["representations"].([]map[string]interface{})

	if !isOK {
		//fmt.Fprintln(os.Stdout, fmt.Sprintf("reps: %v", p.Args["representations"]))

		return nil, fmt.Errorf("invalid representations")
	}

	return nil, nil
	/*reps, isOK := p.Args["representations"].([]map[string]interface{})

	if !isOK {
		return nil, fmt.Errorf("invalid representations")
	}

	var keyValue interface{}
	var keyName string
	_ = keyValue
	_ = keyName

	// Pull off the key/value for the entity
	for k, v := range reps {
		if k == "__typename" {
			continue
		}
		keyName = k
		keyValue = v
		break
	}

	if keyName == "" {
		return nil, fmt.Errorf("no key field given")
	}

	if f.entityResolver != nil {
		f.entityResolver(&Representation{
			TypeName: typeName,
			KeyName:  keyName,
			KeyValue: keyValue,
		})
	}*/
	//return fetchEntityByKey(typeName, keyName, keyValue)

}

// automatically build _Entity union by seaching for entity types
func (f *Federation) buildEntityType(queryFields, mutationFields graphql.Fields) {

	f.objects = make(map[string]*graphql.Object)
	f.interfaces = make(map[string]*graphql.Interface)

	// recurse through entity types to gather possible types
	for _, v := range queryFields {
		obj, ok := v.Type.(*graphql.Object)
		if ok {
			findTypes(f.objects, f.interfaces, obj, false)
		}

	}

	for _, v := range mutationFields {
		obj, ok := v.Type.(*graphql.Object)
		if ok {
			findTypes(f.objects, f.interfaces, obj, false)
		}
	}

	//fmt.Fprintln(os.Stdout, "total objects found: ", len(f.objects))

	entityTypes := make([]*graphql.Object, 0, 10)
	for _, obj := range f.objects {
		keyFields, err := getKeyDirectives(obj)
		if err != nil {
			fmt.Fprintln(os.Stdout, "err getting keys: ", err)
		}
		if len(keyFields) > 0 {
			entityTypes = append(entityTypes, obj)
		}
	}

	if len(entityTypes) == 0 {
		fmt.Fprintln(os.Stdout, "no entity types found")
	}

	f.entityType = graphql.NewUnion(graphql.UnionConfig{
		Name:  "_Entity",
		Types: entityTypes,
		ResolveType: func(p graphql.ResolveTypeParams) *graphql.Object {
			fmt.Fprintln(os.Stdout, "union resolv called: ", p.Value)
			return nil
		},
	})

}

func (f *Federation) BuildSubgraphSchema(queryFields, mutationFields graphql.Fields) *graphql.Schema {

	f.buildEntityType(queryFields, mutationFields)

	queryFields["_entities"] = &graphql.Field{
		Type: f.entityType,
		Args: graphql.FieldConfigArgument{
			"representations": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(AnyType))),
			},
		},
		Resolve: f.resolveEntity,
	}
	queryFields["_service"] = &graphql.Field{
		Type: graphql.NewNonNull(serviceType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			sdl, _ := printSDL(f.schema, f.entityType)

			return struct {
				SDL string `json:"sdl"`
			}{
				SDL: sdl,
			}, nil
		},
	}

	var queryType = graphql.NewObject(
		graphql.ObjectConfig{
			Name:   "Query",
			Fields: queryFields,
		},
	)
	var mutationType = graphql.NewObject(
		graphql.ObjectConfig{
			Name:   "Mutation",
			Fields: mutationFields,
		},
	)

	/*fmt.Fprintln(os.Stdout, "look in query type")
	for k, v := range queryType.Fields() {
		fmt.Fprintln(os.Stdout, k)
		fmt.Fprintln(os.Stdout, v.Type)
	}
	if queryType.Error() != nil {
		fmt.Fprintln(os.Stdout, queryType.Error().Error())
	}
	fmt.Fprintln(os.Stdout, "look in query fields")
	for k, v := range queryFields {
		fmt.Fprintln(os.Stdout, k)
		fmt.Fprintln(os.Stdout, v.Type)
	}*/

	schema, _ := graphql.NewSchema(
		graphql.SchemaConfig{
			Query:    queryType,
			Mutation: mutationType,
			Types: []graphql.Type{
				graphql.NewInputObject(graphql.InputObjectConfig{}),
			},
		},
	)

	f.schema = &schema

	return f.schema
}

func (f *Federation) Schema() *graphql.Schema {
	return f.schema
}

func (f *Federation) SetEntityResolver(resolverFn EntityResolverFn) {
	f.entityResolver = resolverFn
}

func (f *Federation) SetEntityBatchResolver(resolverFn EntityResolverBatchFn) {
	f.batchEntityResolver = resolverFn
}

func (f *Federation) PrintSDL() string {
	sdl, _ := printSDL(f.schema, f.entityType)
	return sdl
}

var serviceType = graphql.NewObject(graphql.ObjectConfig{
	Name: "_Service",
	Fields: graphql.Fields{
		"sdl": &graphql.Field{
			Type: graphql.String,
		},
	},
})

type Any struct {
	value []map[string]interface{}
}

func (id *Any) String() string {
	out, _ := json.Marshal(id.value)
	return string(out)
}

func NewAny(v string) *Any {
	a := &Any{}
	json.Unmarshal([]byte(v), a.value)
	return a
}

var AnyType = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "_Any",
	Description: "The `_Any` scalar is used to pass representations of entities from external services into the root _entities field for execution.",
	// Serialize serializes `_Any` to string.
	Serialize: func(value interface{}) interface{} {
		switch value := value.(type) {
		case Any:
			return value.String()
		case *Any:
			v := *value
			return v.String()
		default:
			return nil
		}
	},
	// ParseValue parses GraphQL variables from `string` to `Any`.
	ParseValue: func(value interface{}) interface{} {
		switch value := value.(type) {
		case string:
			return NewAny(value)
		case *string:
			return NewAny(*value)
		default:
			return nil
		}
	},
	// ParseLiteral parses GraphQL AST value to `CustomID`.
	ParseLiteral: func(valueAST ast.Value) interface{} {
		switch valueAST := valueAST.(type) {
		case *ast.StringValue:
			return NewAny(valueAST.Value)
		default:
			return nil
		}
	},
})

// return get all key fields for a specific object
func getKeyDirectives(obj *graphql.Object) ([]string, error) {

	keyFields := make([]string, 0, 10)

	for key, value := range obj.Extensions() {
		// if extention "name" is directive assert the type
		if key == "directives" {
			directives, ok := value.([]*DirectiveValue)
			if !ok {
				return nil, fmt.Errorf("key directive has invalid type")
			}

			for _, directive := range directives {
				// skip non-key directives
				if directive.Name != "key" {
					continue
				}
				for k, v := range directive.Values {
					if k == "fields" {
						keyFields = append(keyFields, v.(string))
					}
				}
			}

		}

	}
	return keyFields, nil
}

/*
func fetchEntityByKey(typeName, keyName string, keyValue interface{}) (interface{}, error) {
	// Verify typename is an entity
	// Verify keyname is a key for entity
	// Fetch enitity using key value

	// iterate through _Entity types to find type by typeName TODO: optimize with map
	for _, v := range entityType.Types() {
		if v.Name() == typeName {
			// look through all extensions on the object
			for key, value := range v.Extensions() {
				// if extention "name" is directive assert the type
				if key == "directives" {
					object, ok := value.(DirectiveValue)
					if !ok {
						return nil, fmt.Errorf("key directive has invalid type")
					}
					// skip non-key directives
					if object.Name != "key" {
						continue
					}
					for k, v := range object.Values {
						if k == "fields" {
							if v == keyName {
								// found the field being used in the representations as the key
								return databaseFind(typeName, keyName, keyValue)
							}
						}
					}
					// keys not matched, return an error
					return nil, fmt.Errorf("could not use key %s for entity type %s", keyName, typeName)
				}

			}
			// no directives on type
			return nil, fmt.Errorf("no @key directives on type %s", typeName)
		}
	}
	// type is not found in _Entity union
	return nil, fmt.Errorf("%s is not an _Entity type", typeName)
}
*/
