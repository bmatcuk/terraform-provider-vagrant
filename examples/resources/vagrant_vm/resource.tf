resource "vagrant_vm" "my_vagrant_vm" {
  env = {
    # force terraform to re-run vagrant if the Vagrantfile changes
    VAGRANTFILE_HASH = md5(file("./Vagrantfile")),
  }
  get_ports = true
  # see schema for additional options
}
