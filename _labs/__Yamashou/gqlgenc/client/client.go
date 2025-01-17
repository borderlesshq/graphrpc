package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/borderlesshq/axon/v2"
	"github.com/borderlesshq/axon/v2/messages"
	"github.com/borderlesshq/axon/v2/options"
	"github.com/borderlesshq/graphrpc/libs/Yamashou/gqlgenc/graphqljson"
	"github.com/borderlesshq/graphrpc/utils"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"log"
	"strings"
)

type Options struct {
	Headers               Header
	remoteGraphEntrypoint string
	remoteServiceName     string
	applyMsgpackEncoder   bool
}

type Option func(*Options) error

// SetHeader sets the headers for this client. Note, duplicate headers are replaced with the newest value provided
func SetHeader(key, value string) Option {
	return func(options *Options) error {
		if options.Headers == nil {
			options.Headers = make(map[string]string)
		}
		options.Headers[key] = value
		return nil
	}
}

// SetRemoteGraphQLPath is used to set the graphql path of the remote service for generation to occur.
func SetRemoteGraphQLPath(path string) Option {
	return func(o *Options) error {
		if strings.TrimSpace(path) == "" {
			return errors.New("GraphQL entrypoint path is required!")
		}

		// Detect 1st in api/graph entrypoint and strip it
		if path[:1] == "/" {
			path = path[1:]
		}

		o.remoteGraphEntrypoint = path
		return nil
	}
}

// SetRemoteServiceName is used to set the service name of the remote service for this client.
func SetRemoteServiceName(remoteServiceName string) Option {
	return func(o *Options) error {
		if strings.TrimSpace(remoteServiceName) == "" {
			return errors.New("Remote GraphRPC Service name is required!")
		}

		o.remoteServiceName = remoteServiceName
		return nil
	}
}

// ApplyMsgPackEncoder is used to enable internal msgpack encoding over encoding/json
func ApplyMsgPackEncoder() Option {
	return func(o *Options) error {
		o.applyMsgpackEncoder = true
		return nil
	}
}

type Header = map[string]string

// Client is the http client wrapper
type Client struct {
	axonConn            axon.EventStore
	opts                *Options
	BaseURL             string
	Headers             Header
	applyMsgPackEncoder bool
}

// Request represents an outgoing GraphQL request
type Request struct {
	Query         string                 `json:"query" msgpack:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty" msgpack:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty" msgpack:"operationName,omitempty"`
}

// NewClient creates a new http client wrapper
func NewClient(conn axon.EventStore, options ...Option) (*Client, error) {
	opts := &Options{Headers: map[string]string{}}

	for _, option := range options {
		if err := option(opts); err != nil {
			log.Printf("failed to apply client option: %v", err)
			return nil, err
		}
	}

	if opts.remoteGraphEntrypoint == "" {
		log.Print("using default GraphRPC remote graph entrypoint path: '/graphql'...")
		opts.remoteGraphEntrypoint = "graphql"
	}

	if opts.remoteServiceName == "" {
		return nil, errors.New("remote graphrpc service name is required!")
	}

	if conn == nil {
		panic("axon must not be nil. see github.com/borderlesshq/axon for more details on how to connect")
	}

	return &Client{
		axonConn:            conn,
		BaseURL:             fmt.Sprintf("%s.%s", opts.remoteServiceName, opts.remoteGraphEntrypoint),
		opts:                opts,
		Headers:             opts.Headers,
		applyMsgPackEncoder: opts.applyMsgpackEncoder,
	}, nil
}

func (c *Client) exec(ctx context.Context, operationName, query string, variables map[string]interface{}, headers Header) ([]byte, error) {
	if headers == nil {
		headers = make(map[string]string)
	}
	r := &Request{
		Query:         query,
		Variables:     variables,
		OperationName: operationName,
	}

	var requestBody []byte
	var err error
	pubOptions := make([]options.PublisherOption, 0)
	if c.applyMsgPackEncoder {
		requestBody, err = utils.Marshal(r)
		headers["Content-Type"] = "application/msgpack"
		pubOptions = append(pubOptions, options.SetPubContentType("application/msgpack"))
	} else {
		requestBody, err = json.Marshal(r)
		headers["Content-Type"] = "application/json"
		pubOptions = append(pubOptions, options.SetPubContentType("application/json"))
	}

	pubOptions = append(pubOptions, options.SetPubHeaders(headers), options.SetPubContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	mg, err := c.axonConn.Request(c.BaseURL, requestBody, pubOptions...)
	if err != nil {
		return nil, err
	}

	if mg.Type == messages.ErrorMessage {
		return nil, errors.New(mg.Error)
	}

	if mg.ContentType == "application/msgpack" {
		c.applyMsgPackEncoder = true
	} else {
		c.applyMsgPackEncoder = false
	}
	return mg.Body, nil
}

// GqlErrorList is the struct of a standard graphql error response
type GqlErrorList struct {
	Errors gqlerror.List `json:"errors" msgpack:"errors"`
}

func (e *GqlErrorList) Error() string {
	return e.Errors.Error()
}

// HTTPError is the error when a GqlErrorList cannot be parsed
type HTTPError struct {
	Code    int    `json:"code" msgpack:"code"`
	Message string `json:"message" msgpack:"message"`
}

// ErrorResponse represent an handled error
type ErrorResponse struct {
	// populated when http status code is not OK
	NetworkError *HTTPError `json:"networkErrors" msgpack:"networkErrors"`
	// populated when http status code is OK but the server returned at least one graphql error
	GqlErrors *gqlerror.List `json:"graphqlErrors" msgpack:"graphqlErrors"`

	applyMsgPackEncoder bool
}

// HasErrors returns true when at least one error is declared
func (er *ErrorResponse) HasErrors() bool {
	return er.NetworkError != nil || er.GqlErrors != nil
}

func (er *ErrorResponse) Error() string {
	var content []byte
	var err error
	if er.applyMsgPackEncoder {
		content, err = utils.Marshal(er)
	} else {
		content, err = json.Marshal(er)
	}

	if err != nil {
		return err.Error()
	}

	return string(content)
}

// Exec is used to prepare and make a network call over the underlying network stream.
// When data is returned it is parsed with respect to the document structure.
func (c *Client) Exec(ctx context.Context, operationName, query string, respData interface{}, vars map[string]interface{}, headers Header) error {
	result, err := c.exec(ctx, operationName, query, vars, headers)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	isIntrospection := false
	if operationName == "introspect" {
		isIntrospection = true
	}
	return c.parseResponse(result, 200, respData, isIntrospection)
}

func (c *Client) ServiceName() string {
	return c.opts.remoteServiceName
}

func (c *Client) parseResponse(body []byte, httpCode int, result interface{}, isIntrospection bool) error {
	errResponse := &ErrorResponse{}
	isKOCode := httpCode < 200 || 299 < httpCode
	if isKOCode {
		errResponse.NetworkError = &HTTPError{
			Code:    httpCode,
			Message: fmt.Sprintf("Response body %s", string(body)),
		}
	}

	// some servers return a graphql error with a non OK http code, try anyway to parse the body
	if err := c.unmarshal(body, result, isIntrospection); err != nil {
		if gqlErr, ok := err.(*GqlErrorList); ok {
			errResponse.GqlErrors = &gqlErr.Errors
		} else if !isKOCode { // if is KO code there is already the http error, this error should not be returned
			return err
		}
	}
	errResponse.applyMsgPackEncoder = c.applyMsgPackEncoder
	if errResponse.HasErrors() {
		return errResponse
	}

	return nil
}

// response is a GraphQL layer response from a handler.
type response struct {
	Data   utils.RawMessage `json:"data" cbor:"data"`
	Errors utils.RawMessage `json:"errors" cbor:"errors"`
}

func (c *Client) unmarshal(data []byte, res interface{}, isIntrospection bool) error {
	resp := response{}

	if c.applyMsgPackEncoder {
		if err := utils.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("failed to decode (msgpack) data %s: %w", string(data), err)
		}
	} else {
		if err := json.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("failed to decode (json) data %s: %w", string(data), err)
		}
	}

	//if err := cbor.Unmarshal(data, &resp); err != nil {
	//	return fmt.Errorf("failed to decode (json) data %s: %w", string(data), err)
	//}

	if resp.Errors != nil && len(resp.Errors) > 0 {
		// try to parse standard graphql error
		errors := &GqlErrorList{}
		if c.applyMsgPackEncoder {
			if e := utils.Unmarshal(data, errors); e != nil {
				return fmt.Errorf("faild to parse graphql (msgpack) errors. Response content %s - %w ", string(data), e)
			}
		} else {
			if e := json.Unmarshal(data, errors); e != nil {
				return fmt.Errorf("faild to parse graphql (json) errors. Response content %s - %w ", string(data), e)
			}
		}

		//if e := cbor.Unmarshal(data, errors); e != nil {
		//	return fmt.Errorf("faild to parse graphql (msgpack) errors. Response content %s - %w ", string(data), e)
		//}

		return errors
	}

	if !isIntrospection {
		if err := graphqljson.UnmarshalData(resp.Data, res); err != nil {
			return fmt.Errorf("failed to decode data into response %s: %w", string(data), err)
		}

		return nil

	}

	if c.applyMsgPackEncoder {
		if err := utils.UnPack(resp.Data, res); err != nil {
			return fmt.Errorf("failed to decode data into response %s: %w", string(data), err)
		}
	} else {
		if err := json.Unmarshal(resp.Data, res); err != nil {
			return fmt.Errorf("failed to decode data into response %s: %w", string(data), err)
		}
	}

	//if err := json.Unmarshal(resp.Data, res); err != nil {
	//	return fmt.Errorf("failed to decode data into response %s: %w", string(resp.Data), err)
	//}

	//dec := cbor.NewDecoder(bytes.NewBuffer(resp.Data))
	//
	//if err := dec.Decode(res); err != nil {
	//	return fmt.Errorf("failed to decode data into response %s: %w", string(data), err)
	//}
	//
	return nil
}
