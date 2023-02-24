/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"io/fs"
	"os"
	"strings"
)

var propertyType string

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [category_name] [property_name]",
	Short: "Add a new property to already existing command set",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("unexpected number of arguments")
		}
		category := strings.ToLower(args[0])
		propertyName := strings.ToLower(args[1])
		if exists, err := categoryExists(category); !exists || err != nil {
			if err != nil {
				return err
			}
			return fmt.Errorf("the category %v doesn't exist", category)
		}
		if exists, err := propertyExistsInCategory(category, propertyName); exists || err != nil {
			if err != nil {
				return err
			}
			return fmt.Errorf("the property %v in category %v exists already", propertyName, category)
		}
		err := createPropertyOnCategory(category, propertyName, propertyType)
		if err != nil {
			return err
		}
		return nil
	},
}

// todo can signature be simpler? Without error that is
func categoryExists(category string) (bool, error) {
	configDir := "config"
	fileSuffix := "yaml"
	filePath := fmt.Sprintf("%s/%s.%s", configDir, category, fileSuffix)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}

	configSourceFilesDir := "pkg/infra/config"
	configSourceFileSuffix := "go"
	configSourcePath := fmt.Sprintf("%s/%s.%s", configSourceFilesDir, category, configSourceFileSuffix)
	_, err = os.Stat(configSourcePath)
	if os.IsNotExist(err) {
		return false, nil
	}

	var configStructureFound bool
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, configSourcePath, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return false, err
	}
	for _, decl := range node.Decls {
		declaration, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		if declaration.Tok != token.TYPE {
			continue
		}
		typeSpec, ok := declaration.Specs[0].(*ast.TypeSpec)
		if !ok {
			continue
		}
		if typeSpec.Name.Name == fmt.Sprintf("%sConfig", cases.Title(language.Und).String(category)) {
			configStructureFound = true
			break
		}
	}
	return configStructureFound, nil
}

func propertyExistsInCategory(category string, name string) (bool, error) {
	// FIXME duplicated
	configSourceFilesDir := "pkg/infra/config"
	configSourceFileSuffix := "go"
	configSourcePath := fmt.Sprintf("%s/%s.%s", configSourceFilesDir, category, configSourceFileSuffix)

	// FIXME duplicated localizing the struct
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, configSourcePath, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return false, err
	}
	for _, decl := range node.Decls {
		declaration, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		if declaration.Tok != token.TYPE {
			continue
		}
		typeSpec, ok := declaration.Specs[0].(*ast.TypeSpec)
		if !ok {
			continue
		}
		if typeSpec.Name.Name != fmt.Sprintf("%sConfig", cases.Title(language.Und).String(category)) {
			continue
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}
		for _, prop := range structType.Fields.List {
			for _, propName := range prop.Names {
				if strings.ToLower(propName.Name) == name {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// fixme every name is title case, even if the user's input is not. The rest of the letters are lower case
// fixme if I give type like time.Duration, there should be an import added. Let's do it for time for now
func createPropertyOnCategory(category string, name string, typeName string) error {
	// fixme duplicated path resolving
	configDir := "config"
	fileSuffix := "yaml"
	filePath := fmt.Sprintf("%s/%s.%s", configDir, category, fileSuffix)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fs.ModePerm)
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("%s: %v\n", name, resolveDefaultValueForType(typeName)))
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	// FIXME duplicated resolving source file path
	configSourceFilesDir := "pkg/infra/config"
	configSourceFileSuffix := "go"
	configSourcePath := fmt.Sprintf("%s/%s.%s", configSourceFilesDir, category, configSourceFileSuffix)

	// FIXME duplicated localizing the struct
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, configSourcePath, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return err
	}
	for _, decl := range node.Decls {
		declaration, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		if declaration.Tok != token.TYPE {
			continue
		}
		typeSpec, ok := declaration.Specs[0].(*ast.TypeSpec)
		if !ok {
			continue
		}
		if typeSpec.Name.Name != fmt.Sprintf("%sConfig", cases.Title(language.Und).String(category)) {
			continue
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}
		structType.Fields.List = append(structType.Fields.List, &ast.Field{
			Names: []*ast.Ident{
				{
					Name: cases.Title(language.Und).String(name),
				},
			},
			Type: ast.NewIdent(typeName),
			Tag:  &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("`yaml:\"%s\"`", name)},
		})
		break
	}
	var buf bytes.Buffer
	err = printer.Fprint(&buf, fset, node)
	if err != nil {
		return err
	}
	code := buf.String()
	formattedCode, err := format.Source([]byte(code))
	if err != nil {
		return err
	}
	err = os.WriteFile(configSourcePath, formattedCode, 0644)
	if err != nil {
		return err
	}
	return nil
}

func resolveDefaultValueForType(name string) any {
	if strings.Contains(name, "int") {
		return 0
	}
	if strings.Contains(name, "float") {
		return 0.0
	}
	return "string_value"
}

func init() {
	configCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&propertyType, "propertyType", "t", "string", "Provide type of the property")
}
