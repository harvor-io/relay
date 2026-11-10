package handlers

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"go.yaml.in/yaml/v3"
)

// stabilityBadges name the badge the docs show for each x-stability level,
// as on https://readme.exalynt.com/how-it-works/stability-levels. GA is the
// default expectation, so it gets no badge.
var stabilityBadges = map[string]string{
	"prototype": "Prototype",
	"alpha":     "Alpha",
	"beta":      "Beta",
}

// stabilityMark ends every stability badge's name, so the docs page's
// stability-marks.js can find exactly those badges and draw them like the
// design package's StabilityMark. It is U+2060 WORD JOINER, which is
// invisible, so the badge reads normally wherever the script does not run.
// Keep in step with STABILITY_MARK in static/docs/stability-marks.js.
const stabilityMark = "\u2060"

// httpMethods are the keys of an OpenAPI path item that hold operations.
var httpMethods = map[string]bool{
	"get": true, "put": true, "post": true, "delete": true,
	"options": true, "head": true, "patch": true, "trace": true,
}

// bundleSpec reads the OpenAPI document at root in fsys and returns it as a
// single YAML document, so a viewer does not have to resolve references
// across files itself. A reference to a whole file, such as a path item, is
// replaced by that file's contents. A reference into a file's components,
// such as "../../schemas.yml#/components/schemas/Product", becomes a local
// reference, and that file's components are merged into the document's.
// Every operation's x-stability is also shown as an x-badges badge, and
// returned in a docsStability for the page's navigation.
func bundleSpec(fsys fs.FS, root string) ([]byte, docsStability, error) {
	b := &bundler{fsys: fsys, merged: map[string]bool{}}
	doc, err := b.load(root)
	if err != nil {
		return nil, docsStability{}, err
	}
	for len(b.pending) > 0 {
		file := b.pending[0]
		b.pending = b.pending[1:]
		if err := b.mergeComponents(doc, file); err != nil {
			return nil, docsStability{}, err
		}
	}
	stability, err := addStabilityBadges(doc)
	if err != nil {
		return nil, docsStability{}, err
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, docsStability{}, fmt.Errorf("docs: encode bundled spec: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, docsStability{}, fmt.Errorf("docs: encode bundled spec: %w", err)
	}
	return buf.Bytes(), stability, nil
}

// docsStability is every operation's x-stability level, for the docs page
// to mark each operation in its navigation and work out each tag's maturity,
// since the navigation only draws a tag's operations once it is opened.
type docsStability struct {
	// Operations holds each operation's level by its method and path, such
	// as "GET /estimates/{id}".
	Operations map[string]string `json:"operations"`
	// Tags holds the levels of each tag's operations, by the tag's name.
	Tags map[string][]string `json:"tags"`
}

type bundler struct {
	fsys fs.FS

	// merged and pending track the files whose components are, or are
	// waiting to be, merged into the bundled document.
	merged  map[string]bool
	pending []string
}

// load parses file and inlines the external references within it.
func (b *bundler) load(file string) (*yaml.Node, error) {
	data, err := fs.ReadFile(b.fsys, file)
	if err != nil {
		return nil, fmt.Errorf("docs: read %s: %w", file, err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("docs: parse %s: %w", file, err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 {
		return nil, fmt.Errorf("docs: %s is not a single YAML document", file)
	}
	node := doc.Content[0]
	if err := b.resolve(node, file); err != nil {
		return nil, err
	}
	return node, nil
}

// resolve inlines or localizes every external reference under node, which
// was read from file.
func (b *bundler) resolve(node *yaml.Node, file string) error {
	if ref, ok := refOf(node); ok && !strings.HasPrefix(ref, "#") {
		target, pointer, _ := strings.Cut(ref, "#")
		target = path.Join(path.Dir(file), target)

		if pointer == "" {
			inlined, err := b.load(target)
			if err != nil {
				return err
			}
			*node = *inlined
			return nil
		}

		if !strings.HasPrefix(pointer, "/components/") {
			return fmt.Errorf("docs: %s: reference %q must point into components", file, ref)
		}
		node.Content[1].Value = "#" + pointer
		if !b.merged[target] {
			b.merged[target] = true
			b.pending = append(b.pending, target)
		}
		return nil
	}

	for _, child := range node.Content {
		if err := b.resolve(child, file); err != nil {
			return err
		}
	}
	return nil
}

// mergeComponents copies the components of file into doc's, failing on a
// component both define.
func (b *bundler) mergeComponents(doc *yaml.Node, file string) error {
	source, err := b.load(file)
	if err != nil {
		return err
	}
	from := mappingValue(source, "components")
	if from == nil {
		return fmt.Errorf("docs: %s is referenced into but has no components", file)
	}
	into := mappingValue(doc, "components")
	if into == nil {
		into = &yaml.Node{Kind: yaml.MappingNode}
		doc.Content = append(doc.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "components"}, into)
	}

	for i := 0; i < len(from.Content); i += 2 {
		kind, entries := from.Content[i], from.Content[i+1]
		section := mappingValue(into, kind.Value)
		if section == nil {
			into.Content = append(into.Content, kind, entries)
			continue
		}
		for j := 0; j < len(entries.Content); j += 2 {
			name := entries.Content[j].Value
			if mappingValue(section, name) != nil {
				return fmt.Errorf("docs: %s: components.%s.%s is already defined", file, kind.Value, name)
			}
			section.Content = append(section.Content, entries.Content[j], entries.Content[j+1])
		}
	}
	return nil
}

// addStabilityBadges gives every operation in doc that is below GA an
// x-badges badge naming its x-stability level, and returns every
// operation's level. It fails on an operation whose level is missing or
// unknown, so none goes unlabeled.
func addStabilityBadges(doc *yaml.Node) (docsStability, error) {
	stability := docsStability{Operations: map[string]string{}, Tags: map[string][]string{}}
	paths := mappingValue(doc, "paths")
	if paths == nil {
		return stability, nil
	}
	for i := 0; i < len(paths.Content); i += 2 {
		route, item := paths.Content[i].Value, paths.Content[i+1]
		for j := 0; j < len(item.Content); j += 2 {
			method, operation := item.Content[j].Value, item.Content[j+1]
			if !httpMethods[method] {
				continue
			}
			level := mappingValue(operation, "x-stability")
			if level == nil {
				return docsStability{}, fmt.Errorf("docs: %s %s has no x-stability", strings.ToUpper(method), route)
			}
			if _, ok := stabilityBadges[level.Value]; !ok && level.Value != "ga" {
				return docsStability{}, fmt.Errorf("docs: %s %s has unknown x-stability %q", strings.ToUpper(method), route, level.Value)
			}
			stability.Operations[strings.ToUpper(method)+" "+route] = level.Value
			if tags := mappingValue(operation, "tags"); tags != nil {
				for _, tag := range tags.Content {
					stability.Tags[tag.Value] = append(stability.Tags[tag.Value], level.Value)
				}
			}
			if level.Value == "ga" {
				continue
			}
			name := stabilityBadges[level.Value]
			operation.Content = append(operation.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "x-badges"},
				&yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "name"}, {Kind: yaml.ScalarNode, Value: name + stabilityMark},
					},
				}}},
			)
		}
	}
	return stability, nil
}

// refOf returns the target of node if it is a reference object.
func refOf(node *yaml.Node) (string, bool) {
	if node.Kind != yaml.MappingNode || len(node.Content) != 2 || node.Content[0].Value != "$ref" {
		return "", false
	}
	return node.Content[1].Value, true
}

// mappingValue returns the value of key in the mapping node, or nil.
func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
