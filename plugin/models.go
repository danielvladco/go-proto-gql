package plugin

import (
	"sort"

	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

type ModelDescriptor struct {
	BuiltIn     bool
	PackageDir  string
	TypeName    string
	UsedAsInput bool
	Index       int
	OneofTypes  map[string]struct{}

	packageName     string
	originalPackage string
}

type Method struct {
	Name         string
	InputType    string
	OutputType   string
	Phony        bool
	ServiceIndex int
	Index        int

	*descriptor.MethodDescriptorProto
	*descriptor.ServiceDescriptorProto
}

type Methods []*Method

func (ms Methods) NameExists(name string) bool {
	for _, m := range ms {
		if m.Name == name {
			return true
		}
	}
	return false
}

type Type struct {
	*descriptor.DescriptorProto
	*descriptor.EnumDescriptorProto
	ModelDescriptor
}

type TypeMapList []*TypeMapEntry

func (t TypeMapList) Len() int           { return len(t) }
func (t TypeMapList) Less(i, j int) bool { return t[i].Key < t[j].Key }
func (t TypeMapList) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }

func TypeMap2List(t map[string]*Type) (m TypeMapList) {
	for key, val := range t {
		m = append(m, &TypeMapEntry{Key: key, Value: val})
	}
	sort.Sort(m)
	return
}

type TypeMapEntry struct {
	Key   string
	Value *Type
}

type OneofRef struct {
	Parent *Type
	Index  int32
}
