package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

// Provider returns the terraform provider schema.
func Provider() *schema.Provider {
	return &schema.Provider{
		ConfigureFunc: vagrantConfigure,

		ResourcesMap: map[string]*schema.Resource{
			"vagrant_vm": resourceVagrantVM(),
		},
	}
}

func vagrantConfigure(d *schema.ResourceData) (interface{}, error) {
	return VagrantConfig{}, nil
}
