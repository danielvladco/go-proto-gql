package plugin

import (
	//"github.com/golang/protobuf/protoc-gen-go/generator"
	"strings"
	"unicode"

	"github.com/danielvladco/go-proto-gql/pb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

func getEnum(file *descriptor.FileDescriptorProto, typeName string) *descriptor.EnumDescriptorProto {
	for _, enum := range file.GetEnumType() {
		if enum.GetName() == typeName {
			return enum
		}
	}

	for _, msg := range file.GetMessageType() {
		if nes := getNestedEnum(file, msg, strings.TrimPrefix(typeName, msg.GetName()+".")); nes != nil {
			return nes
		}
	}
	return nil
}

func getNestedEnum(file *descriptor.FileDescriptorProto, msg *descriptor.DescriptorProto, typeName string) *descriptor.EnumDescriptorProto {
	for _, enum := range msg.GetEnumType() {
		if enum.GetName() == typeName {
			return enum
		}
	}

	for _, nes := range msg.GetNestedType() {
		if res := getNestedEnum(file, nes, strings.TrimPrefix(typeName, nes.GetName()+".")); res != nil {
			return res
		}
	}
	return nil
}

func resolveRequired(field *descriptor.FieldDescriptorProto) bool {
	if v := GetGqlFieldOptions(field); v != nil {
		return v.GetRequired()
	}
	return false
}

func ToLowerFirst(s string) string {
	if len(s) > 0 {
		return string(unicode.ToLower(rune(s[0]))) + s[1:]
	}
	return ""
}

func GetGqlFieldOptions(field *descriptor.FieldDescriptorProto) *gql.Field {
	if field.Options != nil {
		v, err := proto.GetExtension(field.Options, gql.E_Field)
		if err == nil && v.(*gql.Field) != nil {
			return v.(*gql.Field)
		}
	}
	return nil
}

// Match parsing algorithm from Generator.CommandLineParameters
//func Params(gen *generator.Generator) map[string]string {
//	params := make(map[string]string)
//
//	for _, parameter := range strings.Split(gen.Request.GetParameter(), ",") {
//		kvp := strings.SplitN(parameter, "=", 2)
//		if len(kvp) != 2 {
//			continue
//		}
//
//		params[kvp[0]] = kvp[1]
//	}
//
//	return params
//}
