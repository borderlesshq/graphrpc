module example

replace github.com/borderlesshq/graphrpc/libs/infiotinc/gqlgenc => ../

go 1.15

require (
	github.com/gorilla/websocket v1.4.2
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.4.0
	github.com/urfave/cli/v2 v2.3.0
	github.com/vektah/gqlparser/v2 v2.2.0
	golang.org/x/tools v0.1.5
	nhooyr.io/websocket v1.8.7
)
