package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func init() {
	schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
		desc := s.Description
		if s.Default != nil {
			desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
		}
		if s.Deprecated != "" {
			desc += " " + s.Deprecated
		}
		return desc
	}
}

// Provider returns the terraform provider schema.
func NewVagrantProvider(version string) func() *schema.Provider {
	return func() *schema.Provider {
		return &schema.Provider{
			ConfigureContextFunc: vagrantConfigure(version),

			ResourcesMap: map[string]*schema.Resource{
				"vagrant_vm": resourceVagrantVM(),
			},
		}
	}
}

func vagrantConfigure(version string) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return VagrantConfig{}, nil
	}
}
