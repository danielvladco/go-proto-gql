package plugin

import (
	"sort"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type stringSet map[string]struct{}

func (s stringSet) Strings() (ss []string) {
	slen, i := len(s), 0
	if slen > 0 {
		ss = make([]string, slen)
	}
	for v := range s {
		ss[i] = v
		i++
	}
	return
}

type ModelDescriptor struct {
	FQN     string
	BuiltIn bool
	//PackageDir  string
	TypeName    string
	UsedAsInput bool
	Index       int
	OneofTypes  stringSet
	Phony       bool

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
