/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"go/ast"
	"go/parser"
	"go/token"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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
		createPropertyOnCategory(category, propertyName, propertyType)
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

func init() {
	configCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&propertyType, "propertyType", "t", "string", "Provide type of the property")
}
