package plugin

import (
	"strings"
	"unicode"

	"github.com/danielvladco/go-proto-gql"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/mwitkow/go-proto-validators"
)

func getValidatorType(field *descriptor.FieldDescriptorProto) *validator.FieldValidator {
	if field.Options != nil {
		v, err := proto.GetExtension(field.Options, validator.E_Field)
		if err == nil && v.(*validator.FieldValidator) != nil {
			return v.(*validator.FieldValidator)
		}
	}
	return nil
}

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
	if v := getValidatorType(field); v != nil {
		switch {
		case v.GetMsgExists():
			return true
		case v.GetStringNotEmpty():
			return true
		case v.GetRepeatedCountMin() > 0:
			return true
		}
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
