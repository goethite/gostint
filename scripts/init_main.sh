#!/bin/bash -e

# Locales
locale-gen en_GB
locale-gen en_GB.UTF-8
update-locale en_GB

export DEBIAN_FRONTEND=noninteractive

# Install docker
apt update
apt install -y \
  apt-transport-https \
  ca-certificates \
  curl \
  software-properties-common
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
apt-key fingerprint 0EBFCD88
add-apt-repository \
  "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) \
  stable"
apt update
apt install -y docker-ce
gpasswd -a vagrant docker
dockerd >/tmp/docker.log 2>&1 &

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
