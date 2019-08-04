# -*- mode: ruby -*-
# vi: set ft=ruby :

extras = "~vagrant/gostint/scripts/init_main.sh"

VAGRANTFILE_API_VERSION = "2"
Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  config.vm.define "gostint-dev", primary: true do |ubuntu|
    ubuntu.vm.provision "shell", inline: extras
    ubuntu.vm.synced_folder ".", "/home/vagrant/gostint"
    ubuntu.vm.provider "docker" do |d|
      d.image = "gbevan/vagrant-ubuntu-dev:bionic"
      d.has_ssh = true
      d.ports = ["3232:3232", "8300:8200", "27017:27017"]
      d.privileged = true # needed for dind
      d.volumes = [
        "/etc/localtime:/etc/localtime:ro",
        "/etc/timezone:/etc/timezone:ro"
      ]
    end
  end
end
