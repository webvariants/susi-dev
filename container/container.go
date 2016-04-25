package container

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

//Prepare prepares a pod
func Prepare(node string) (uuid string) {
	script := fmt.Sprintf("sudo rkt prepare %v/containers/*.aci", node)
	cmd := exec.Command("/bin/bash", "-c", script)
	var out bytes.Buffer
	cmd.Stderr = os.Stderr
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}
	uuid = strings.Trim(out.String(), "\n")
	return uuid
}

// Run starts a pod for a node
func Run(uuid, ip string) (systemdID string) {
	script := fmt.Sprintf("sudo systemd-run -p Environment=CNI_ARGS=IP=%v rkt run-prepared %v", ip, uuid)
	cmd := exec.Command("/bin/bash", "-c", script)
	var out bytes.Buffer
	cmd.Stderr = &out
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}
	text := out.String()
	words := strings.Split(text, " ")
	last := words[3]
	words = strings.Split(last, ".")
	systemdID = words[0]
	return systemdID
}
