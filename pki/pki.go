package pki

import (
	"fmt"
	"log"
	"os/exec"
)

// Init the pki in a directory
func Init(directory string) {
	createScript := fmt.Sprintf(`
    mkdir -p %v
    pushd %v
    wget https://github.com/OpenVPN/easy-rsa/releases/download/3.0.1/EasyRSA-3.0.1.tgz
    tar xfvz EasyRSA-3.0.1.tgz
    mv EasyRSA-3.0.1/* .
    rm -r EasyRSA-3.0.1
    rm EasyRSA-3.0.1.tgz
    ./easyrsa init-pki
    echo "" | ./easyrsa build-ca nopass
    popd
  `, directory, directory)
	cmd := exec.Command("/bin/bash", "-c", createScript)
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}
}

// CreateCertificate creates and signes a certificate/key pair
func CreateCertificate(directory, name string) {
	createScript := fmt.Sprintf(`
    pushd %v
    ./easyrsa build-client-full %v nopass
    popd
  `, directory, name)
	cmd := exec.Command("/bin/bash", "-c", createScript)
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}
}
