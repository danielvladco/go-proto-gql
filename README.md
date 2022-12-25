[![codecov](https://codecov.io/gh/danielvladco/go-proto-gql/branch/refactor-and-e2e-tests/graph/badge.svg?token=L3N8kUGpGV)](https://codecov.io/gh/danielvladco/go-proto-gql)

This project aims to solve the problem of communication between services that use different API protocols. 
Such as GraphAL and gRPC.

Let's say your backend services use gRPC for fast and reliable communication however your frontend uses GraphQL.
Normally your only two options:

1. Either expose gRPC endpoints alongside GraphQL endpoints or build and
2. Maintain another service that acts as a gateway between your backend infrastructure and frontend application.

This project provides tools to help you build your API much quicker for both of these cases.

1. Use `protoc-gen-gql`, `protoc-gen-gogql` and `proto2graphql` to generate boilerplate code on your backend.
2. Spin up a gateway and convey the messages from one protocol to the other on the fly!

Check out the docs for getting started and usage examples at: https://danielvladco.github.io/go-proto-gql/

## Community:
Will be happy for any contributions. So feel free to create issues, forks and PRs.

## License:

`go-proto-gql` is released under the Apache 2.0 license. See the LICENSE file for details.
