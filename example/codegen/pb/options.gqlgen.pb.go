package pb

import (
	context "context"
)

type ServiceResolvers struct{ Service ServiceServer }

func (s *ServiceResolvers) ServiceMutate1(ctx context.Context, in *Data) (*Data, error) {
	return s.Service.Mutate1(ctx, in)
}
func (s *ServiceResolvers) ServiceMutate2(ctx context.Context, in *Data) (*Data, error) {
	return s.Service.Mutate2(ctx, in)
}
func (s *ServiceResolvers) ServiceQuery1(ctx context.Context, in *Data) (*Data, error) {
	return s.Service.Query1(ctx, in)
}
func (s *ServiceResolvers) NewName(ctx context.Context, in *Data) (*Data, error) {
	return s.Service.Name(ctx, in)
}

type TestResolvers struct{ Service TestServer }

func (s *TestResolvers) Name(ctx context.Context, in *Data) (*Data, error) {
	return s.Service.Name(ctx, in)
}
func (s *TestResolvers) NewName0(ctx context.Context, in *Data) (*Data, error) {
	return s.Service.NewName(ctx, in)
}

type QueryResolvers struct{ Service QueryServer }

func (s *QueryResolvers) QueryQuery1(ctx context.Context, in *Data) (*Data, error) {
	return s.Service.Query1(ctx, in)
}
func (s *QueryResolvers) QueryQuery2(ctx context.Context, in *Data) (*Data, error) {
	return s.Service.Query2(ctx, in)
}
func (s *QueryResolvers) QueryMutate1(ctx context.Context, in *Data) (*Data, error) {
	return s.Service.Mutate1(ctx, in)
}

type DataInput = Data
type Foo2Input = Foo2
