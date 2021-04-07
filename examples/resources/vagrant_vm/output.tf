output "host_port" {
  value = vagrant_vm.my_vagrant_vm.ports[0][0].host
}
