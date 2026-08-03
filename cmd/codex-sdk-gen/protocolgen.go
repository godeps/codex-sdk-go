package main

import (
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

type schemaBundle struct {
	Definitions        map[string]map[string]any
	ClientRequest      map[string]any
	ServerNotification map[string]any
	ServerRequest      map[string]any
	Supplemental       map[string]map[string]any
}

type generationOutput struct {
	Files    map[string][]byte
	Manifest manifest
}

type protocolGenerator struct {
	bundle *schemaBundle

	decls      map[string]string
	order      []string
	emitted    map[string]bool
	helperOnce bool
}

func loadSchemaBundle(schemaDir string) (*schemaBundle, error) {
	bundle := &schemaBundle{
		Definitions:  map[string]map[string]any{},
		Supplemental: map[string]map[string]any{},
	}

	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(schemaDir, entry.Name())
		var root map[string]any
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &root); err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		if defs, ok := root["definitions"].(map[string]any); ok {
			target := bundle.Supplemental
			if entry.Name() == "codex_app_server_protocol.v2.schemas.json" {
				target = bundle.Definitions
			}
			for name, raw := range defs {
				node, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				if _, exists := target[name]; !exists {
					target[name] = node
				}
			}
		}
		switch entry.Name() {
		case "ClientRequest.json":
			bundle.ClientRequest = root
		case "ServerNotification.json":
			bundle.ServerNotification = root
		case "ServerRequest.json":
			bundle.ServerRequest = root
		default:
			name := strings.TrimSuffix(entry.Name(), ".json")
			if _, exists := bundle.Definitions[name]; !exists {
				bundle.Supplemental[name] = root
			}
		}
	}
	return bundle, nil
}

func generateProtocol(schemaDir string, lock lockFile) (generationOutput, error) {
	bundle, err := loadSchemaBundle(schemaDir)
	if err != nil {
		return generationOutput{}, err
	}
	g := &protocolGenerator{
		bundle:  bundle,
		decls:   map[string]string{},
		emitted: map[string]bool{},
		order:   nil,
	}

	names := make([]string, 0, len(bundle.Definitions)+len(bundle.Supplemental)+3)
	for name := range bundle.Definitions {
		names = append(names, name)
	}
	for name := range bundle.Supplemental {
		names = append(names, name)
	}
	names = append(names, "ClientRequest", "ServerNotification", "ServerRequest")
	sort.Strings(names)
	for _, name := range names {
		if name == "" {
			continue
		}
		if err := g.ensureNamedType(name, g.lookup(name)); err != nil {
			return generationOutput{}, err
		}
	}

	manifestValue, err := buildManifestFromBundle(bundle, lock)
	if err != nil {
		return generationOutput{}, err
	}

	typesFile, err := g.emitTypesFile()
	if err != nil {
		return generationOutput{}, err
	}
	registryFile, err := g.emitRegistryFile()
	if err != nil {
		return generationOutput{}, err
	}
	helpersFile, err := emitProtocolHelpers()
	if err != nil {
		return generationOutput{}, err
	}
	return generationOutput{
		Files: map[string][]byte{
			"protocol/generated_types.go":      typesFile,
			"protocol/generated_registries.go": registryFile,
			"protocol/helpers.go":              helpersFile,
		},
		Manifest: manifestValue,
	}, nil
}

func (g *protocolGenerator) lookup(name string) map[string]any {
	switch name {
	case "ClientRequest":
		return g.bundle.ClientRequest
	case "ServerNotification":
		return g.bundle.ServerNotification
	case "ServerRequest":
		return g.bundle.ServerRequest
	}
	if node, ok := g.bundle.Definitions[name]; ok {
		return node
	}
	return g.bundle.Supplemental[name]
}

func (g *protocolGenerator) ensureNamedType(name string, node map[string]any) error {
	if node == nil || g.emitted[name] {
		return nil
	}
	g.emitted[name] = true
	g.order = append(g.order, name)
	code, err := g.emitNamedType(name, node)
	if err != nil {
		return err
	}
	g.decls[name] = code
	return nil
}

func (g *protocolGenerator) emitNamedType(name string, node map[string]any) (string, error) {
	switch {
	case isStringEnum(node):
		return emitEnumType(name, node), nil
	case hasUnion(node):
		return g.emitUnionType(name, node)
	case nodeType(node) == "object" || len(objectProperties(node)) != 0 || additionalProperties(node) != nil:
		return g.emitStructType(name, node)
	case nodeType(node) == "array":
		expr, _, err := g.goTypeExpr(name+"Item", itemSchema(node))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("type %s []%s\n", goTypeName(name), expr), nil
	case nodeType(node) == "string":
		return fmt.Sprintf("type %s string\n", goTypeName(name)), nil
	case nodeType(node) == "integer":
		return fmt.Sprintf("type %s int64\n", goTypeName(name)), nil
	case nodeType(node) == "boolean":
		return fmt.Sprintf("type %s bool\n", goTypeName(name)), nil
	default:
		return fmt.Sprintf("type %s any\n", goTypeName(name)), nil
	}
}

func (g *protocolGenerator) emitStructType(name string, node map[string]any) (string, error) {
	props := objectProperties(node)
	keys := make([]string, 0, len(props))
	for key := range props {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	required := requiredSet(node)
	hasExtras := additionalProperties(node) != nil

	type fieldInfo struct {
		GoName   string
		GoType   string
		JSONName string
		JSONTag  string
		Required bool
		Nullable bool
	}
	fields := make([]fieldInfo, 0, len(keys))
	hasRequiredNullable := false

	var b strings.Builder
	fmt.Fprintf(&b, "type %s struct {\n", goTypeName(name))
	if len(keys) == 0 && !hasExtras {
		b.WriteString("}\n")
		return b.String(), nil
	}
	for _, key := range keys {
		prop, _ := props[key].(map[string]any)
		expr, nullable, err := g.goTypeExpr(goTypeName(name)+goFieldName(key), prop)
		if err != nil {
			return "", err
		}
		isOptional := !required[key]
		fieldType := applyOptional(expr, isOptional || nullable)
		tag := jsonTag(key, isOptional)
		goName := goFieldName(key)
		fmt.Fprintf(&b, "\t%s %s `%s`\n", goName, fieldType, tag)
		fields = append(fields, fieldInfo{
			GoName:   goName,
			GoType:   fieldType,
			JSONName: key,
			JSONTag:  tag,
			Required: required[key],
			Nullable: nullable,
		})
		if required[key] && nullable {
			hasRequiredNullable = true
		}
	}
	if hasExtras {
		b.WriteString("\tExtras map[string]json.RawMessage `json:\"-\"`\n")
	}
	b.WriteString("}\n")
	if hasExtras || hasRequiredNullable {
		aliasName := goTypeName(name) + "Alias"
		fmt.Fprintf(&b, "\nfunc (v *%s) UnmarshalJSON(data []byte) error {\n", goTypeName(name))
		fmt.Fprintf(&b, "\ttype %s struct {\n", aliasName)
		for _, field := range fields {
			fmt.Fprintf(&b, "\t\t%s %s `%s`\n", field.GoName, field.GoType, field.JSONTag)
		}
		b.WriteString("\t}\n")
		fmt.Fprintf(&b, "\tvar aux %s\n", aliasName)
		b.WriteString("\tif err := json.Unmarshal(data, &aux); err != nil {\n\t\treturn err\n\t}\n")
		for _, field := range fields {
			fmt.Fprintf(&b, "\tv.%s = aux.%s\n", field.GoName, field.GoName)
		}
		if hasRequiredNullable {
			b.WriteString("\tvar required map[string]json.RawMessage\n")
			b.WriteString("\tif err := json.Unmarshal(data, &required); err != nil {\n\t\treturn err\n\t}\n")
			for _, field := range fields {
				if field.Required && field.Nullable {
					fmt.Fprintf(&b, "\tif _, ok := required[%q]; !ok {\n\t\treturn &json.UnmarshalTypeError{Value: \"missing required field\", Type: nil, Field: %q}\n\t}\n", field.JSONName, field.JSONName)
				}
			}
		}
		if hasExtras {
			b.WriteString("\tvar extras map[string]json.RawMessage\n")
			b.WriteString("\tif err := json.Unmarshal(data, &extras); err != nil {\n\t\treturn err\n\t}\n")
			for _, field := range fields {
				fmt.Fprintf(&b, "\tdelete(extras, %q)\n", field.JSONName)
			}
			b.WriteString("\tif len(extras) == 0 {\n\t\tv.Extras = nil\n\t} else {\n\t\tv.Extras = extras\n\t}\n")
		}
		b.WriteString("\treturn nil\n}\n")

		if hasExtras {
			fmt.Fprintf(&b, "\nfunc (v %s) MarshalJSON() ([]byte, error) {\n", goTypeName(name))
			fmt.Fprintf(&b, "\ttype %s struct {\n", aliasName)
			for _, field := range fields {
				fmt.Fprintf(&b, "\t\t%s %s `%s`\n", field.GoName, field.GoType, field.JSONTag)
			}
			b.WriteString("\t}\n")
			fmt.Fprintf(&b, "\taux := %s{\n", aliasName)
			for _, field := range fields {
				fmt.Fprintf(&b, "\t\t%s: v.%s,\n", field.GoName, field.GoName)
			}
			b.WriteString("\t}\n")
			b.WriteString("\tencoded, err := json.Marshal(aux)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n")
			b.WriteString("\tvar payload map[string]json.RawMessage\n")
			b.WriteString("\tif err := json.Unmarshal(encoded, &payload); err != nil {\n\t\treturn nil, err\n\t}\n")
			b.WriteString("\tfor key, value := range v.Extras {\n\t\tif _, exists := payload[key]; exists {\n\t\t\tcontinue\n\t\t}\n\t\tpayload[key] = append(json.RawMessage(nil), value...)\n\t}\n")
			b.WriteString("\treturn json.Marshal(payload)\n}\n")
		}
	}
	return b.String(), nil
}

func (g *protocolGenerator) emitUnionType(name string, node map[string]any) (string, error) {
	variants := unionVariants(node)
	discField, discMap := detectDiscriminator(variants)
	typeName := goTypeName(name)

	type variantInfo struct {
		TypeName string
		Check    string
		Disc     string
	}
	infos := make([]variantInfo, 0, len(variants))
	for i, variant := range variants {
		info, err := g.variantInfo(typeName, variant, i)
		if err != nil {
			return "", err
		}
		infos = append(infos, info)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "type %s struct {\n\tValue any `json:\"-\"`\n\tRaw json.RawMessage `json:\"-\"`\n}\n\n", typeName)
	fmt.Fprintf(&b, "func (u *%s) UnmarshalJSON(data []byte) error {\n", typeName)
	b.WriteString("\tif u == nil {\n\t\treturn nil\n\t}\n")
	b.WriteString("\tu.Raw = append(json.RawMessage(nil), data...)\n")
	if discField != "" {
		fmt.Fprintf(&b, "\tif jsonHasField(data, %q) {\n", discField)
		fmt.Fprintf(&b, "\t\tif disc, ok := unionDiscriminator(data, %q); ok {\n", discField)
		b.WriteString("\t\tswitch disc {\n")
		for _, info := range infos {
			if info.Disc == "" {
				continue
			}
			fmt.Fprintf(&b, "\t\tcase %q:\n", info.Disc)
			if info.Check != "" {
				fmt.Fprintf(&b, "\t\t\tif !(%s) {\n\t\t\t\tbreak\n\t\t\t}\n", info.Check)
			}
			fmt.Fprintf(&b, "\t\t\tvar v %s\n", info.TypeName)
			b.WriteString("\t\t\tif err := json.Unmarshal(data, &v); err != nil {\n\t\t\t\treturn err\n\t\t\t}\n")
			b.WriteString("\t\t\tu.Value = &v\n\t\t\treturn nil\n")
		}
		b.WriteString("\t\t}\n\t\t}\n")
		b.WriteString("\t\tu.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}\n")
		b.WriteString("\t\treturn nil\n\t}\n")
		_ = discMap
	}
	for _, info := range infos {
		if info.Check == "" {
			continue
		}
		fmt.Fprintf(&b, "\tif %s {\n", info.Check)
		fmt.Fprintf(&b, "\t\tvar v %s\n", info.TypeName)
		b.WriteString("\t\tif err := json.Unmarshal(data, &v); err != nil {\n\t\t\treturn err\n\t\t}\n")
		b.WriteString("\t\tu.Value = &v\n\t\treturn nil\n\t}\n")
	}
	b.WriteString("\tu.Value = &UnknownUnionValue{Raw: append(json.RawMessage(nil), data...)}\n")
	b.WriteString("\treturn nil\n}\n\n")
	fmt.Fprintf(&b, "func (u %s) MarshalJSON() ([]byte, error) {\n", typeName)
	b.WriteString("\tif u.Value == nil && len(u.Raw) != 0 {\n\t\treturn append(json.RawMessage(nil), u.Raw...), nil\n\t}\n")
	b.WriteString("\tif u.Value == nil {\n\t\treturn []byte(\"null\"), nil\n\t}\n")
	b.WriteString("\treturn json.Marshal(u.Value)\n}\n")
	return b.String(), nil
}

func (g *protocolGenerator) variantInfo(parent string, variant map[string]any, index int) (struct {
	TypeName string
	Check    string
	Disc     string
}, error) {
	var out struct {
		TypeName string
		Check    string
		Disc     string
	}
	name := variantTypeName(parent, variant, index)
	switch {
	case refName(variant) != "":
		out.TypeName = refName(variant)
	case isStringEnum(variant):
		enumName := name
		if err := g.ensureNamedType(enumName, variant); err != nil {
			return out, err
		}
		out.TypeName = goTypeName(enumName)
	case nodeType(variant) == "object" || len(objectProperties(variant)) != 0:
		if err := g.ensureNamedType(name, variant); err != nil {
			return out, err
		}
		out.TypeName = goTypeName(name)
	default:
		// Fallback variants stay raw-only for now.
		out.TypeName = "UnknownUnionValue"
	}
	out.Check = unionCheck(variant)
	if field, disc := variantDiscriminator(variant); field != "" {
		_ = field
		out.Disc = disc
	}
	return out, nil
}

func (g *protocolGenerator) goTypeExpr(hint string, node map[string]any) (string, bool, error) {
	if node == nil {
		return "any", false, nil
	}
	if ref := refName(node); ref != "" {
		return ref, false, g.ensureNamedType(ref, g.lookup(ref))
	}
	if inner, ok := nullableInner(node); ok {
		expr, _, err := g.goTypeExpr(hint, inner)
		return expr, true, err
	}
	if allOf := schemaList(node["allOf"]); len(allOf) == 1 {
		return g.goTypeExpr(hint, allOf[0])
	}
	if isStringEnum(node) {
		return "string", false, nil
	}
	switch nodeType(node) {
	case "string":
		return "string", false, nil
	case "integer":
		return "int64", false, nil
	case "boolean":
		return "bool", false, nil
	case "array":
		item := itemSchema(node)
		expr, _, err := g.goTypeExpr(hint+"Item", item)
		if err != nil {
			return "", false, err
		}
		return "[]" + expr, false, nil
	case "object":
		if add := additionalProperties(node); add != nil {
			expr, _, err := g.goTypeExpr(hint+"Value", add)
			if err != nil {
				return "", false, err
			}
			return "map[string]" + expr, false, nil
		}
		return "map[string]any", false, nil
	default:
		if hasUnion(node) {
			return "json.RawMessage", false, nil
		}
	}
	return "any", false, nil
}

func (g *protocolGenerator) emitTypesFile() ([]byte, error) {
	var b strings.Builder
	b.WriteString("package protocol\n\n")
	b.WriteString("import \"encoding/json\"\n\n")
	for _, name := range g.order {
		b.WriteString(g.decls[name])
		if !strings.HasSuffix(g.decls[name], "\n") {
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	return format.Source([]byte(b.String()))
}

func (g *protocolGenerator) emitRegistryFile() ([]byte, error) {
	client := registryEntries(g.bundle.ClientRequest)
	notify := registryEntries(g.bundle.ServerNotification)
	server := registryEntries(g.bundle.ServerRequest)

	var b strings.Builder
	b.WriteString("package protocol\n\n")
	b.WriteString("type MethodSpec struct {\n\tMethod string\n\tWrapperType string\n\tPayloadType string\n\tResponseType string\n}\n\n")
	writeRegistry := func(name string, items []registryEntry, serverRequest bool) {
		fmt.Fprintf(&b, "var %s = map[string]MethodSpec{\n", name)
		sort.Slice(items, func(i, j int) bool { return items[i].Method < items[j].Method })
		for _, item := range items {
			response := deriveResponseType(g.bundle, item, serverRequest)
			fmt.Fprintf(&b, "\t%q: {Method: %q, WrapperType: %q, PayloadType: %q, ResponseType: %q},\n",
				item.Method, item.Method, item.WrapperType, item.PayloadType, response)
		}
		b.WriteString("}\n\n")
	}
	writeRegistry("ClientRequestRegistry", client, false)
	writeRegistry("ServerNotificationRegistry", notify, false)
	writeRegistry("ServerRequestRegistry", server, true)
	return format.Source([]byte(b.String()))
}

type registryEntry struct {
	Method       string
	WrapperType  string
	PayloadType  string
	ResponseType string
}

func registryEntries(root map[string]any) []registryEntry {
	if root == nil {
		return nil
	}
	var entries []registryEntry
	for _, variant := range unionVariants(root) {
		props := objectProperties(variant)
		methodSchema, _ := props["method"].(map[string]any)
		paramsSchema, _ := props["params"].(map[string]any)
		responseSchema, _ := props["result"].(map[string]any)
		methods := extractEnum(methodSchema)
		if len(methods) != 1 {
			continue
		}
		entries = append(entries, registryEntry{
			Method:       methods[0],
			WrapperType:  goTypeName(variantTypeName("Wrapper", variant, len(entries))),
			PayloadType:  refName(paramsSchema),
			ResponseType: refName(responseSchema),
		})
	}
	return entries
}

func buildManifestFromBundle(bundle *schemaBundle, lock lockFile) (manifest, error) {
	definitions := make([]string, 0, len(bundle.Definitions))
	for name := range bundle.Definitions {
		definitions = append(definitions, name)
	}
	sort.Strings(definitions)
	supplemental := make([]string, 0, len(bundle.Supplemental))
	for name := range bundle.Supplemental {
		supplemental = append(supplemental, name)
	}
	sort.Strings(supplemental)
	requests := registryEntries(bundle.ClientRequest)
	notifications := registryEntries(bundle.ServerNotification)
	serverRequests := registryEntries(bundle.ServerRequest)
	sort.Slice(requests, func(i, j int) bool { return requests[i].Method < requests[j].Method })
	sort.Slice(notifications, func(i, j int) bool { return notifications[i].Method < notifications[j].Method })
	sort.Slice(serverRequests, func(i, j int) bool { return serverRequests[i].Method < serverRequests[j].Method })

	toManifest := func(items []registryEntry) []manifestMethod {
		out := make([]manifestMethod, 0, len(items))
		for _, item := range items {
			out = append(out, manifestMethod{Method: item.Method, Title: item.WrapperType})
		}
		return out
	}

	return manifest{
		SchemaVersion: 1,
		Source: manifestSource{
			UpstreamRepo:   lock.UpstreamRepo,
			Commit:         lock.Commit,
			PythonRoot:     lock.PythonRoot,
			RuntimePackage: lock.RuntimePackage,
			RuntimeVersion: lock.RuntimeVersion,
			SchemaFile:     lock.SchemaFile,
			SchemaSHA256:   lock.SchemaSHA256,
		},
		Counts: manifestCounts{
			AggregateDefinitions:    len(definitions),
			SupplementalDefinitions: len(supplemental),
			Requests:                len(requests),
			Notifications:           len(notifications),
			ServerRequests:          len(serverRequests),
		},
		Definitions:             definitions,
		SupplementalDefinitions: supplemental,
		Requests:                toManifest(requests),
		Notifications:           toManifest(notifications),
		ServerRequests:          toManifest(serverRequests),
		MissingDefinitions:      []string{},
		MissingMethods:          []string{},
		UnsupportedConstructs:   []string{},
		DuplicateSymbols:        []string{},
	}, nil
}

func emitProtocolHelpers() ([]byte, error) {
	src := `package protocol

import "encoding/json"

type UnknownUnionValue struct {
	Raw json.RawMessage
}

func (v *UnknownUnionValue) UnmarshalJSON(data []byte) error {
	if v == nil {
		return nil
	}
	v.Raw = append(json.RawMessage(nil), data...)
	return nil
}

func (v UnknownUnionValue) MarshalJSON() ([]byte, error) {
	if len(v.Raw) == 0 {
		return []byte("null"), nil
	}
	return append(json.RawMessage(nil), v.Raw...), nil
}

func unionDiscriminator(data []byte, field string) (string, bool) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", false
	}
	raw, ok := payload[field]
	if !ok {
		return "", false
	}
	var out string
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", false
	}
	return out, true
}

func jsonLooksObject(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		case '{':
			return true
		default:
			return false
		}
	}
	return false
}

func jsonLooksString(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		case '"':
			return true
		default:
			return false
		}
	}
	return false
}

func jsonStringIn(data []byte, values ...string) bool {
	var out string
	if err := json.Unmarshal(data, &out); err != nil {
		return false
	}
	for _, value := range values {
		if out == value {
			return true
		}
	}
	return false
}

func jsonLooksArray(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		case '[':
			return true
		default:
			return false
		}
	}
	return false
}

func jsonLooksBool(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		case 't', 'f':
			return true
		default:
			return false
		}
	}
	return false
}

func jsonLooksNumber(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return true
		default:
			return false
		}
	}
	return false
}

func jsonHasKeys(data []byte, keys ...string) bool {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return false
	}
	for _, key := range keys {
		if _, ok := payload[key]; !ok {
			return false
		}
	}
	return true
}

func jsonHasField(data []byte, field string) bool {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return false
	}
	_, ok := payload[field]
	return ok
}
`
	return format.Source([]byte(src))
}

func walkSchema(value any, visit func(title string, enum []string)) {
	switch typed := value.(type) {
	case map[string]any:
		title, _ := typed["title"].(string)
		enum := extractEnum(typed)
		if len(enum) != 0 {
			visit(title, enum)
		}
		for _, child := range typed {
			walkSchema(child, visit)
		}
	case []any:
		for _, child := range typed {
			walkSchema(child, visit)
		}
	}
}

func sortManifestMethods(methods []manifestMethod) {
	sort.Slice(methods, func(i, j int) bool {
		if methods[i].Method == methods[j].Method {
			return methods[i].Title < methods[j].Title
		}
		return methods[i].Method < methods[j].Method
	})
}

func variantTypeName(parent string, variant map[string]any, index int) string {
	if title, _ := variant["title"].(string); title != "" {
		if isEnvelopeVariant(variant) {
			return title + "Envelope"
		}
		return title
	}
	return fmt.Sprintf("%sVariant%d", parent, index+1)
}

func isEnvelopeVariant(node map[string]any) bool {
	props := objectProperties(node)
	_, hasMethod := props["method"]
	_, hasParams := props["params"]
	return hasMethod && hasParams
}

func unionCheck(node map[string]any) string {
	if field, _ := variantDiscriminator(node); field != "" {
		return requiredObjectCheck(node)
	}
	switch nodeType(node) {
	case "object":
		return requiredObjectCheck(node)
	case "string":
		enum := extractEnum(node)
		if len(enum) != 0 {
			quoted := make([]string, 0, len(enum))
			for _, item := range enum {
				quoted = append(quoted, fmt.Sprintf("%q", item))
			}
			return fmt.Sprintf("jsonStringIn(data, %s)", strings.Join(quoted, ", "))
		}
		return "jsonLooksString(data)"
	case "array":
		return "jsonLooksArray(data)"
	case "boolean":
		return "jsonLooksBool(data)"
	case "integer":
		return "jsonLooksNumber(data)"
	default:
		if isStringEnum(node) {
			return "jsonLooksString(data)"
		}
	}
	return ""
}

func requiredObjectCheck(node map[string]any) string {
	req := requiredList(node)
	if len(req) == 0 {
		return "jsonLooksObject(data)"
	}
	quoted := make([]string, 0, len(req))
	for _, key := range req {
		quoted = append(quoted, fmt.Sprintf("%q", key))
	}
	return fmt.Sprintf("jsonLooksObject(data) && jsonHasKeys(data, %s)", strings.Join(quoted, ", "))
}

func detectDiscriminator(variants []map[string]any) (string, map[string]string) {
	candidates := []string{"type", "method"}
	for _, field := range candidates {
		mapping := map[string]string{}
		ok := true
		for _, variant := range variants {
			discriminatorField, value := variantDiscriminator(variant)
			if discriminatorField != field || value == "" {
				ok = false
				break
			}
			mapping[value] = discriminatorField
		}
		if ok {
			return field, mapping
		}
	}
	return "", nil
}

func variantDiscriminator(node map[string]any) (string, string) {
	props := objectProperties(node)
	for _, key := range []string{"type", "method"} {
		prop, _ := props[key].(map[string]any)
		enum := extractEnum(prop)
		if len(enum) == 1 {
			return key, enum[0]
		}
	}
	return "", ""
}

func hasUnion(node map[string]any) bool {
	return len(schemaList(node["oneOf"])) != 0 || len(schemaList(node["anyOf"])) != 0
}

func unionVariants(node map[string]any) []map[string]any {
	variants := schemaList(node["oneOf"])
	if len(variants) == 0 {
		variants = schemaList(node["anyOf"])
	}
	return variants
}

func isStringEnum(node map[string]any) bool {
	return nodeType(node) == "string" && len(extractEnum(node)) != 0
}

func nodeType(node map[string]any) string {
	switch raw := node["type"].(type) {
	case string:
		return raw
	case []any:
		for _, item := range raw {
			if text, ok := item.(string); ok && text != "null" {
				return text
			}
		}
	}
	return ""
}

func nullableInner(node map[string]any) (map[string]any, bool) {
	switch raw := node["type"].(type) {
	case []any:
		if len(raw) == 2 {
			hasNull := false
			var nonNull string
			for _, item := range raw {
				text, _ := item.(string)
				if text == "null" {
					hasNull = true
				} else {
					nonNull = text
				}
			}
			if hasNull && nonNull != "" {
				copy := map[string]any{}
				for key, value := range node {
					if key == "type" {
						copy[key] = nonNull
						continue
					}
					copy[key] = value
				}
				return copy, true
			}
		}
	}
	for _, key := range []string{"anyOf", "oneOf"} {
		items := schemaList(node[key])
		if len(items) == 2 {
			var other map[string]any
			hasNull := false
			for _, item := range items {
				if nodeType(item) == "null" {
					hasNull = true
				} else {
					other = item
				}
			}
			if hasNull && other != nil {
				return other, true
			}
		}
	}
	return nil, false
}

func schemaList(value any) []map[string]any {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if ok {
			out = append(out, m)
		}
	}
	return out
}

func objectProperties(node map[string]any) map[string]any {
	props, _ := node["properties"].(map[string]any)
	return props
}

func additionalProperties(node map[string]any) map[string]any {
	switch raw := node["additionalProperties"].(type) {
	case map[string]any:
		return raw
	case bool:
		if raw {
			return map[string]any{}
		}
	}
	return nil
}

func itemSchema(node map[string]any) map[string]any {
	item, _ := node["items"].(map[string]any)
	return item
}

func refName(node map[string]any) string {
	raw, _ := node["$ref"].(string)
	if raw == "" {
		return ""
	}
	parts := strings.Split(raw, "/")
	return goTypeName(parts[len(parts)-1])
}

func requiredSet(node map[string]any) map[string]bool {
	out := map[string]bool{}
	for _, key := range requiredList(node) {
		out[key] = true
	}
	return out
}

func requiredList(node map[string]any) []string {
	raw, _ := node["required"].([]any)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}
	sort.Strings(out)
	return out
}

func emitEnumType(name string, node map[string]any) string {
	typeName := goTypeName(name)
	values := extractEnum(node)
	var b strings.Builder
	fmt.Fprintf(&b, "type %s string\n\nconst (\n", typeName)
	for _, value := range values {
		fmt.Fprintf(&b, "\t%s%s %s = %q\n", typeName, goTypeName(value), typeName, value)
	}
	b.WriteString(")\n")
	return b.String()
}

func deriveResponseType(bundle *schemaBundle, item registryEntry, serverRequest bool) string {
	if item.ResponseType != "" && typeExists(bundle, item.ResponseType) {
		return item.ResponseType
	}
	var candidates []string
	if item.PayloadType != "" && strings.HasSuffix(item.PayloadType, "Params") {
		candidates = append(candidates, strings.TrimSuffix(item.PayloadType, "Params")+"Response")
	}
	if strings.HasSuffix(item.WrapperType, "RequestEnvelope") {
		candidates = append(candidates, strings.TrimSuffix(item.WrapperType, "RequestEnvelope")+"Response")
	}
	if override := responseOverrides[item.Method]; override != "" {
		candidates = append([]string{override}, candidates...)
	}
	if serverRequest && item.PayloadType != "" && strings.HasSuffix(item.PayloadType, "Params") {
		candidates = append(candidates, strings.TrimSuffix(item.PayloadType, "Params")+"Response")
	}
	for _, candidate := range candidates {
		if typeExists(bundle, candidate) {
			return candidate
		}
	}
	return ""
}

func typeExists(bundle *schemaBundle, name string) bool {
	if name == "" {
		return false
	}
	if _, ok := bundle.Definitions[name]; ok {
		return true
	}
	_, ok := bundle.Supplemental[name]
	return ok
}

var responseOverrides = map[string]string{
	"account/read":                             "GetAccountResponse",
	"account/login/start":                      "LoginAccountResponse",
	"account/login/cancel":                     "CancelLoginAccountResponse",
	"account/logout":                           "LogoutAccountResponse",
	"account/rateLimits/read":                  "GetAccountRateLimitsResponse",
	"account/usage/read":                       "GetAccountTokenUsageResponse",
	"account/workspaceMessages/read":           "GetWorkspaceMessagesResponse",
	"app/installed":                            "AppsInstalledResponse",
	"app/list":                                 "AppsListResponse",
	"app/read":                                 "AppsReadResponse",
	"config/batchWrite":                        "ConfigWriteResponse",
	"config/value/write":                       "ConfigWriteResponse",
	"mcpServerStatus/list":                     "ListMcpServerStatusResponse",
	"mcpServer/resource/read":                  "McpResourceReadResponse",
	"configRequirements/read":                  "ConfigRequirementsReadResponse",
	"externalAgentConfig/import/readHistories": "ExternalAgentConfigImportHistoriesReadResponse",
}

func extractEnum(node map[string]any) []string {
	raw, _ := node["enum"].([]any)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		text, ok := item.(string)
		if !ok {
			return nil
		}
		out = append(out, text)
	}
	return out
}

func applyOptional(expr string, optional bool) string {
	if !optional {
		return expr
	}
	if strings.HasPrefix(expr, "[]") || strings.HasPrefix(expr, "map[") || expr == "any" || expr == "json.RawMessage" {
		return expr
	}
	return "*" + expr
}

func jsonTag(name string, optional bool) string {
	if optional {
		return `json:"` + name + `,omitempty"`
	}
	return `json:"` + name + `"`
}

func goTypeName(name string) string {
	if name == "" {
		return "Unnamed"
	}
	var b strings.Builder
	upperNext := true
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if upperNext {
				b.WriteRune(unicode.ToUpper(r))
			} else {
				b.WriteRune(r)
			}
			upperNext = false
			continue
		}
		upperNext = true
	}
	out := b.String()
	if out == "" {
		return "Unnamed"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "N" + out
	}
	return out
}

func goFieldName(name string) string {
	field := goTypeName(name)
	if field == "Id" {
		return "ID"
	}
	if strings.HasSuffix(field, "Id") {
		return strings.TrimSuffix(field, "Id") + "ID"
	}
	return field
}
