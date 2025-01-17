package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/borderlesshq/graphrpc/libs/99designs/gqlgen/graphql/errcode"
	"github.com/borderlesshq/graphrpc/utils"
	"github.com/pkg/errors"
	"mime"
	"net/http"
	"time"

	"github.com/borderlesshq/graphrpc/libs/99designs/gqlgen/graphql"
	"github.com/borderlesshq/graphrpc/libs/99designs/gqlgen/graphql/executor"
	"github.com/borderlesshq/graphrpc/libs/99designs/gqlgen/graphql/handler/extension"
	"github.com/borderlesshq/graphrpc/libs/99designs/gqlgen/graphql/handler/lru"
	"github.com/borderlesshq/graphrpc/libs/99designs/gqlgen/graphql/handler/transport"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type (
	Server struct {
		transports []graphql.Transport
		exec       *executor.Executor
	}
)

func New(es graphql.ExecutableSchema) *Server {
	return &Server{
		exec: executor.New(es),
	}
}

func NewDefaultServer(es graphql.ExecutableSchema) *Server {
	srv := New(es)

	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	srv.SetQueryCache(lru.New(1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New(100),
	})

	return srv
}

func (s *Server) AddTransport(transport graphql.Transport) {
	s.transports = append(s.transports, transport)
}

func (s *Server) SetErrorPresenter(f graphql.ErrorPresenterFunc) {
	s.exec.SetErrorPresenter(f)
}

func (s *Server) SetRecoverFunc(f graphql.RecoverFunc) {
	s.exec.SetRecoverFunc(f)
}

func (s *Server) SetQueryCache(cache graphql.Cache) {
	s.exec.SetQueryCache(cache)
}

func (s *Server) Use(extension graphql.HandlerExtension) {
	s.exec.Use(extension)
}

// AroundFields is a convenience method for creating an extension that only implements field middleware
func (s *Server) AroundFields(f graphql.FieldMiddleware) {
	s.exec.AroundFields(f)
}

// AroundRootFields is a convenience method for creating an extension that only implements field middleware
func (s *Server) AroundRootFields(f graphql.RootFieldMiddleware) {
	s.exec.AroundRootFields(f)
}

// AroundOperations is a convenience method for creating an extension that only implements operation middleware
func (s *Server) AroundOperations(f graphql.OperationMiddleware) {
	s.exec.AroundOperations(f)
}

// AroundResponses is a convenience method for creating an extension that only implements response middleware
func (s *Server) AroundResponses(f graphql.ResponseMiddleware) {
	s.exec.AroundResponses(f)
}

func (s *Server) getTransport(r *http.Request) graphql.Transport {
	for _, t := range s.transports {
		if t.Supports(r) {
			return t
		}
	}
	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	applyMsgpackEncoder := false

	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))

	if mediaType == "application/msgpack" {
		applyMsgpackEncoder = true
	}

	defer func() {
		if err := recover(); err != nil {
			err := s.exec.PresentRecoveredError(r.Context(), err)
			resp := &graphql.Response{Errors: []*gqlerror.Error{err}}

			var b []byte
			if applyMsgpackEncoder {
				b, _ = utils.Marshal(resp)
			} else {
				b, _ = json.Marshal(resp)
			}

			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write(b)
		}
	}()

	r = r.WithContext(graphql.StartOperationTrace(r.Context()))

	transport := s.getTransport(r)
	if transport == nil {
		sendErrorf(w, http.StatusBadRequest, "transport not supported")
		return
	}

	transport.Do(w, r, s.exec)
}

func (s *Server) ExecGraphCommand(ctx context.Context, params *graphql.RawParams) (*graphql.Response, error) {
	var response *graphql.Response
	defer func() {
		if err := recover(); err != nil {
			err := s.exec.PresentRecoveredError(ctx, err)
			response = &graphql.Response{Errors: []*gqlerror.Error{err}}
		}
	}()

	// Deliberately assigning value to `response` so that we can also capture the value from `defer func() block`
	ctx = graphql.StartOperationTrace(ctx)
	rc, err := s.exec.CreateOperationContext(ctx, params)
	if err != nil {
		response = s.exec.DispatchError(graphql.WithOperationContext(ctx, rc), err)
		return response, nil
	}
	responses, responseContext := s.exec.DispatchOperation(ctx, rc)
	response = responses(responseContext)
	return response, nil
}

type SubscriptionHandler struct {
	operationContext *graphql.OperationContext
	exec             *executor.Executor
	ctx              context.Context
	Response         *graphql.Response
	PanicHandler     func() *gqlerror.Error
}

func (s SubscriptionHandler) Exec() (graphql.ResponseHandler, context.Context) {
	return s.exec.DispatchOperation(s.ctx, s.operationContext)
}

func (s *Server) ExecGraphSubscriptionsCommand(ctx context.Context, params *graphql.RawParams) (SubscriptionHandler, error) {
	ctx = graphql.StartOperationTrace(ctx)
	rc, err := s.exec.CreateOperationContext(ctx, params)
	if err != nil {
		resp := s.exec.DispatchError(graphql.WithOperationContext(ctx, rc), err)
		switch errcode.GetErrorKind(err) {
		case errcode.KindProtocol:
			return SubscriptionHandler{}, resp.Errors
		default:
			return SubscriptionHandler{Response: &graphql.Response{Errors: err}}, nil
		}
	}

	ctx = graphql.WithOperationContext(ctx, rc)

	ctx, cancel := context.WithCancel(ctx)
	//c.mu.Lock()
	//c.active[msg.id] = cancel
	//c.mu.Unlock()

	subHandler := SubscriptionHandler{}
	subHandler.operationContext = rc
	subHandler.ctx = ctx
	subHandler.exec = s.exec
	subHandler.PanicHandler = func() *gqlerror.Error {
		defer cancel()
		if r := recover(); r != nil {
			err := subHandler.operationContext.Recover(ctx, r)
			var gqlerr *gqlerror.Error
			if !errors.As(err, &gqlerr) {
				gqlerr = &gqlerror.Error{}
				if err != nil {
					gqlerr.Message = err.Error()
					return gqlerr
				}
			}
		}
		//c.complete(msg.id)
		//c.mu.Lock()
		//delete(c.active, msg.id)
		//c.mu.Unlock()
		return nil
	}
	return subHandler, nil
}

func sendError(w http.ResponseWriter, code int, errors ...*gqlerror.Error) {
	w.WriteHeader(code)
	b, err := json.Marshal(&graphql.Response{Errors: errors})
	if err != nil {
		panic(err)
	}
	w.Write(b)
}

func sendErrorf(w http.ResponseWriter, code int, format string, args ...interface{}) {
	sendError(w, code, &gqlerror.Error{Message: fmt.Sprintf(format, args...)})
}

type OperationFunc func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler

func (r OperationFunc) ExtensionName() string {
	return "InlineOperationFunc"
}

func (r OperationFunc) Validate(schema graphql.ExecutableSchema) error {
	if r == nil {
		return fmt.Errorf("OperationFunc can not be nil")
	}
	return nil
}

func (r OperationFunc) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	return r(ctx, next)
}

type ResponseFunc func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response

func (r ResponseFunc) ExtensionName() string {
	return "InlineResponseFunc"
}

func (r ResponseFunc) Validate(schema graphql.ExecutableSchema) error {
	if r == nil {
		return fmt.Errorf("ResponseFunc can not be nil")
	}
	return nil
}

func (r ResponseFunc) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	return r(ctx, next)
}

type FieldFunc func(ctx context.Context, next graphql.Resolver) (res interface{}, err error)

func (f FieldFunc) ExtensionName() string {
	return "InlineFieldFunc"
}

func (f FieldFunc) Validate(schema graphql.ExecutableSchema) error {
	if f == nil {
		return fmt.Errorf("FieldFunc can not be nil")
	}
	return nil
}

func (f FieldFunc) InterceptField(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
	return f(ctx, next)
}
