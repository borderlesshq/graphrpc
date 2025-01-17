package clientgen

import (
	"fmt"

	"github.com/borderlesshq/graphrpc/libs/99designs/gqlgen/codegen/config"
	"github.com/borderlesshq/graphrpc/libs/99designs/gqlgen/codegen/templates"
)

func RenderTemplate(cfg *config.Config,
	query *Query,
	mutation *Mutation,
	fragments []*Fragment,
	operations []*Operation,
	operationResponses []*OperationResponse,
	generateClient bool,
	client config.PackageConfig,
	remoteServiceName, remoteServiceGraphEntrypoint string) error {
	if err := templates.Render(templates.Options{
		PackageName: client.Package,
		Filename:    client.Filename,
		Data: map[string]interface{}{
			"RemoteServiceName":            remoteServiceName,
			"RemoteServiceGraphEntrypoint": remoteServiceGraphEntrypoint,
			"Query":                        query,
			"Mutation":                     mutation,
			"Fragment":                     fragments,
			"Operation":                    operations,
			"OperationResponse":            operationResponses,
			"GenerateClient":               generateClient,
		},
		Packages:   cfg.Packages,
		PackageDoc: "// Code generated by github.com/borderlesshq/graphrpc, DO NOT EDIT.\n",
	}); err != nil {
		return fmt.Errorf("%s generating failed: %w", client.Filename, err)
	}

	return nil
}
