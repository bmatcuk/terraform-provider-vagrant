# -*- mode: ruby -*-
# vi: set ft=ruby :

required_plugins = %w(vagrant-ignition)
plugins_to_install = required_plugins.reject(&Vagrant.method(:has_plugin?))
unless plugins_to_install.empty?
  puts "Installing plugins: #{plugins_to_install.join(', ')}"
  if system "vagrant plugin install #{plugins_to_install.join(' ')}"
    exec "vagrant #{ARGV.join(' ')}"
  else
    abort 'Installation of one or more plugins failed.'
  end
end

$vm_cpus = 1
$vm_memory = 1024

Vagrant.configure("2") do |config|
  config.ssh.insert_key = false
  config.ssh.forward_agent = true

  config.vm.box = 'coreos-stable'
  config.vm.box_url = 'https://stable.release.core-os.net/amd64-usr/current/coreos_production_vagrant_virtualbox.json'

  if Vagrant.has_plugin? 'vagrant-vbguest'
    config.vbguest.auto_update = false
  end

  config.vm.provider :virtualbox do |vb|
    vb.gui = false
    vb.cpus = $vm_cpus
    vb.memory = $vm_memory
    vb.check_guest_additions = false
    vb.functional_vboxsf = false
    config.ignition.enabled = true
    config.ignition.config_obj = vb
  end
end
