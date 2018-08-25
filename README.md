![Release](https://img.shields.io/github/release/bmatcuk/terraform-provider-vagrant.svg?branch=master)
[![Build Status](https://travis-ci.com/bmatcuk/terraform-provider-vagrant.svg?branch=master)](https://travis-ci.com/bmatcuk/terraform-provider-vagrant)

# terraform-provider-vagrant
A Vagrant provider for terraform.

## Installation
TODO

## Usage
```hcl
resource "vagrant_vm" "my_vagrant_vm" {
  vagrantfile_dir = "path/to/dir"
}
```

**vagrantfile_dir** is the path to a directory where a Vagrantfile lives. The
Vagrantfile must exist when terraform runs or else it will throw an error. This
option defaults to `.`, ie, the current directory and you may set this value to
absolute or relative paths.

### Outputs
* `ssh_config.#` - SSH connection info. Since a Vagrantfile may create multiple
  machines, this is a list with the following variables:

  * `ssh_config.*.type` - always "ssh" for now
  * `ssh_config.*.user` - the user for the connection
  * `ssh_config.*.host` - the address to connect to
  * `ssh_config.*.port` - the port to connect to
  * `ssh_config.*.private_key` - private ssh key for the connection
  * `ssh_config.*.agent` - whether or not to use the agent for authentication
    (always "false" for now).
