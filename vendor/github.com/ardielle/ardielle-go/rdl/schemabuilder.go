// Copyright 2015 Yahoo Inc.
// Licensed under the terms of the Apache version 2.0 license. See LICENSE file for terms.

package rdl

import (
	"encoding/json"
	"fmt"
)

var _ = json.Marshal
var _ = fmt.Printf

var cachedSchema *Schema

type SchemaBuilder struct {
	proto *Schema
	err   error
}

func NewSchemaBuilder(name string) *SchemaBuilder {
	sb := &SchemaBuilder{}
	sb.proto = &Schema{Name: Identifier(name)}
	sb.err = nil
	return sb
}

func (sb *SchemaBuilder) Namespace(ns string) *SchemaBuilder {
	sb.proto.Namespace = NamespacedIdentifier(ns)
	return sb
}

func (sb *SchemaBuilder) Version(version int32) *SchemaBuilder {
	sb.proto.Version = &version
	return sb
}

func (sb *SchemaBuilder) Base(base string) *SchemaBuilder {
	sb.proto.Base = base
	return sb
}

func (sb *SchemaBuilder) Comment(comment string) *SchemaBuilder {
	sb.proto.Comment = comment
	return sb
}

func (sb *SchemaBuilder) AddType(t *Type) *SchemaBuilder {
	sb.proto.Types = append(sb.proto.Types, t)
	return sb
}

func (sb *SchemaBuilder) AddResource(r *Resource) *SchemaBuilder {
	sb.proto.Resources = append(sb.proto.Resources, r)
	return sb
}

func (sb *SchemaBuilder) Build() *Schema {
	var ordered []*Type
	all := make(map[string]*Type)
	resolved := make(map[string]bool)
	for _, bt := range namesBaseType {
		resolved[bt] = true
	}
	for _, t := range sb.proto.Types {
		name, _, _ := TypeInfo(t)
		all[string(name)] = t
	}
	for _, t := range sb.proto.Types {
		name, super, _ := TypeInfo(t)
		ordered = sb.resolve(ordered, resolved, all, string(name), string(super))
	}
	sb.proto.Types = ordered
	return sb.proto
}

func (sb *SchemaBuilder) isBaseType(name string) bool {
	switch name {
	case "Bool", "Int8", "Int16", "Int32", "Int64", "Float32", "Float64":
		return true
	case "String", "Bytes", "Timestamp", "Symbol", "UUID":
		return true
	case "Struct", "Array", "Map", "Enum", "Union", "Any":
		return true
	default:
		return false
	}
}

func (sb *SchemaBuilder) resolve(ordered []*Type, resolved map[string]bool, all map[string]*Type, name, super string) []*Type {
	if _, ok := resolved[name]; ok || sb.isBaseType(name) {
		return ordered
	}
	t := all[name]
	switch super {
	case "String", "Bytes", "Bool", "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "UUID", "Timestamp":
		//no dependencies
	case "Array":
		ordered = sb.resolveRef(ordered, resolved, all, string(t.ArrayTypeDef.Items))
	case "Map":
		ordered = sb.resolveRef(ordered, resolved, all, string(t.MapTypeDef.Items))
		ordered = sb.resolveRef(ordered, resolved, all, string(t.MapTypeDef.Keys))
	case "Struct":
		for _, f := range t.StructTypeDef.Fields {
			ordered = sb.resolveRef(ordered, resolved, all, string(f.Type))
		}
	default:
		ordered = sb.resolveRef(ordered, resolved, all, string(super))
	}
	resolved[name] = true
	return append(ordered, t)
}

func (sb *SchemaBuilder) resolveRef(ordered []*Type, resolved map[string]bool, all map[string]*Type, ref string) []*Type {
	if !sb.isBaseType(ref) {
		t := all[ref]
		_, super, _ := TypeInfo(t)
		ordered = sb.resolve(ordered, resolved, all, ref, string(super))
	}
	return ordered
}

func (sb *SchemaBuilder) find(ordered []*Type, name string) *Type {
	for _, t := range ordered {
		n, _, _ := TypeInfo(t)
		if name == string(n) {
			return t
		}
	}
	return nil
}

type StringTypeBuilder struct {
	st StringTypeDef
}

func NewStringTypeBuilder(name string) *StringTypeBuilder {
	tb := new(StringTypeBuilder)
	tb.st = StringTypeDef{Type: "String", Name: TypeName(name)}
	return tb
}

func (tb *StringTypeBuilder) Comment(comment string) *StringTypeBuilder {
	tb.st.Comment = comment
	return tb
}

func (tb *StringTypeBuilder) Pattern(pattern string) *StringTypeBuilder {
	tb.st.Pattern = pattern
	return tb
}

func (tb *StringTypeBuilder) MaxSize(maxsize int32) *StringTypeBuilder {
	tb.st.MaxSize = &maxsize
	return tb
}

func (tb *StringTypeBuilder) MinSize(minsize int32) *StringTypeBuilder {
	tb.st.MinSize = &minsize
	return tb
}

func (tb *StringTypeBuilder) Build() *Type {
	t := new(Type)
	if tb.st.Pattern == "" && tb.st.MaxSize == nil && tb.st.MinSize == nil && tb.st.Values == nil {
		t.Variant = TypeVariantAliasTypeDef
		t.AliasTypeDef = &AliasTypeDef{Type: tb.st.Type, Name: tb.st.Name, Comment: tb.st.Comment}
		//annotations
	} else {
		t.Variant = TypeVariantStringTypeDef
		t.StringTypeDef = &tb.st
		//annotations
		//values
	}
	return t
}

type AliasTypeBuilder struct {
	proto AliasTypeDef
}

func NewAliasTypeBuilder(supertype string, name string) *AliasTypeBuilder {
	tb := new(AliasTypeBuilder)
	tb.proto = AliasTypeDef{Type: TypeRef(supertype), Name: TypeName(name)}
	return tb
}

func (tb *AliasTypeBuilder) Comment(comment string) *AliasTypeBuilder {
	tb.proto.Comment = comment
	return tb
}

func (tb *AliasTypeBuilder) Build() *Type {
	t := new(Type)
	t.Variant = TypeVariantAliasTypeDef
	t.AliasTypeDef = &tb.proto
	return t
}

type NumberTypeBuilder struct {
	proto NumberTypeDef
}

func NewNumberTypeBuilder(supertype string, name string) *NumberTypeBuilder {
	tb := new(NumberTypeBuilder)
	tb.proto = NumberTypeDef{Type: TypeRef(supertype), Name: TypeName(name)}
	return tb
}

func (tb *NumberTypeBuilder) Comment(comment string) *NumberTypeBuilder {
	tb.proto.Comment = comment
	return tb
}

func makeNumber(x interface{}) *Number {
	n := &Number{}
	switch v := x.(type) {
	case int8:
		n.Variant = NumberVariantInt8
		n.Int8 = &v
	case int16:
		n.Variant = NumberVariantInt16
		n.Int16 = &v
	case int32:
		n.Variant = NumberVariantInt32
		n.Int32 = &v
	case int:
		n.Variant = NumberVariantInt32
		vv := int32(v)
		n.Int32 = &vv
	case int64:
		n.Variant = NumberVariantInt64
		n.Int64 = &v
	case float32:
		n.Variant = NumberVariantFloat32
		n.Float32 = &v
	case float64:
		n.Variant = NumberVariantFloat64
		n.Float64 = &v
	}
	return n
}

func (tb *NumberTypeBuilder) Min(min interface{}) *NumberTypeBuilder {
	tb.proto.Min = makeNumber(min)
	return tb
}

func (tb *NumberTypeBuilder) Max(max interface{}) *NumberTypeBuilder {
	tb.proto.Max = makeNumber(max)
	return tb
}

func (tb *NumberTypeBuilder) Build() *Type {
	t := new(Type)
	t.Variant = TypeVariantNumberTypeDef
	t.NumberTypeDef = &tb.proto
	return t
}

type StructTypeBuilder struct {
	proto StructTypeDef
}

func NewStructTypeBuilder(supertype string, name string) *StructTypeBuilder {
	tb := new(StructTypeBuilder)
	tb.proto = StructTypeDef{Type: TypeRef(supertype), Name: TypeName(name)}
	return tb
}

func (tb *StructTypeBuilder) Comment(comment string) *StructTypeBuilder {
	tb.proto.Comment = comment
	return tb
}

func (tb *StructTypeBuilder) Field(fname string, ftype string, optional bool, def interface{}, comment string) *StructTypeBuilder {
	f := &StructFieldDef{Name: Identifier(fname), Type: TypeRef(ftype), Optional: optional, Comment: comment, Default: def}
	tb.proto.Fields = append(tb.proto.Fields, f)
	return tb
}

func (tb *StructTypeBuilder) MapField(fname string, fkeys string, fitems string, optional bool, comment string) *StructTypeBuilder {
	f := &StructFieldDef{Name: Identifier(fname), Type: "Map", Keys: TypeRef(fkeys), Items: TypeRef(fitems), Optional: optional, Comment: comment}
	tb.proto.Fields = append(tb.proto.Fields, f)
	return tb
}

func (tb *StructTypeBuilder) ArrayField(fname string, fitems string, optional bool, comment string) *StructTypeBuilder {
	f := &StructFieldDef{Name: Identifier(fname), Type: "Array", Items: TypeRef(fitems), Optional: optional, Comment: comment}
	tb.proto.Fields = append(tb.proto.Fields, f)
	return tb
}

func (tb *StructTypeBuilder) Build() *Type {
	t := new(Type)
	t.Variant = TypeVariantStructTypeDef
	t.StructTypeDef = &tb.proto
	return t
}

type ArrayTypeBuilder struct {
	proto ArrayTypeDef
}

func NewArrayTypeBuilder(supertype string, name string) *ArrayTypeBuilder {
	tb := new(ArrayTypeBuilder)
	tb.proto = ArrayTypeDef{Type: TypeRef(supertype), Name: TypeName(name)}
	return tb
}

func (tb *ArrayTypeBuilder) Comment(comment string) *ArrayTypeBuilder {
	tb.proto.Comment = comment
	return tb
}

func (tb *ArrayTypeBuilder) Items(items string) *ArrayTypeBuilder {
	tb.proto.Items = TypeRef(items)
	return tb
}

func (tb *ArrayTypeBuilder) Build() *Type {
	t := new(Type)
	t.Variant = TypeVariantArrayTypeDef
	t.ArrayTypeDef = &tb.proto
	return t
}

type MapTypeBuilder struct {
	proto MapTypeDef
}

func NewMapTypeBuilder(supertype string, name string) *MapTypeBuilder {
	tb := new(MapTypeBuilder)
	tb.proto = MapTypeDef{Type: TypeRef(supertype), Name: TypeName(name)}
	return tb
}

func (tb *MapTypeBuilder) Comment(comment string) *MapTypeBuilder {
	tb.proto.Comment = comment
	return tb
}

func (tb *MapTypeBuilder) Keys(keys string) *MapTypeBuilder {
	tb.proto.Keys = TypeRef(keys)
	return tb
}

func (tb *MapTypeBuilder) Items(items string) *MapTypeBuilder {
	tb.proto.Items = TypeRef(items)
	return tb
}

func (tb *MapTypeBuilder) Build() *Type {
	t := new(Type)
	t.Variant = TypeVariantMapTypeDef
	t.MapTypeDef = &tb.proto
	return t
}

type EnumTypeBuilder struct {
	proto EnumTypeDef
}

func NewEnumTypeBuilder(supertype string, name string) *EnumTypeBuilder {
	tb := new(EnumTypeBuilder)
	tb.proto = EnumTypeDef{Type: TypeRef(supertype), Name: TypeName(name)}
	return tb
}

func (tb *EnumTypeBuilder) Comment(comment string) *EnumTypeBuilder {
	tb.proto.Comment = comment
	return tb
}

func (tb *EnumTypeBuilder) Element(sym string, comment string) *EnumTypeBuilder {
	e := &EnumElementDef{Symbol: Identifier(sym), Comment: comment}
	tb.proto.Elements = append(tb.proto.Elements, e)
	return tb
}

func (tb *EnumTypeBuilder) Build() *Type {
	t := new(Type)
	t.Variant = TypeVariantEnumTypeDef
	t.EnumTypeDef = &tb.proto
	return t
}

type UnionTypeBuilder struct {
	proto UnionTypeDef
}

func NewUnionTypeBuilder(supertype string, name string) *UnionTypeBuilder {
	tb := new(UnionTypeBuilder)
	tb.proto = UnionTypeDef{Type: TypeRef(supertype), Name: TypeName(name)}
	return tb
}

func (tb *UnionTypeBuilder) Comment(comment string) *UnionTypeBuilder {
	tb.proto.Comment = comment
	return tb
}

func (tb *UnionTypeBuilder) Variant(variant string) *UnionTypeBuilder {
	tb.proto.Variants = append(tb.proto.Variants, TypeRef(variant))
	return tb
}

func (tb *UnionTypeBuilder) Build() *Type {
	t := new(Type)
	t.Variant = TypeVariantUnionTypeDef
	t.UnionTypeDef = &tb.proto
	return t
}

type ResourceBuilder struct {
	proto Resource
}

func NewResourceBuilder(typename string, method string, path string) *ResourceBuilder {
	rb := new(ResourceBuilder)
	rb.proto = Resource{Type: TypeRef(typename), Method: method, Path: path, Expected: "OK"}
	return rb
}

func (rb *ResourceBuilder) Comment(comment string) *ResourceBuilder {
	rb.proto.Comment = comment
	return rb
}

func (rb *ResourceBuilder) Input(name string, typename string, pparam bool, qparam string, header string, optional bool, def interface{}, comment string) *ResourceBuilder {
	ri := &ResourceInput{Name: Identifier(name), Type: TypeRef(typename), Comment: comment, PathParam: pparam, QueryParam: qparam, Header: header, Default: def, Optional: optional}
	rb.proto.Inputs = append(rb.proto.Inputs, ri)
	return rb
}

func (rb *ResourceBuilder) Output(name string, typename string, header string, optional bool, comment string) *ResourceBuilder {
	ro := &ResourceOutput{Name: Identifier(name), Type: TypeRef(typename), Comment: comment, Header: header, Optional: optional}
	rb.proto.Outputs = append(rb.proto.Outputs, ro)
	return rb
}

func (rb *ResourceBuilder) Auth(action string, resource string, authn bool, domain string) *ResourceBuilder {
	ra := &ResourceAuth{Authenticate: authn, Action: action, Resource: resource, Domain: domain}
	rb.proto.Auth = ra
	return rb
}

func (rb *ResourceBuilder) Expected(sym string) *ResourceBuilder {
	rb.proto.Expected = sym
	return rb
}

func (rb *ResourceBuilder) Exception(sym string, typename string, comment string) *ResourceBuilder {
	e := &ExceptionDef{Type: typename, Comment: comment}
	if rb.proto.Exceptions == nil {
		rb.proto.Exceptions = make(map[string]*ExceptionDef)
	}
	rb.proto.Exceptions[sym] = e
	return rb
}

func (rb *ResourceBuilder) Name(sym string) *ResourceBuilder {
	rb.proto.Name = Identifier(sym)
	return rb
}

func (rb *ResourceBuilder) Build() *Resource {
	return &rb.proto
}
