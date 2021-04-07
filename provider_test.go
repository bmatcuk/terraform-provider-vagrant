package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var providerFactories = map[string]func() (*schema.Provider, error){
	"vagrant": func() (*schema.Provider, error) {
		return NewVagrantProvider("dev")(), nil
	},
}

func TestProvider(t *testing.T) {
	if err := NewVagrantProvider("test")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
