---
page_title: "Provider: Vagrant"
description: |-
  Integrate vagrant into terraform.
---

# Vagrant Provider
A Vagrant provider for terraform 0.12+.

A note about lippertmarkus/vagrant in the registry: when I originally wrote
this provider, the terraform registry didn't exist. My terraform needs waned
and I didn't hear about the registry until some time later. lippertmarkus
forked my provider and published to the registry as a convenience. Thanks! But,
it's just an older version of this exact same codebase. So, I recommend you use
bmatcuk/vagrant to get the latest updates instead.

## Installation
Add bmatcuk/vagrant to [required_providers](https://www.terraform.io/docs/language/providers/requirements.html#requiring-providers):

```terraform
terraform {
  required_providers {
    vagrant = {
      source  = "bmatcuk/vagrant"
      version = "~> 4.0.0"
    }
  }
}
```

## Example Usage

```terraform
provider "vagrant" {
  # no config
}
```

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
