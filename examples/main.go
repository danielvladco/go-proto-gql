package main

import (
	"context"
	"log"
	"net/http"

	"github.com/99designs/gqlgen/handler"
	. "github.com/danielvladco/go-proto-gql/examples/account"
	"github.com/danielvladco/go-proto-gql/examples/gql"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.Playground("Playground", "/query"))
	mux.HandleFunc("/query", handler.GraphQL(gql.NewExecutableSchema(gql.Config{
		Resolvers: root{},
	})))
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Println(err)
	}
}

type root struct{}

func (r root) Mutation() gql.MutationResolver         { return mut{} }
func (r root) Query() gql.QueryResolver               { return que{} }
func (r root) Subscription() gql.SubscriptionResolver { return subscription{} }

type mut struct{}

func (m mut) ServiceSignIn(ctx context.Context, in *SignInReq) (*SignInRes, error) {
	return &SignInRes{ //dummy request to test if works
		Account: &Account{
			Email:          in.GetEmail(),
			AccountId:      "some id",
			EmailConfirmed: true,
		},
	}, nil
}

func (m mut) Test(ctx context.Context) (bool, error) { panic("implement me") }

func (m mut) ServiceGetCurrentAccount(ctx context.Context, in *SignInReq) (*GetCurrentAccountRes, error) {
	panic("implement me")
}
func (m mut) ServiceSignUpWithEmail(ctx context.Context, in *SignUpWithEmailReq) (*SignUpWithEmailRes, error) {
	panic("implement me")
}
func (m mut) ServiceResendConfirmationEmail(ctx context.Context, in *ResendConfirmationEmailReq) (*SignUpWithEmailRes, error) {
	panic("implement me")
}
func (m mut) ServiceConfirmEmail(ctx context.Context, in *ConfirmEmailReq) (*SignUpWithEmailRes, error) {
	panic("implement me")
}
func (m mut) ServiceForgotPassword(ctx context.Context, in *ForgotPasswordReq) (*SignUpWithEmailRes, error) {
	panic("implement me")
}
func (m mut) ServiceCheckResetPasswordToken(ctx context.Context, in *CheckResetPasswordTokenReq) (*SignUpWithEmailRes, error) {
	panic("implement me")
}
func (m mut) ServiceResetPassword(ctx context.Context, in *ResetPasswordReq) (*SignUpWithEmailRes, error) {
	panic("implement me")
}
func (m mut) ServiceChangePassword(ctx context.Context, in *ChangePasswordReq) (*SignUpWithEmailRes, error) {
	panic("implement me")
}
func (m mut) ServiceRequestChangeEmail(ctx context.Context, in *RequestChangeEmailReq) (*SignUpWithEmailRes, error) {
	panic("implement me")
}
func (m mut) ServiceChangeEmail(ctx context.Context, in *ChangeEmailReq) (*ChangeEmailRes, error) {
	panic("implement me")
}
func (m mut) ServiceRequestDeleteAccount(ctx context.Context, in *ChangeEmailReq) (*ChangeEmailRes, error) {
	panic("implement me")
}

type que struct{}

func (q que) Test(ctx context.Context) (bool, error) { panic("implement me") }
func (q que) ServiceDeleteAccount(ctx context.Context, in *DeleteAccountReq) (bool, error) {
	panic("implement me")
}
func (q que) AuthSvcSignIn(ctx context.Context, in *SignInReq1) (*SignInRes1, error) {
	panic("implement me")
}
func (q que) AuthSvcGetCurrentAccount(ctx context.Context, in *SignInReq1) (*GetCurrentAccountRes1, error) {
	panic("implement me")
}
func (q que) AuthSvcSignUpWithEmail(ctx context.Context, in *SignUpWithEmailReq1) (*SignUpWithEmailRes1, error) {
	panic("implement me")
}
func (q que) AuthSvcResendConfirmationEmail(ctx context.Context, in *ResendConfirmationEmailReq1) (*ResendConfirmationEmailRes1, error) {
	panic("implement me")
}

type subscription struct{}

func (s subscription) Test(ctx context.Context) (<-chan bool, error) { panic("implement me") }
