package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type generator struct {
	rootDir string
}

type packageMeta struct {
	name             string
	structs          map[string]structMeta
	namedStringTypes map[string]bool
	funcNames        map[string]bool
}

type structMeta struct {
	name   string
	fields map[string]fieldMeta
}

type fieldMeta struct {
	name     string
	typeExpr ast.Expr
}

func newGenerator(rootDir string) *generator {
	return &generator{rootDir: rootDir}
}

func (g *generator) generateAll(specs []packageSpec) error {
	for _, spec := range specs {
		if err := g.generatePackage(spec, false); err != nil {
			return err
		}
	}
	return nil
}

func (g *generator) checkAll(specs []packageSpec) error {
	for _, spec := range specs {
		if err := g.generatePackage(spec, true); err != nil {
			return err
		}
	}
	return nil
}

func (g *generator) generatePackage(spec packageSpec, checkOnly bool) error {
	meta, err := g.loadPackageMeta(spec.Dir)
	if err != nil {
		return err
	}
	if err := validatePackageSpec(spec, meta); err != nil {
		return err
	}
	content, err := renderPackage(spec, meta.name)
	if err != nil {
		return err
	}
	outputPath := filepath.Join(g.rootDir, filepath.FromSlash(spec.Output))
	if checkOnly {
		current, err := os.ReadFile(outputPath)
		if err != nil {
			return fmt.Errorf("%s: generated file missing or unreadable: %w", spec.Dir, err)
		}
		if !bytes.Equal(current, content) {
			return fmt.Errorf("%s: generated normalization is stale; run `make normgen`", spec.Dir)
		}
		return nil
	}
	if err := os.WriteFile(outputPath, content, 0o644); err != nil {
		return fmt.Errorf("%s: write generated file: %w", spec.Dir, err)
	}
	return nil
}

func (g *generator) loadPackageMeta(dir string) (packageMeta, error) {
	absDir := filepath.Join(g.rootDir, filepath.FromSlash(dir))
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return packageMeta{}, fmt.Errorf("%s: read package dir: %w", dir, err)
	}

	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "zz_normalize.go" {
			continue
		}
		path := filepath.Join(absDir, name)
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return packageMeta{}, fmt.Errorf("%s: parse %s: %w", dir, name, err)
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		return packageMeta{}, fmt.Errorf("%s: no package files found", dir)
	}

	meta := packageMeta{
		name:             files[0].Name.Name,
		structs:          make(map[string]structMeta),
		namedStringTypes: make(map[string]bool),
		funcNames:        make(map[string]bool),
	}
	for _, file := range files {
		for _, decl := range file.Decls {
			switch typed := decl.(type) {
			case *ast.GenDecl:
				if typed.Tok != token.TYPE {
					continue
				}
				for _, spec := range typed.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					switch expr := typeSpec.Type.(type) {
					case *ast.StructType:
						structInfo := structMeta{name: typeSpec.Name.Name, fields: make(map[string]fieldMeta)}
						for _, field := range expr.Fields.List {
							for _, name := range field.Names {
								structInfo.fields[name.Name] = fieldMeta{name: name.Name, typeExpr: field.Type}
							}
						}
						meta.structs[typeSpec.Name.Name] = structInfo
					case *ast.Ident:
						if expr.Name == "string" {
							meta.namedStringTypes[typeSpec.Name.Name] = true
						}
					}
				}
			case *ast.FuncDecl:
				meta.funcNames[typed.Name.Name] = true
			}
		}
	}
	return meta, nil
}

func validatePackageSpec(spec packageSpec, meta packageMeta) error {
	if strings.TrimSpace(spec.Dir) == "" {
		return fmt.Errorf("registry package dir is required")
	}
	if strings.TrimSpace(spec.Output) == "" {
		return fmt.Errorf("%s: output path is required", spec.Dir)
	}
	for _, fn := range spec.Functions {
		if strings.TrimSpace(fn.Name) == "" || strings.TrimSpace(fn.TypeName) == "" {
			return fmt.Errorf("%s: function name and type name are required", spec.Dir)
		}
		switch fn.Mode {
		case functionModeFinal, functionModeBase:
		default:
			return fmt.Errorf("%s: function %s has unsupported mode %s", spec.Dir, fn.Name, fn.Mode)
		}
		structInfo, ok := meta.structs[fn.TypeName]
		if !ok {
			return fmt.Errorf("%s: type %s not found", spec.Dir, fn.TypeName)
		}
		for _, op := range fn.Ops {
			field, ok := structInfo.fields[op.FieldName]
			if !ok {
				return fmt.Errorf("%s: type %s field %s not found", spec.Dir, fn.TypeName, op.FieldName)
			}
			switch op.Kind {
			case fieldOpTrimString:
				if !isStringExpr(field.typeExpr) {
					return fmt.Errorf("%s: type %s field %s must be string for %s", spec.Dir, fn.TypeName, op.FieldName, op.Kind)
				}
			case fieldOpTrimStringCast:
				if !isNamedStringExpr(field.typeExpr, meta) {
					return fmt.Errorf("%s: type %s field %s must be named string for %s", spec.Dir, fn.TypeName, op.FieldName, op.Kind)
				}
			case fieldOpNormalizeStringList:
				if !isStringSliceExpr(field.typeExpr) {
					return fmt.Errorf("%s: type %s field %s must be []string for %s", spec.Dir, fn.TypeName, op.FieldName, op.Kind)
				}
			case fieldOpCall:
				if strings.TrimSpace(op.Helper) == "" {
					return fmt.Errorf("%s: type %s field %s helper is required", spec.Dir, fn.TypeName, op.FieldName)
				}
				if !meta.funcNames[op.Helper] {
					return fmt.Errorf("%s: helper %s not found", spec.Dir, op.Helper)
				}
			default:
				return fmt.Errorf("%s: type %s field %s has unsupported op %s", spec.Dir, fn.TypeName, op.FieldName, op.Kind)
			}
		}
	}
	return nil
}

func isStringExpr(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "string"
}

func isNamedStringExpr(expr ast.Expr, meta packageMeta) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && meta.namedStringTypes[ident.Name]
}

func isStringSliceExpr(expr ast.Expr) bool {
	array, ok := expr.(*ast.ArrayType)
	if !ok || array.Len != nil {
		return false
	}
	return isStringExpr(array.Elt)
}

func renderPackage(spec packageSpec, packageName string) ([]byte, error) {
	imports := requiredImports(spec)
	var builder strings.Builder
	builder.WriteString("// Code generated by tools/normgen. DO NOT EDIT.\n")
	builder.WriteString("\n")
	builder.WriteString("package ")
	builder.WriteString(packageName)
	builder.WriteString("\n\n")
	if len(imports) > 0 {
		builder.WriteString("import (\n")
		for _, path := range imports {
			builder.WriteString("\t")
			builder.WriteString(strconvQuote(path))
			builder.WriteString("\n")
		}
		builder.WriteString(")\n\n")
	}
	if needsStringSliceHelper(spec) {
		builder.WriteString(renderStringSliceHelper())
		builder.WriteString("\n")
	}
	for idx, fn := range spec.Functions {
		builder.WriteString("func ")
		builder.WriteString(fn.Name)
		builder.WriteString("(message ")
		builder.WriteString(fn.TypeName)
		builder.WriteString(") ")
		builder.WriteString(fn.TypeName)
		builder.WriteString(" {\n")
		for _, op := range fn.Ops {
			builder.WriteString(renderOp(op))
		}
		builder.WriteString("\treturn message\n")
		builder.WriteString("}\n")
		if idx != len(spec.Functions)-1 {
			builder.WriteString("\n")
		}
	}
	formatted, err := format.Source([]byte(builder.String()))
	if err != nil {
		return nil, fmt.Errorf("%s: format generated file: %w", spec.Dir, err)
	}
	return formatted, nil
}

func requiredImports(spec packageSpec) []string {
	set := make(map[string]struct{})
	for _, fn := range spec.Functions {
		for _, op := range fn.Ops {
			switch op.Kind {
			case fieldOpTrimString, fieldOpTrimStringCast, fieldOpNormalizeStringList:
				set["strings"] = struct{}{}
			}
			if op.Kind == fieldOpNormalizeStringList && op.Sort {
				set["slices"] = struct{}{}
			}
		}
	}
	imports := make([]string, 0, len(set))
	for path := range set {
		imports = append(imports, path)
	}
	sort.Strings(imports)
	return imports
}

func needsStringSliceHelper(spec packageSpec) bool {
	for _, fn := range spec.Functions {
		for _, op := range fn.Ops {
			if op.Kind == fieldOpNormalizeStringList {
				return true
			}
		}
	}
	return false
}

func renderStringSliceHelper() string {
	return `func zzNormalizeStringSlice(input []string, trim bool, dropEmpty bool, unique bool, sortValues bool) []string {
	if len(input) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(input))
	output := make([]string, 0, len(input))
	for _, value := range input {
		if trim {
			value = strings.TrimSpace(value)
		}
		if dropEmpty && value == "" {
			continue
		}
		if unique {
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
		}
		output = append(output, value)
	}
	if sortValues {
		slices.Sort(output)
	}
	if len(output) == 0 {
		return nil
	}
	return output
}
`
}

func renderOp(op fieldOp) string {
	switch op.Kind {
	case fieldOpTrimString:
		return fmt.Sprintf("\tmessage.%s = strings.TrimSpace(message.%s)\n", op.FieldName, op.FieldName)
	case fieldOpTrimStringCast:
		return fmt.Sprintf("\tmessage.%s = %s(strings.TrimSpace(string(message.%s)))\n", op.FieldName, op.FieldName, op.FieldName)
	case fieldOpNormalizeStringList:
		return fmt.Sprintf("\tmessage.%s = zzNormalizeStringSlice(message.%s, %t, %t, %t, %t)\n", op.FieldName, op.FieldName, op.Trim, op.DropEmpty, op.Unique, op.Sort)
	case fieldOpCall:
		return fmt.Sprintf("\tmessage.%s = %s(message.%s)\n", op.FieldName, op.Helper, op.FieldName)
	default:
		panic(fmt.Sprintf("unsupported op kind %s", op.Kind))
	}
}

func strconvQuote(value string) string {
	return `"` + value + `"`
}
