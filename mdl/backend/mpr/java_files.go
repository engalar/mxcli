// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	"github.com/mendixlabs/mxcli/sdk/javaactions"
	sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
)

// javaSourceDir returns the javasource/actions directory for the given module,
// derived from b.path (the .mpr file path) without touching b.writer.
func (b *MprBackend) javaSourceDir(moduleName string) string {
	projectRoot := filepath.Dir(b.path)
	return filepath.Join(projectRoot, "javasource", strings.ToLower(moduleName), "actions")
}

func (b *MprBackend) writeJavaSourceFileViaPath(moduleName, actionName string, javaCode string, params []*javaactions.JavaActionParameter, returnType javaactions.CodeActionReturnType, extraImports []string, extraCode string) error {
	dir := b.javaSourceDir(moduleName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create javasource directory: %w", err)
	}
	source := sdkmpr.GenerateJavaSource(moduleName, actionName, javaCode, params, returnType, extraImports, extraCode)
	filePath := filepath.Join(dir, actionName+".java")
	if err := os.WriteFile(filePath, []byte(source), 0644); err != nil {
		return fmt.Errorf("write Java source file: %w", err)
	}
	return nil
}

// writeJavaSourceFileViaPathGen is the gen-typed counterpart of
// writeJavaSourceFileViaPath (Stage 3.3.2.D6). Bridges to the existing
// sdkmpr.GenerateJavaSource via gen→sdk adapters for the parameter list
// and return type. Phase E will inline a gen-native generator once
// sdk/mpr is retired by Stage 4.
func (b *MprBackend) writeJavaSourceFileViaPathGen(moduleName, actionName string, javaCode string, params []*genJA.JavaActionParameter, returnType element.Element, extraImports []string, extraCode string) error {
	sdkParams := genParamsToSDK(params)
	sdkReturnType := genReturnTypeToSDK(returnType)
	return b.writeJavaSourceFileViaPath(moduleName, actionName, javaCode, sdkParams, sdkReturnType, extraImports, extraCode)
}

// genParamsToSDK converts gen JavaActionParameters to sdk equivalents
// for the Java source generator. Only Name and ParameterType are read by
// the generator, so the conversion is intentionally minimal.
func genParamsToSDK(params []*genJA.JavaActionParameter) []*javaactions.JavaActionParameter {
	if len(params) == 0 {
		return nil
	}
	out := make([]*javaactions.JavaActionParameter, 0, len(params))
	for _, p := range params {
		if p == nil {
			continue
		}
		out = append(out, &javaactions.JavaActionParameter{
			Name:          p.Name(),
			ParameterType: genTypeToSDKParam(p.ActionParameterType()),
		})
	}
	return out
}

// genTypeToSDK returns the sdk concrete type matching a gen-typed
// element. Both CodeActions$X and JavaActions$X namespaces accepted (per
// the schema gap documented in cmd_javaactions_gen.go).
func genTypeToSDK(e element.Element) any {
	if e == nil {
		return nil
	}
	switch e.TypeName() {
	case "CodeActions$BooleanType", "JavaActions$BooleanType":
		return &javaactions.BooleanType{}
	case "CodeActions$IntegerType", "JavaActions$IntegerType":
		return &javaactions.IntegerType{}
	case "CodeActions$LongType", "JavaActions$LongType":
		return &javaactions.LongType{}
	case "CodeActions$DecimalType", "JavaActions$DecimalType":
		return &javaactions.DecimalType{}
	case "CodeActions$StringType", "JavaActions$StringType":
		return &javaactions.StringType{}
	case "CodeActions$DateTimeType", "JavaActions$DateTimeType":
		return &javaactions.DateTimeType{}
	case "CodeActions$EntityType", "JavaActions$EntityType",
		"CodeActions$ConcreteEntityType", "JavaActions$ConcreteEntityType":
		// EntityType carries Entity qualified name; both gen and sdk
		// read from the same BSON "Entity" key.
		return &javaactions.EntityType{}
	default:
		return nil
	}
}

// genReturnTypeToSDK adapts genTypeToSDK for the return-type interface.
func genReturnTypeToSDK(e element.Element) javaactions.CodeActionReturnType {
	if v, ok := genTypeToSDK(e).(javaactions.CodeActionReturnType); ok {
		return v
	}
	return nil
}

// genTypeToSDKParam adapts genTypeToSDK for the parameter-type interface.
func genTypeToSDKParam(e element.Element) javaactions.CodeActionParameterType {
	if v, ok := genTypeToSDK(e).(javaactions.CodeActionParameterType); ok {
		return v
	}
	return nil
}

func (b *MprBackend) deleteJavaSourceFileViaPath(moduleName, actionName string) error {
	filePath := filepath.Join(b.javaSourceDir(moduleName), actionName+".java")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete Java source file: %w", err)
	}
	return nil
}

func (b *MprBackend) renameJavaSourceFileViaPath(moduleName, oldName, newName string) error {
	dir := b.javaSourceDir(moduleName)
	oldPath := filepath.Join(dir, oldName+".java")
	newPath := filepath.Join(dir, newName+".java")
	if err := os.Rename(oldPath, newPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("rename Java source file: %w", err)
	}
	return nil
}

func (b *MprBackend) readJavaSourceFileViaPath(moduleName, actionName string) (string, error) {
	filePath := filepath.Join(b.javaSourceDir(moduleName), actionName+".java")
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read Java source file: %w", err)
	}
	return string(content), nil
}
