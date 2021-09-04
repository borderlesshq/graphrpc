// Code generated by github.com/Just4Ease/graphrpc, DO NOT EDIT.

package vendorService

import (
	"context"

	axon "github.com/Just4Ease/axon/v2"
	"github.com/Just4Ease/graphrpc"
	"github.com/Just4Ease/graphrpc/client"
)

type ServiceClient struct {
	client *client.Client
}

func NewClient(conn axon.EventStore, options ...graphrpc.ClientOption) (*ServiceClient, error) {
	client, err := graphrpc.NewClient(conn, options...)
	if err != nil {
		return nil, err
	}

	return &ServiceClient{client: client}, nil
}

type Query struct {
	ListVendors   *VendorList "json:\"listVendors\" graphql:\"listVendors\""
	GetVendorByID *Vendor     "json:\"getVendorById\" graphql:\"getVendorById\""
}
type Mutation struct {
	CreateVendor     *Result "json:\"createVendor\" graphql:\"createVendor\""
	UpdateVendor     *Result "json:\"updateVendor\" graphql:\"updateVendor\""
	ActivateVendor   *Result "json:\"activateVendor\" graphql:\"activateVendor\""
	DeactivateVendor *Result "json:\"deactivateVendor\" graphql:\"deactivateVendor\""
}
type ListVendors struct {
	ListVendors *struct {
		Count int "json:\"count\" graphql:\"count\""
		Data  []*struct {
			ID      string "json:\"id\" graphql:\"id\""
			Name    string "json:\"name\" graphql:\"name\""
			Country string "json:\"country\" graphql:\"country\""
			Address string "json:\"address\" graphql:\"address\""
		} "json:\"data\" graphql:\"data\""
	} "json:\"listVendors\" graphql:\"listVendors\""
}

const ListVendorsDocument = `query listVendors {
	listVendors(filters: {cursorId:"",previousPage:false,iso2:"",stateCode:"",limit:2,userId:""}) {
		count
		data {
			id
			name
			country
			address
		}
	}
}
`

func (c *ServiceClient) ListVendors(ctx context.Context, headers client.Header) (*ListVendors, error) {
	vars := map[string]interface{}{}

	var res ListVendors
	if err := c.client.Exec(ctx, "listVendors", ListVendorsDocument, &res, vars, headers); err != nil {
		return nil, err
	}

	return &res, nil
}
