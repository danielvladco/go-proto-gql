package main

import (
	gogoproto "github.com/gogo/protobuf/proto"
	gogodescriptor "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)
func fields(ff []*descriptor.FieldDescriptorProto) (fields []*gogodescriptor.FieldDescriptorProto) {
	for _, v := range ff {
		var typ = gogodescriptor.FieldDescriptorProto_Type(*v.Type)
		var label = gogodescriptor.FieldDescriptorProto_Label(*v.Label)
		var ctype = gogodescriptor.FieldOptions_CType(*v.Options.Ctype)
		var jstype = gogodescriptor.FieldOptions_JSType(*v.Options.Jstype)
		fields = append(fields, &gogodescriptor.FieldDescriptorProto{
			Name:         v.Name,
			Number:       v.Number,
			Label:        &label,
			Type:         &typ,
			TypeName:     v.TypeName,
			Extendee:     v.Extendee,
			DefaultValue: v.DefaultValue,
			OneofIndex:   v.OneofIndex,
			JsonName:     v.JsonName,
			Options:              &gogodescriptor.FieldOptions{
				Ctype:                  &ctype,
				Packed:                 v.Options.Packed,
				Jstype:                 &jstype,
				Lazy:                   v.Options.Lazy,
				Deprecated:             v.Options.Deprecated,
				Weak:                   v.Options.Weak,
				UninterpretedOption:    uninterpretedOption(v.Options.UninterpretedOption),
				XXX_InternalExtensions: gogoproto.XXX_InternalExtensions{},
				XXX_NoUnkeyedLiteral: v.XXX_NoUnkeyedLiteral,
				XXX_unrecognized:     v.XXX_unrecognized,
				XXX_sizecache:        v.XXX_sizecache,
			},
			XXX_NoUnkeyedLiteral: struct{}{},
			XXX_unrecognized:     nil,
			XXX_sizecache:        0,
		})
	}

	return fields
}

func reservedRange(rr []*descriptor.DescriptorProto_ReservedRange) (rezervedRange []*gogodescriptor.DescriptorProto_ReservedRange) {
	for _, v := range rr {
		rezervedRange = append(rezervedRange, &gogodescriptor.DescriptorProto_ReservedRange{
			Start:                v.Start,
			End:                  v.End,
			XXX_NoUnkeyedLiteral: v.XXX_NoUnkeyedLiteral,
			XXX_unrecognized:     v.XXX_unrecognized,
			XXX_sizecache:        v.XXX_sizecache,
		})
	}
	return rezervedRange
}
func reservedRangeEnum(rr []*descriptor.EnumDescriptorProto_EnumReservedRange) (rezervedRange []*gogodescriptor.EnumDescriptorProto_EnumReservedRange) {
	for _, v := range rr {
		rezervedRange = append(rezervedRange, &gogodescriptor.EnumDescriptorProto_EnumReservedRange{
			Start:                v.Start,
			End:                  v.End,
			XXX_NoUnkeyedLiteral: v.XXX_NoUnkeyedLiteral,
			XXX_unrecognized:     v.XXX_unrecognized,
			XXX_sizecache:        v.XXX_sizecache,
		})
	}
	return rezervedRange
}

func enums(ee []*descriptor.EnumDescriptorProto) (enums []*gogodescriptor.EnumDescriptorProto) {
	for _, v := range ee {
		var values []*gogodescriptor.EnumValueDescriptorProto
		for _, v := range v.Value {
			values = append(values, &gogodescriptor.EnumValueDescriptorProto{
				Name:   v.Name,
				Number: v.Number,
				Options: &gogodescriptor.EnumValueOptions{
					Deprecated:             v.Options.Deprecated,
					UninterpretedOption:    uninterpretedOption(v.Options.UninterpretedOption),
					XXX_InternalExtensions: gogoproto.XXX_InternalExtensions{},
					XXX_NoUnkeyedLiteral:   v.XXX_NoUnkeyedLiteral,
					XXX_unrecognized:       v.XXX_unrecognized,
					XXX_sizecache:          v.XXX_sizecache,
				},
				XXX_NoUnkeyedLiteral: v.XXX_NoUnkeyedLiteral,
				XXX_unrecognized:     v.XXX_unrecognized,
				XXX_sizecache:        v.XXX_sizecache,
			})
		}
		enums = append(enums, &gogodescriptor.EnumDescriptorProto{
			Name:  v.Name,
			Value: values,
			Options: &gogodescriptor.EnumOptions{
				AllowAlias:             v.Options.AllowAlias,
				Deprecated:             v.Options.Deprecated,
				UninterpretedOption:    uninterpretedOption(v.Options.UninterpretedOption),
				XXX_InternalExtensions: gogoproto.XXX_InternalExtensions{},
				XXX_NoUnkeyedLiteral:   v.XXX_NoUnkeyedLiteral,
				XXX_unrecognized:       v.XXX_unrecognized,
				XXX_sizecache:          v.XXX_sizecache,
			},
			ReservedRange:        reservedRangeEnum(v.ReservedRange),
			ReservedName:         nil,
			XXX_NoUnkeyedLiteral: struct{}{},
			XXX_unrecognized:     nil,
			XXX_sizecache:        0,
		})
	}
	return enums
}

func descriptors(vv []*descriptor.DescriptorProto) (messages []*gogodescriptor.DescriptorProto) {
	var messageType []*gogodescriptor.DescriptorProto
	for _, v := range vv {
		var extensionRange []*gogodescriptor.DescriptorProto_ExtensionRange
		for _, v := range v.ExtensionRange {
			extensionRange = append(extensionRange, &gogodescriptor.DescriptorProto_ExtensionRange{
				Start: v.Start,
				End:   v.End,
				Options: &gogodescriptor.ExtensionRangeOptions{
					UninterpretedOption:    uninterpretedOption(v.Options.UninterpretedOption),
					XXX_InternalExtensions: gogoproto.XXX_InternalExtensions{},
					XXX_NoUnkeyedLiteral:   v.XXX_NoUnkeyedLiteral,
					XXX_unrecognized:       v.XXX_unrecognized,
					XXX_sizecache:          v.XXX_sizecache,
				},
				XXX_NoUnkeyedLiteral: v.XXX_NoUnkeyedLiteral,
				XXX_unrecognized:     v.XXX_unrecognized,
				XXX_sizecache:        v.XXX_sizecache,
			})
		}

		var oneofs []*gogodescriptor.OneofDescriptorProto

		messageType = append(messageType, &gogodescriptor.DescriptorProto{
			Name:           v.Name,
			Field:          fields(v.Field),
			Extension:      fields(v.Extension),
			NestedType:     descriptors(v.NestedType),
			EnumType:       enums(v.EnumType),
			ExtensionRange: extensionRange,
			OneofDecl:      oneofs,
			Options: &gogodescriptor.MessageOptions{
				MessageSetWireFormat:         v.Options.MessageSetWireFormat,
				NoStandardDescriptorAccessor: v.Options.NoStandardDescriptorAccessor,
				Deprecated:                   v.Options.Deprecated,
				MapEntry:                     v.Options.MapEntry,
				UninterpretedOption:          uninterpretedOption(v.Options.UninterpretedOption),
				XXX_InternalExtensions:       gogoproto.XXX_InternalExtensions{},
				XXX_NoUnkeyedLiteral:         v.XXX_NoUnkeyedLiteral,
				XXX_unrecognized:             v.XXX_unrecognized,
				XXX_sizecache:                v.XXX_sizecache,
			},
			ReservedRange:        reservedRange(v.ReservedRange),
			ReservedName:         v.ReservedName,
			XXX_NoUnkeyedLiteral: v.XXX_NoUnkeyedLiteral,
			XXX_unrecognized:     v.XXX_unrecognized,
			XXX_sizecache:        v.XXX_sizecache,
		})
	}

	return messages
}

func services(ss []*descriptor.ServiceDescriptorProto) (services []*gogodescriptor.ServiceDescriptorProto) {
	for _, v := range ss {
		var methods []*gogodescriptor.MethodDescriptorProto
		for _, v := range v.Method {
			lvl := gogodescriptor.MethodOptions_IdempotencyLevel(*v.Options.IdempotencyLevel)
			methods = append(methods, &gogodescriptor.MethodDescriptorProto{
				Name:            nil,
				InputType:       v.InputType,
				OutputType:      v.OutputType,
				ClientStreaming: v.ClientStreaming,
				ServerStreaming: v.ServerStreaming,
				Options: &gogodescriptor.MethodOptions{
					Deprecated:             v.Options.Deprecated,
					IdempotencyLevel:       &lvl,
					UninterpretedOption:    uninterpretedOption(v.Options.UninterpretedOption),
					XXX_NoUnkeyedLiteral:   struct{}{},
					XXX_InternalExtensions: gogoproto.XXX_InternalExtensions{},
					XXX_unrecognized:       nil,
					XXX_sizecache:          0,
				},
				XXX_NoUnkeyedLiteral: v.XXX_NoUnkeyedLiteral,
				XXX_unrecognized:     v.XXX_unrecognized,
				XXX_sizecache:        v.XXX_sizecache,
			})
		}
		services = append(services, &gogodescriptor.ServiceDescriptorProto{
			Name:   v.Name,
			Method: methods,
			Options: &gogodescriptor.ServiceOptions{
				Deprecated:             v.Options.Deprecated,
				UninterpretedOption:    uninterpretedOption(v.Options.UninterpretedOption),
				XXX_InternalExtensions: gogoproto.XXX_InternalExtensions{},
				XXX_NoUnkeyedLiteral:   v.XXX_NoUnkeyedLiteral,
				XXX_unrecognized:       v.XXX_unrecognized,
				XXX_sizecache:          v.XXX_sizecache,
			},
			XXX_NoUnkeyedLiteral: v.XXX_NoUnkeyedLiteral,
			XXX_unrecognized:     v.XXX_unrecognized,
			XXX_sizecache:        v.XXX_sizecache,
		})

	}
	return services
}

func uninterpretedOption(oo []*descriptor.UninterpretedOption) (uninterpreted []*gogodescriptor.UninterpretedOption) {
	for _, v := range oo {
		var namePart []*gogodescriptor.UninterpretedOption_NamePart
		for _, v := range v.Name {
			newV := gogodescriptor.UninterpretedOption_NamePart(*v)
			namePart = append(namePart, &newV)
		}

		uninterpreted = append(uninterpreted, &gogodescriptor.UninterpretedOption{
			Name:                 namePart,
			IdentifierValue:      v.IdentifierValue,
			PositiveIntValue:     v.PositiveIntValue,
			NegativeIntValue:     v.NegativeIntValue,
			DoubleValue:          v.DoubleValue,
			StringValue:          v.StringValue,
			AggregateValue:       v.AggregateValue,
			XXX_NoUnkeyedLiteral: v.XXX_NoUnkeyedLiteral,
			XXX_unrecognized:     v.XXX_unrecognized,
			XXX_sizecache:        v.XXX_sizecache,
		})
	}
	return uninterpreted
}

func convertFile(f *descriptor.FileDescriptorProto) (newFile *gogodescriptor.FileDescriptorProto){
	o := f.Options

	var sourceCodeLoc []*gogodescriptor.SourceCodeInfo_Location
	for _, v := range f.SourceCodeInfo.Location {
		sourceCodeLoc = append(sourceCodeLoc, &gogodescriptor.SourceCodeInfo_Location{
			Path:                    v.Path,
			Span:                    v.Span,
			LeadingComments:         v.LeadingComments,
			TrailingComments:        v.TrailingComments,
			LeadingDetachedComments: v.LeadingDetachedComments,
			XXX_NoUnkeyedLiteral:    v.XXX_NoUnkeyedLiteral,
			XXX_unrecognized:        v.XXX_unrecognized,
			XXX_sizecache:           v.XXX_sizecache,
		})
	}

	optfor := gogodescriptor.FileOptions_OptimizeMode(*o.OptimizeFor)
	newFile = &gogodescriptor.FileDescriptorProto{
		Name:             f.Name,
		Package:          f.Package,
		Dependency:       f.Dependency,
		PublicDependency: f.PublicDependency,
		WeakDependency:   f.WeakDependency,
		MessageType:      descriptors(f.MessageType),
		EnumType:         enums(f.EnumType),
		Service:          services(f.Service),
		Extension:        fields(f.Extension),
		Options: &gogodescriptor.FileOptions{
			JavaPackage:               o.JavaPackage,
			JavaOuterClassname:        o.JavaOuterClassname,
			JavaMultipleFiles:         o.JavaMultipleFiles,
			JavaGenerateEqualsAndHash: o.JavaGenerateEqualsAndHash,
			JavaStringCheckUtf8:       o.JavaStringCheckUtf8,
			OptimizeFor:               &optfor,
			GoPackage:                 o.GoPackage,
			CcGenericServices:         o.CcGenericServices,
			JavaGenericServices:       o.JavaGenericServices,
			PyGenericServices:         o.PyGenericServices,
			PhpGenericServices:        o.PhpGenericServices,
			Deprecated:                o.Deprecated,
			CcEnableArenas:            o.CcEnableArenas,
			ObjcClassPrefix:           o.ObjcClassPrefix,
			CsharpNamespace:           o.CsharpNamespace,
			SwiftPrefix:               o.SwiftPrefix,
			PhpClassPrefix:            o.PhpClassPrefix,
			PhpNamespace:              o.PhpNamespace,
			PhpMetadataNamespace:      o.PhpMetadataNamespace,
			RubyPackage:               o.RubyPackage,
			UninterpretedOption:       uninterpretedOption(o.UninterpretedOption),
			XXX_InternalExtensions:    gogoproto.XXX_InternalExtensions{},
			XXX_NoUnkeyedLiteral:      o.XXX_NoUnkeyedLiteral,
			XXX_unrecognized:          o.XXX_unrecognized,
			XXX_sizecache:             o.XXX_sizecache,
		},
		SourceCodeInfo: &gogodescriptor.SourceCodeInfo{
			Location:             sourceCodeLoc,
			XXX_NoUnkeyedLiteral: f.SourceCodeInfo.XXX_NoUnkeyedLiteral,
			XXX_unrecognized:     f.SourceCodeInfo.XXX_unrecognized,
			XXX_sizecache:        f.SourceCodeInfo.XXX_sizecache,
		},
		Syntax:               f.Syntax,
		XXX_NoUnkeyedLiteral: f.XXX_NoUnkeyedLiteral,
		XXX_unrecognized:     f.XXX_unrecognized,
		XXX_sizecache:        f.XXX_sizecache,
	}

	return newFile
}
