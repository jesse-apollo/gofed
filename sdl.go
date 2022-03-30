package gofed

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/graphql-go/graphql"
)

const federatedSDL = `#### Apollo Federation ####

scalar _Any
scalar _FieldSet

directive @external on FIELD_DEFINITION
directive @requires(fields: _FieldSet!) on FIELD_DEFINITION
directive @provides(fields: _FieldSet!) on FIELD_DEFINITION
directive @key(fields: _FieldSet!) repeatable on OBJECT | INTERFACE

# this is an optional directive discussed below
directive @extends on OBJECT | INTERFACE`

func findTypes(typeMap map[string]*graphql.Object, ifaces map[string]*graphql.Interface, base *graphql.Object, isRoot bool) {

	if base == nil {
		return
	}

	//fmt.Fprintln(os.Stdout, fmt.Sprintf("findTypes - %s", base.Name()))

	for _, i := range base.Interfaces() {
		ifaces[i.Name()] = i
	}

	if _, ok := typeMap[base.Name()]; ok {
		// type already in map
		//fmt.Fprintln(os.Stdout, "already in map ", base.Name())
		return
	}

	// skip root types
	if !isRoot {
		typeMap[base.Name()] = base
	}

	for _, field := range base.Fields() {
		//fmt.Fprintln(os.Stdout, fmt.Sprintf("%v", field.Type))
		//log.Errorf(field.Type.Name())

		switch field.Type.Name() {
		case "String":
		case "Int":
		case "Float":
		case "Boolean":
		case "ID":
		default:
			switch t := field.Type.(type) {
			case *graphql.Object:
				findTypes(typeMap, ifaces, t, false)
			case *graphql.List:
				innerType, ok := t.OfType.(*graphql.Object)
				if ok {
					findTypes(typeMap, ifaces, innerType, false)
				}
			case *graphql.NonNull:
				innerType, ok := t.OfType.(*graphql.Object)
				if ok {
					findTypes(typeMap, ifaces, innerType, false)
				}
			}
		}
	}
}

// printSDL - render the schema objec to a Federation compatible SDL
func printSDL(schema *graphql.Schema, entityType *graphql.Union) (string, error) {
	//fmt.Fprintln(os.Stdout, "printSDL")

	var output strings.Builder

	// type map to hold all types we can find
	typeMap := make(map[string]*graphql.Object)
	interfaces := make(map[string]*graphql.Interface)

	// recurse through entity types to gather possible types
	for _, t := range entityType.Types() {
		//fmt.Fprintln(os.Stdout, "111")
		//fmt.Fprintln(os.Stdout, t.Name())
		findTypes(typeMap, interfaces, t, false)
	}

	printUnion(entityType, &output)
	printDirectives(schema.Directives(), &output)

	// print all interfaces
	for _, v := range sortInterfaces(interfaces) {
		printInterface(v, &output)
	}

	// also check query root in case we missed something in the entity types
	findTypes(typeMap, interfaces, schema.QueryType(), true)
	findTypes(typeMap, interfaces, schema.MutationType(), true)

	for _, v := range sortObjects(typeMap) {
		printType(v, &output)
	}

	printQuery(schema.QueryType(), &output)
	printMutation(schema.MutationType(), &output)

	output.WriteString("\n\n")

	// write federation specific types, directives, and query extensions
	output.WriteString(federatedSDL)

	return output.String(), nil
}

func printUnion(u *graphql.Union, out *strings.Builder) {
	if desc := u.Description(); desc != "" {
		printDescription(desc, 0, out)
	}
	fmt.Fprintf(out, "union %s = ", u.Name())

	typeNames := make([]string, 0, len(u.Types()))
	for _, t := range u.Types() {
		typeNames = append(typeNames, t.Name())
	}
	out.WriteString(strings.Join(typeNames, " | "))
	out.WriteString("\n\n")
}

func printDirectives(d []*graphql.Directive, out *strings.Builder) error {

	for _, directive := range d {

		out.WriteString("directive @")
		out.WriteString(directive.Name)
		out.WriteString("(")

		for _, arg := range directive.Args {
			switch arg.Type.(type) {
			case *graphql.List:
				fmt.Fprintf(out, "%s: [%s]", arg.Name(), arg.Type.Name())
			default:
				fmt.Fprintf(out, "%s: %s", arg.Name(), arg.Type.Name())
			}

		}
		out.WriteString(") on ")
		out.WriteString(strings.Join(directive.Locations, " | "))
		out.WriteString("\n\n")

	}

	return nil
}

func printInterface(t *graphql.Interface, out *strings.Builder) error {
	if desc := t.Description(); desc != "" {
		printDescription(desc, 0, out)
	}

	out.WriteString("interface ")
	out.WriteString(t.Name())
	out.WriteString(" {\n")

	for _, v := range sortFields(t.Fields()) {
		printField(v, out)
	}

	out.WriteString("}\n\n")
	return nil
}

// sort fields in types so not so nondeterministic for testing
func sortFields(fields graphql.FieldDefinitionMap) []*graphql.FieldDefinition {
	sortedFields := make([]*graphql.FieldDefinition, 0, len(fields))
	for _, v := range fields {
		sortedFields = append(sortedFields, v)
	}
	sort.Slice(sortedFields, func(i, j int) bool {
		val := strings.Compare(sortedFields[i].Name, sortedFields[j].Name)
		return val <= 0
	})
	return sortedFields
}

func sortObjects(objs map[string]*graphql.Object) []*graphql.Object {
	sorted := make([]*graphql.Object, 0, len(objs))
	for _, v := range objs {
		sorted = append(sorted, v)
	}
	sort.Slice(sorted, func(i, j int) bool {
		val := strings.Compare(sorted[i].Name(), sorted[j].Name())
		return val <= 0

	})
	return sorted
}

func sortInterfaces(objs map[string]*graphql.Interface) []*graphql.Interface {
	sorted := make([]*graphql.Interface, 0, len(objs))
	for _, v := range objs {
		sorted = append(sorted, v)
	}
	sort.Slice(sorted, func(i, j int) bool {
		val := strings.Compare(sorted[i].Name(), sorted[j].Name())
		return val <= 0
	})
	return sorted
}

func printType(t *graphql.Object, out *strings.Builder) error {
	if desc := t.Description(); desc != "" {
		printDescription(desc, 0, out)
	}
	out.WriteString("type ")
	out.WriteString(t.Name())

	for k, v := range t.Extensions() {
		if k == "directives" {
			//fmt.Fprintln(os.Stdout, "found directives")
			obj, ok := v.([]*DirectiveValue)
			if ok {
				for _, directive := range obj {
					//fmt.Fprintln(os.Stdout, "d", directive.Name)
					out.WriteString(fmt.Sprintf(" @%s(", directive.Name))
					for k, v := range directive.Values {
						out.WriteString(k)
						out.WriteString(": \"")
						out.WriteString(fmt.Sprintf("%s", v))
						out.WriteString("\")")
					}
				}

			} else {
				fmt.Fprintln(os.Stdout, "directive assertion failed")
			}
		}
	}
	out.WriteString(" {\n")
	for _, v := range sortFields(t.Fields()) {
		printField(v, out)
	}

	out.WriteString("}\n\n")
	return nil
}

func printField(f *graphql.FieldDefinition, out *strings.Builder) {
	if desc := f.Description; desc != "" {
		printDescription(desc, 2, out)
	}

	out.WriteString("  ")
	out.WriteString(f.Name)

	if len(f.Args) > 0 {
		out.WriteString("(")
		for i, arg := range f.Args {
			out.WriteString(arg.Name())
			out.WriteString(": ")
			switch arg.Type.(type) {
			case *graphql.List:
				out.WriteString("[")
				out.WriteString(arg.Type.Name())
				out.WriteString("]")
			default:
				out.WriteString(arg.Type.Name())
			}
			if i < len(f.Args)-1 {
				out.WriteString(", ")
			}
		}

		out.WriteString(")")
	}
	out.WriteString(": ")

	switch f.Type.(type) {
	case *graphql.List:
		out.WriteString("[")
		out.WriteString(f.Type.Name())
		out.WriteString("]")
	default:
		out.WriteString(f.Type.Name())
	}

	// TODO: add args
	out.WriteString("\n")
}

func printQuery(t *graphql.Object, out *strings.Builder) {
	if t == nil {
		return
	}
	if desc := t.Description(); desc != "" {
		printDescription(desc, 0, out)
	}
	out.WriteString("type Query {\n")
	for _, v := range sortFields(t.Fields()) {
		printField(v, out)
	}
	out.WriteString("}\n\n")
}

func printMutation(t *graphql.Object, out *strings.Builder) {
	if t == nil || len(t.Fields()) == 0 {
		return
	}
	if desc := t.Description(); desc != "" {
		printDescription(desc, 0, out)
	}
	out.WriteString("type Mutation {\n")
	for _, v := range sortFields(t.Fields()) {
		printField(v, out)
	}
	out.WriteString("}\n\n")
}

func printDescription(desc string, indent int, out *strings.Builder) {
	out.WriteString(strings.Repeat(" ", indent))

	maxLineLength := 80 - indent - 4

	if !strings.Contains(desc, "\"") && len(desc) < maxLineLength {
		out.WriteString("\" ")
		out.WriteString(desc)
		out.WriteString("\"\n")
	} else {
		out.WriteString("\"\"\"\n")
		out.WriteString(strings.Repeat(" ", indent))
		out.WriteString(desc)
		out.WriteString("\n")
		out.WriteString(strings.Repeat(" ", indent))
		out.WriteString("\"\"\"\n")
	}
}
