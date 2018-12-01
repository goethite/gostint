#!/bin/bash -e

GOVER="1.11.2"
NODEVER="10"
DOCKERVER="18.06.1~ce~3-0~ubuntu"  # match Dockerfile

apt update
apt install locales

# Locales
locale-gen en_GB
locale-gen en_GB.UTF-8
update-locale en_GB

export DEBIAN_FRONTEND=noninteractive

# Install docker
# apt update
apt install -y \
  apt-transport-https \
  ca-certificates \
  curl \
  software-properties-common \
  bats \
  uuid
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
apt-key fingerprint 0EBFCD88
add-apt-repository \
  "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) \
  stable"
apt update
apt install -y docker-ce=${DOCKERVER}
gpasswd -a vagrant docker

# install nodejs
curl -sL https://deb.nodesource.com/setup_${NODEVER}.x | bash -
apt-get install -y nodejs

# Start dockerd
dockerd -s vfs >/tmp/docker.log 2>&1 &

# Install Go
wget -qO- https://dl.google.com/go/go${GOVER}.linux-amd64.tar.gz | \
  tar zx -C /usr/local/
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin:~/go/bin' >> ~vagrant/.bashrc

export MYPATH=~vagrant/go/src/github.com/gbevan/gostint

. $MYPATH/scripts/init_mongodb.sh
. $MYPATH/scripts/init_vault.sh

echo "Creating self signed cert"
su - vagrant -c "echo -e 'GB\n\n\ngostint\n\n$(hostname)\n\n' | openssl req  -nodes -new -x509  -keyout go/src/github.com/gbevan/gostint/etc/key.pem -out go/src/github.com/gbevan/gostint/etc/cert.pem -days 365 2>&1 && chmod 644 go/src/github.com/gbevan/gostint/etc/key.pem"

# Ready!
echo '========================================================='
echo 'Vault server running in DEV mode.  root-token-id is root'
