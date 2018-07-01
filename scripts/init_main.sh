#!/bin/bash -e

# Locales
locale-gen en_GB
locale-gen en_GB.UTF-8
update-locale en_GB

export DEBIAN_FRONTEND=noninteractive

# Install Go
GOVER="1.10.3"
wget -qO- https://dl.google.com/go/go${GOVER}.linux-amd64.tar.gz | \
  tar zx -C /usr/local/
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin:~/go/bin' >> ~vagrant/.bashrc

export MYPATH=~vagrant/go/src/github.com/gbevan/goswim

. $MYPATH/scripts/init_mongodb.sh
. $MYPATH/scripts/init_vault.sh

# Ready!
echo '========================================================='
echo 'Vault server running in DEV mode.  root-token-id is root'
