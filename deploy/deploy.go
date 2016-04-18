package deploy

import (
	"bytes"
	"html/template"
	"log"
	"os"
	"os/exec"
)

//Raw deploys a raw installation to a target
func Raw(node, target string) {
	type DeployData struct {
		Node   string
		Target string
	}

	tmplString := `pushd {{.Node}}

	keys="$(find pki/pki/private -type f ! -name 'ca.key')"
	keys+=" $(find foreignKeys/ -type f)"
	keys+=" $(find pki/pki/issued -type f) pki/pki/ca.crt"
	keys+=" pki/pki/dh.pem"

	services="$(find configs -name "*.service" -exec basename {} \;)"

	ssh {{.Target}} "rm -rf ~/.susi-dev-temp/* && mkdir -p ~/.susi-dev-temp/keys && mkdir ~/.susi-dev-temp/configs && mkdir ~/.susi-dev-temp/assets"

	scp $keys {{.Target}}:~/.susi-dev-temp/keys/
	scp configs/* {{.Target}}:~/.susi-dev-temp/configs/
	scp -r assets/* {{.Target}}:~/.susi-dev-temp/assets/

	sshCommand="sudo mkdir -p /etc/susi/keys && sudo mkdir -p /usr/share/susi"
	sshCommand+=" && sudo cp ~/.susi-dev-temp/configs/*.json /etc/susi/ || true"
	sshCommand+=" && sudo cp ~/.susi-dev-temp/configs/*.ovpn /etc/susi/ || true"
	sshCommand+=" && sudo cp ~/.susi-dev-temp/configs/*.service /etc/systemd/system/ || true"
	sshCommand+=" && sudo cp ~/.susi-dev-temp/keys/* /etc/susi/keys/ || true"
	sshCommand+=" && sudo cp -rf ~/.susi-dev-temp/assets/* /usr/share/susi/ || true"
	sshCommand+=" && sudo systemctl daemon-reload"
	sshCommand+=" && sudo systemctl enable $services"
	sshCommand+=" && sudo systemctl restart $services"

	ssh {{.Target}} "$(echo $sshCommand)"

  exit 0
  `

	tmpl := template.Must(template.New("").Parse(tmplString))
	buff := bytes.Buffer{}
	tmpl.Execute(&buff, DeployData{node, target})

	cmd := exec.Command("/bin/bash", "-c", buff.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}

}
