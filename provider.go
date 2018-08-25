package main

import (
	"github.com/hashicorp/terraform/helper/schema"
)

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
