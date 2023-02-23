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
		category := strings.ToLower(args[0])
		propertyName := strings.ToLower(args[1])
		//if !isCurrentDirProject() {
		//	return fmt.Errorf("not in a project")
		//}
		if !categoryExists(category) {
			return fmt.Errorf("the category %v doesn't exist", category)
		}
		//if propertyExistsInCategory(propertyName, category) {
		//	return fmt.Errorf("the property %v in category %v exists already", propertyName, category)
		//}
		//createPropertyOnCategory(propertyName, propertyType, category)
		return nil
	},
}

func categoryExists(category string) bool {
	configDir := "config"
	fileSuffix := "yaml"
	filePath := fmt.Sprintf("%s/%s.%s", configDir, category, fileSuffix)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}

	configSourceFilesDir := "pkg/infra/config"
	configSourceFileSuffix := "go"
	configSourcePath := fmt.Sprintf("%s/%s.%s", configSourceFilesDir, category, configSourceFileSuffix)
	_, err = os.Stat(configSourcePath)
	if os.IsNotExist(err) {
		return false
	}

	var configStructureFound bool
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, configSourcePath, nil, parser.AllErrors|parser.ParseComments)
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
	return configStructureFound
}

func init() {
	configCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&propertyType, "propertyType", "t", "string", "Provide type of the property")
}
