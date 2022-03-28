#!/bin/bash
# install all dependencies for Debian or Ubuntu

# test for distro type
# could test for lsb_release missing and try and install
which lsb_release

if [ $? -ne 0 ]
then
    echo "Error: missing lsb_release"
    exit 1
fi

DISTRO=$(lsb_release -d|awk '{ print $2 }')
echo $DISTRO " detected"

if [ $DISTRO != "Debian" ] && [ $DISTRO != "Ubuntu" ]
then
    echo "Error: distribution must be Debian or Ubuntu"
    exit 1
fi

sudo apt-get update

# Golang 1.18
curl -O -L  https://go.dev/dl/go1.18.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.18.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo "export PATH=$PATH:/usr/local/go/bin" >> $HOME/.profile
 
# needed by Go
sudo apt install -y build-essential

# to get golang dependencies and code
sudo apt install -y git

# to optionally use Makefile
sudo apt install -y make

# optional Docker install 
sudo apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

if [ $DISTRO == "Debian" ]
then
    curl -fsSL https://download.docker.com/linux/debian/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

    echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian \
    $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

elif [ $DISTRO == "Ubuntu" ]
then
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

    echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
    $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
fi

sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io

# Since Debian 10 Buster iptables is being replaced by nftables, let's use iptables for now
if [ $DISTRO == "Debian" ]
then
sudo update-alternatives --set iptables /usr/sbin/iptables-legacy
fi