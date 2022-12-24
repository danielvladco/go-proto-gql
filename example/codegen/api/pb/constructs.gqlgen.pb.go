package pb

import (
	context "context"
	fmt "fmt"
	graphql "github.com/99designs/gqlgen/graphql"
	any1 "github.com/golang/protobuf/ptypes/any"
	empty "github.com/golang/protobuf/ptypes/empty"
	io "io"
)

type ConstructsResolvers struct{ Service ConstructsServer }

func (s *ConstructsResolvers) ConstructsScalars(ctx context.Context, in *Scalars) (*Scalars, error) {
	return s.Service.Scalars_(ctx, in)
}
func (s *ConstructsResolvers) ConstructsRepeated(ctx context.Context, in *Repeated) (*Repeated, error) {
	return s.Service.Repeated_(ctx, in)
}
func (s *ConstructsResolvers) ConstructsMaps(ctx context.Context, in *Maps) (*Maps, error) {
	return s.Service.Maps_(ctx, in)
}
func (s *ConstructsResolvers) ConstructsAny(ctx context.Context, in *any1.Any) (*Any, error) {
	return s.Service.Any_(ctx, in)
}
func (s *ConstructsResolvers) ConstructsEmpty(ctx context.Context) (*bool, error) {
	_, err := s.Service.Empty_(ctx, &empty.Empty{})
	return nil, err
}
func (s *ConstructsResolvers) ConstructsEmpty2(ctx context.Context) (*bool, error) {
	_, err := s.Service.Empty2_(ctx, &EmptyRecursive{})
	return nil, err
}
func (s *ConstructsResolvers) ConstructsEmpty3(ctx context.Context) (*bool, error) {
	_, err := s.Service.Empty3_(ctx, &Empty3{})
	return nil, err
}
func (s *ConstructsResolvers) ConstructsRef(ctx context.Context, in *Ref) (*Ref, error) {
	return s.Service.Ref_(ctx, in)
}
func (s *ConstructsResolvers) ConstructsOneof(ctx context.Context, in *Oneof) (*Oneof, error) {
	return s.Service.Oneof_(ctx, in)
}
func (s *ConstructsResolvers) ConstructsCallWithID(ctx context.Context) (*bool, error) {
	_, err := s.Service.CallWithId(ctx, &Empty{})
	return nil, err
}

type Empty3Input = Empty3
type Empty3_IntInput = Empty3_Int
type FooInput = Foo
type Foo_Foo2Input = Foo_Foo2
type BazInput = Baz
type ScalarsInput = Scalars
type RepeatedInput = Repeated
type MapsInput = Maps
type MapsResolvers struct{}
type MapsInputResolvers struct{}

func (r MapsResolvers) Int32Int32(_ context.Context, obj *Maps) (list []*Maps_Int32Int32Entry, _ error) {
	for k, v := range obj.Int32Int32 {
		list = append(list, &Maps_Int32Int32Entry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) Int32Int32(_ context.Context, obj *Maps, data []*Maps_Int32Int32Entry) error {
	for _, v := range data {
		obj.Int32Int32[v.Key] = v.Value
	}
	return nil
}

func (r MapsResolvers) Int64Int64(_ context.Context, obj *Maps) (list []*Maps_Int64Int64Entry, _ error) {
	for k, v := range obj.Int64Int64 {
		list = append(list, &Maps_Int64Int64Entry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) Int64Int64(_ context.Context, obj *Maps, data []*Maps_Int64Int64Entry) error {
	for _, v := range data {
		obj.Int64Int64[v.Key] = v.Value
	}
	return nil
}

func (r MapsResolvers) Uint32Uint32(_ context.Context, obj *Maps) (list []*Maps_Uint32Uint32Entry, _ error) {
	for k, v := range obj.Uint32Uint32 {
		list = append(list, &Maps_Uint32Uint32Entry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) Uint32Uint32(_ context.Context, obj *Maps, data []*Maps_Uint32Uint32Entry) error {
	for _, v := range data {
		obj.Uint32Uint32[v.Key] = v.Value
	}
	return nil
}

func (r MapsResolvers) Uint64Uint64(_ context.Context, obj *Maps) (list []*Maps_Uint64Uint64Entry, _ error) {
	for k, v := range obj.Uint64Uint64 {
		list = append(list, &Maps_Uint64Uint64Entry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) Uint64Uint64(_ context.Context, obj *Maps, data []*Maps_Uint64Uint64Entry) error {
	for _, v := range data {
		obj.Uint64Uint64[v.Key] = v.Value
	}
	return nil
}

func (r MapsResolvers) Sint32Sint32(_ context.Context, obj *Maps) (list []*Maps_Sint32Sint32Entry, _ error) {
	for k, v := range obj.Sint32Sint32 {
		list = append(list, &Maps_Sint32Sint32Entry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) Sint32Sint32(_ context.Context, obj *Maps, data []*Maps_Sint32Sint32Entry) error {
	for _, v := range data {
		obj.Sint32Sint32[v.Key] = v.Value
	}
	return nil
}

func (r MapsResolvers) Sint64Sint64(_ context.Context, obj *Maps) (list []*Maps_Sint64Sint64Entry, _ error) {
	for k, v := range obj.Sint64Sint64 {
		list = append(list, &Maps_Sint64Sint64Entry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) Sint64Sint64(_ context.Context, obj *Maps, data []*Maps_Sint64Sint64Entry) error {
	for _, v := range data {
		obj.Sint64Sint64[v.Key] = v.Value
	}
	return nil
}

func (r MapsResolvers) Fixed32Fixed32(_ context.Context, obj *Maps) (list []*Maps_Fixed32Fixed32Entry, _ error) {
	for k, v := range obj.Fixed32Fixed32 {
		list = append(list, &Maps_Fixed32Fixed32Entry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) Fixed32Fixed32(_ context.Context, obj *Maps, data []*Maps_Fixed32Fixed32Entry) error {
	for _, v := range data {
		obj.Fixed32Fixed32[v.Key] = v.Value
	}
	return nil
}

func (r MapsResolvers) Fixed64Fixed64(_ context.Context, obj *Maps) (list []*Maps_Fixed64Fixed64Entry, _ error) {
	for k, v := range obj.Fixed64Fixed64 {
		list = append(list, &Maps_Fixed64Fixed64Entry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) Fixed64Fixed64(_ context.Context, obj *Maps, data []*Maps_Fixed64Fixed64Entry) error {
	for _, v := range data {
		obj.Fixed64Fixed64[v.Key] = v.Value
	}
	return nil
}

func (r MapsResolvers) Sfixed32Sfixed32(_ context.Context, obj *Maps) (list []*Maps_Sfixed32Sfixed32Entry, _ error) {
	for k, v := range obj.Sfixed32Sfixed32 {
		list = append(list, &Maps_Sfixed32Sfixed32Entry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) Sfixed32Sfixed32(_ context.Context, obj *Maps, data []*Maps_Sfixed32Sfixed32Entry) error {
	for _, v := range data {
		obj.Sfixed32Sfixed32[v.Key] = v.Value
	}
	return nil
}

func (r MapsResolvers) Sfixed64Sfixed64(_ context.Context, obj *Maps) (list []*Maps_Sfixed64Sfixed64Entry, _ error) {
	for k, v := range obj.Sfixed64Sfixed64 {
		list = append(list, &Maps_Sfixed64Sfixed64Entry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) Sfixed64Sfixed64(_ context.Context, obj *Maps, data []*Maps_Sfixed64Sfixed64Entry) error {
	for _, v := range data {
		obj.Sfixed64Sfixed64[v.Key] = v.Value
	}
	return nil
}

func (r MapsResolvers) BoolBool(_ context.Context, obj *Maps) (list []*Maps_BoolBoolEntry, _ error) {
	for k, v := range obj.BoolBool {
		list = append(list, &Maps_BoolBoolEntry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) BoolBool(_ context.Context, obj *Maps, data []*Maps_BoolBoolEntry) error {
	for _, v := range data {
		obj.BoolBool[v.Key] = v.Value
	}
	return nil
}

func (r MapsResolvers) StringString(_ context.Context, obj *Maps) (list []*Maps_StringStringEntry, _ error) {
	for k, v := range obj.StringString {
		list = append(list, &Maps_StringStringEntry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) StringString(_ context.Context, obj *Maps, data []*Maps_StringStringEntry) error {
	for _, v := range data {
		obj.StringString[v.Key] = v.Value
	}
	return nil
}

func (r MapsResolvers) StringBytes(_ context.Context, obj *Maps) (list []*Maps_StringBytesEntry, _ error) {
	for k, v := range obj.StringBytes {
		list = append(list, &Maps_StringBytesEntry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) StringBytes(_ context.Context, obj *Maps, data []*Maps_StringBytesEntry) error {
	for _, v := range data {
		obj.StringBytes[v.Key] = v.Value
	}
	return nil
}

func (r MapsResolvers) StringFloat(_ context.Context, obj *Maps) (list []*Maps_StringFloatEntry, _ error) {
	for k, v := range obj.StringFloat {
		list = append(list, &Maps_StringFloatEntry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) StringFloat(_ context.Context, obj *Maps, data []*Maps_StringFloatEntry) error {
	for _, v := range data {
		obj.StringFloat[v.Key] = v.Value
	}
	return nil
}

func (r MapsResolvers) StringDouble(_ context.Context, obj *Maps) (list []*Maps_StringDoubleEntry, _ error) {
	for k, v := range obj.StringDouble {
		list = append(list, &Maps_StringDoubleEntry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) StringDouble(_ context.Context, obj *Maps, data []*Maps_StringDoubleEntry) error {
	for _, v := range data {
		obj.StringDouble[v.Key] = v.Value
	}
	return nil
}

func (r MapsResolvers) StringFoo(_ context.Context, obj *Maps) (list []*Maps_StringFooEntry, _ error) {
	for k, v := range obj.StringFoo {
		list = append(list, &Maps_StringFooEntry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) StringFoo(_ context.Context, obj *Maps, data []*Maps_StringFooEntry) error {
	for _, v := range data {
		obj.StringFoo[v.Key] = v.Value
	}
	return nil
}

func (r MapsResolvers) StringBar(_ context.Context, obj *Maps) (list []*Maps_StringBarEntry, _ error) {
	for k, v := range obj.StringBar {
		list = append(list, &Maps_StringBarEntry{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m MapsInputResolvers) StringBar(_ context.Context, obj *Maps, data []*Maps_StringBarEntry) error {
	for _, v := range data {
		obj.StringBar[v.Key] = v.Value
	}
	return nil
}

type Maps_Int32Int32EntryInput = Maps_Int32Int32Entry
type Maps_Int32Int32Entry struct {
	Key   int32
	Value int32
}

type Maps_Int64Int64EntryInput = Maps_Int64Int64Entry
type Maps_Int64Int64Entry struct {
	Key   int64
	Value int64
}

type Maps_Uint32Uint32EntryInput = Maps_Uint32Uint32Entry
type Maps_Uint32Uint32Entry struct {
	Key   uint32
	Value uint32
}

type Maps_Uint64Uint64EntryInput = Maps_Uint64Uint64Entry
type Maps_Uint64Uint64Entry struct {
	Key   uint64
	Value uint64
}

type Maps_Sint32Sint32EntryInput = Maps_Sint32Sint32Entry
type Maps_Sint32Sint32Entry struct {
	Key   int32
	Value int32
}

type Maps_Sint64Sint64EntryInput = Maps_Sint64Sint64Entry
type Maps_Sint64Sint64Entry struct {
	Key   int64
	Value int64
}

type Maps_Fixed32Fixed32EntryInput = Maps_Fixed32Fixed32Entry
type Maps_Fixed32Fixed32Entry struct {
	Key   uint32
	Value uint32
}

type Maps_Fixed64Fixed64EntryInput = Maps_Fixed64Fixed64Entry
type Maps_Fixed64Fixed64Entry struct {
	Key   uint64
	Value uint64
}

type Maps_Sfixed32Sfixed32EntryInput = Maps_Sfixed32Sfixed32Entry
type Maps_Sfixed32Sfixed32Entry struct {
	Key   int32
	Value int32
}

type Maps_Sfixed64Sfixed64EntryInput = Maps_Sfixed64Sfixed64Entry
type Maps_Sfixed64Sfixed64Entry struct {
	Key   int64
	Value int64
}

type Maps_BoolBoolEntryInput = Maps_BoolBoolEntry
type Maps_BoolBoolEntry struct {
	Key   bool
	Value bool
}

type Maps_StringStringEntryInput = Maps_StringStringEntry
type Maps_StringStringEntry struct {
	Key   string
	Value string
}

type Maps_StringBytesEntryInput = Maps_StringBytesEntry
type Maps_StringBytesEntry struct {
	Key   string
	Value []byte
}

type Maps_StringFloatEntryInput = Maps_StringFloatEntry
type Maps_StringFloatEntry struct {
	Key   string
	Value float32
}

type Maps_StringDoubleEntryInput = Maps_StringDoubleEntry
type Maps_StringDoubleEntry struct {
	Key   string
	Value float64
}

type Maps_StringFooEntryInput = Maps_StringFooEntry
type Maps_StringFooEntry struct {
	Key   string
	Value *Foo
}

type Maps_StringBarEntryInput = Maps_StringBarEntry
type Maps_StringBarEntry struct {
	Key   string
	Value Bar
}

type AnyInput = Any
type EmptyInput = Empty
type EmptyRecursiveInput = EmptyRecursive
type EmptyNestedInput = EmptyNested
type EmptyNested_EmptyNested1Input = EmptyNested_EmptyNested1
type EmptyNested_EmptyNested1_EmptyNested2Input = EmptyNested_EmptyNested1_EmptyNested2
type TimestampInput = Timestamp
type RefInput = Ref
type Ref_BarInput = Ref_Bar
type Ref_FooInput = Ref_Foo

func MarshalRef_Foo_En(x Ref_Foo_En) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "%q", x.String())
	})
}

func UnmarshalRef_Foo_En(v interface{}) (Ref_Foo_En, error) {
	code, ok := v.(string)
	if ok {
		return Ref_Foo_En(Ref_Foo_En_value[code]), nil
	}
	return 0, fmt.Errorf("cannot unmarshal Ref_Foo_En enum")
}

type Ref_Foo_BazInput = Ref_Foo_Baz
type Ref_Foo_Baz_GzInput = Ref_Foo_Baz_Gz
type Ref_Foo_BarInput = Ref_Foo_Bar

func MarshalRef_Foo_Bar_En(x Ref_Foo_Bar_En) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "%q", x.String())
	})
}

func UnmarshalRef_Foo_Bar_En(v interface{}) (Ref_Foo_Bar_En, error) {
	code, ok := v.(string)
	if ok {
		return Ref_Foo_Bar_En(Ref_Foo_Bar_En_value[code]), nil
	}
	return 0, fmt.Errorf("cannot unmarshal Ref_Foo_Bar_En enum")
}

type OneofInput = Oneof
type OneofResolvers struct{}
type OneofInputResolvers struct{}

func (o OneofInputResolvers) Param2(_ context.Context, obj *Oneof, data *string) error {
	obj.Oneof1 = &Oneof_Param2{Param2: *data}
	return nil
}

func (o OneofInputResolvers) Param3(_ context.Context, obj *Oneof, data *string) error {
	obj.Oneof1 = &Oneof_Param3{Param3: *data}
	return nil
}

func (o OneofResolvers) Oneof1(_ context.Context, obj *Oneof) (Oneof_Oneof1, error) {
	return obj.Oneof1, nil
}

type Oneof_Oneof1 interface{}

func (o OneofInputResolvers) Param4(_ context.Context, obj *Oneof, data *string) error {
	obj.Oneof2 = &Oneof_Param4{Param4: *data}
	return nil
}

func (o OneofInputResolvers) Param5(_ context.Context, obj *Oneof, data *string) error {
	obj.Oneof2 = &Oneof_Param5{Param5: *data}
	return nil
}

func (o OneofResolvers) Oneof2(_ context.Context, obj *Oneof) (Oneof_Oneof2, error) {
	return obj.Oneof2, nil
}

type Oneof_Oneof2 interface{}

func (o OneofInputResolvers) Param6(_ context.Context, obj *Oneof, data *string) error {
	obj.Oneof3 = &Oneof_Param6{Param6: *data}
	return nil
}

func (o OneofResolvers) Oneof3(_ context.Context, obj *Oneof) (Oneof_Oneof3, error) {
	return obj.Oneof3, nil
}

type Oneof_Oneof3 interface{}

func MarshalBar(x Bar) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "%q", x.String())
	})
}

func UnmarshalBar(v interface{}) (Bar, error) {
	code, ok := v.(string)
	if ok {
		return Bar(Bar_value[code]), nil
	}
	return 0, fmt.Errorf("cannot unmarshal Bar enum")
}
