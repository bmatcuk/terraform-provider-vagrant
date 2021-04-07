![Release](https://img.shields.io/github/release/bmatcuk/terraform-provider-vagrant.svg?branch=master)
[![Build Status](https://github.com/bmatcuk/terraform-provider-vagrant/actions/workflows/release.yml/badge.svg)](https://github.com/bmatcuk/terraform-provider-vagrant/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/bmatcuk/terraform-provider-vagrant)](https://goreportcard.com/report/github.com/bmatcuk/terraform-provider-vagrant)

# terraform-provider-vagrant
A Vagrant provider for terraform 0.12+.

A note about lippertmarkus/vagrant in the registry: when I originally wrote
this provider, the terraform registry didn't exist. My terraform needs waned
and I didn't hear about the registry until some time later. lippertmarkus
forked my provider and published to the registry as a convenience. Thanks! But,
it's just an older version of this exact same codebase. So, I recommend you use
bmatcuk/vagrant to get the latest updates instead.

## Installation
Add bmatcuk/vagrant to [required_providers]:

```hcl
terraform {
  required_providers {
    vagrant = {
      source  = "bmatcuk/vagrant"
      version = "~> 4.0.0"
    }
  }
}
```

## Usage
```hcl
resource "vagrant_vm" "my_vagrant_vm" {
  vagrantfile_dir = "path/to/dir"
  env = {
    KEY = "value",
  }
  get_ports = true
}
```

**vagrantfile_dir** is the path to a directory where a Vagrantfile lives. The
Vagrantfile must exist when terraform runs or else it will throw an error. This
option defaults to `.`, ie, the current directory and you may set this value to
absolute or relative paths.

**env** is a map of additional environment variables to pass to the Vagrantfile.
The environment variables set by the calling process are always passed.

**get_ports** if `true`, information about forwarded ports will be filled in
(see `ports` below). This is `false` by default because it may take some time
to run.

If you have multiple Vagrantfiles, provide an `alias` in the `provider` block
and use the `provider` meta-argument in the resource/data-source
configurations.

### Outputs
* `machine_names.#` - a list of machine names as defined in the Vagrantfile.
* `ssh_config.#` - SSH connection info. Since a Vagrantfile may create multiple
  machines, this is a list with the following variables:

  * `ssh_config.*.type` - always "ssh" for now
  * `ssh_config.*.user` - the user for the connection
  * `ssh_config.*.host` - the address to connect to
  * `ssh_config.*.port` - the port to connect to
  * `ssh_config.*.private_key` - private ssh key for the connection
  * `ssh_config.*.agent` - whether or not to use the agent for authentication
    (always "false" for now).

  If there is only one machine built by the Vagrantfile, the connection info
  will be set in the `resource` block so you can include provisioners without
  any additional configuration. However, if there is more than one machine, the
  connection info will not be set; you'll need to create some `null_resources`
  to do your provisioning.
* `ports.#` - information about forwarded ports if `get_ports` is `true`. This
  is a list of lists: for each machine in the Vagrantfile, `ports` will have a
  list with the following variables:

  * `ports.*.*.guest` - the port on the guest VM
  * `ports.*.*.host` - the host port forwarded to the guest VM

Note that `machine_names`, `ssh_config`, and `ports` are guaranteed to be in
the same order (ie, `ssh_config[0]` is the corresponding config for the machine
named `machine_names[0]`), but the order is undefined (ie, don't count on
`machine_names[0]` being the first machine defined in the Vagrantfile).

## Forcing an Update
The easiest way to force an update is to set, or change the value of, some
environment variable. This will signal to terraform that the vagrant_vm
resource needs to update.

For example, if you want to force updates when your Vagrantfile changes, try
something like this:

```hcl
resource "vagrant_vm" "my_vagrant_vm" {
  vagrantfile_dir = "path/to/dir"
  env = {
    VAGRANTFILE_HASH = md5(file("path/to/dir/Vagrantfile")),
  }
}
```

When the file changes, the hash will change, and terraform will ask for an
update.

## Removing Machines
Sadly, due to some limitations in vagrant, it's not possible to automatically
remove a portion of machines from a Vagrantfile. In other words, if your
Vagrantfile defines 5 machines and you remove 2 of them from the Vagrantfile,
they will be left running in your vagrant provider (ie, virtualbox or whatever)
with no way of removing them via vagrant (or terraform).

If you intend of removing some machines, you should manually run `vagrant
destroy MACHINE_NAME` on those machines you wish to remove *before* editing the
Vagrantfile. Then update your Vagrantfile and allow terraform to do the rest.

If you forget, you can manually cleanup these old VMs by launching your vagrant
provider's UI and deleting the machines. Then run `vagrant global-status
--prune` to cleanup vagrant's cache of these machines.

## Debugging
If terrafrom is failing on the vagrant step, you can get additional output by
running terraform with logging output enabled. Try something like:

```bash
env TF_LOG=TRACE terraform apply ...
```

And, of course, you can always run vagrant on your Vagrantfile directly.

## Local Development
The example in `examples/resources/vagrant_vm` is fully functioning, but you'll
need to compile this provider and put it in a place terraform can find it:

```bash
go build
mkdir -p examples/resources/vagrant_vm/terraform.d/plugins/registry.terraform.io/bmatcuk/vagrant/4.0.0/darwin_amd64
mv terraform-provider-vagrant examples/resources/vagrant_vm/terraform.d/plugins/registry.terraform.io/bmatcuk/vagrant/4.0.0/darwin_amd64/
cd examples/resources/vagrant_vm
terraform init
terraform apply
```

Adjust `darwin_amd64` to match your system.

[required_providers]: https://www.terraform.io/docs/language/providers/requirements.html#requiring-providers
