package components

import (
	"bytes"
	"html/template"
)

type susiSerialComponent struct{}

func (p *susiSerialComponent) Config() string {
	return `{
    "susi-addr": "localhost",
    "susi-port": 4000,
    "cert": "/etc/susi/keys/susi-serial.crt",
    "key": "/etc/susi/keys/susi-serial.key",
    "component": {
      "ports" : [
        {
          "id" : "arduino",
          "port" : "/dev/ttyUSB0",
          "baudrate" : 9600
        }
      ]
	  }
  }`
}

func (p *susiSerialComponent) StartCommand() string {
	return "/usr/local/bin/susi-serial -c /etc/susi/susi-serial.json"
}

func (p *susiSerialComponent) BuildContainer(node, gpgpass string) {
	buildBaseContainer()
	templateString := `

	acbuild --debug begin .containers/susi-base-latest-linux-amd64.aci

  acbuild --debug set-name susi.io/susi-serial

  acbuild --debug copy .build/alpine/bin/susi-serial /usr/local/bin/susi-serial
  acbuild --debug copy {{.Node}}/pki/pki/issued/susi-serial.crt /etc/susi/keys/susi-serial.crt
  acbuild --debug copy {{.Node}}/pki/pki/private/susi-serial.key /etc/susi/keys/susi-serial.key
  acbuild --debug copy {{.Node}}/configs/susi-serial.json /etc/susi/susi-serial.json || true
  for asset in $(find {{.Node}}/assets -type f); do
    acbuild --debug copy $asset /usr/share/susi/$(echo $asset|cut -d\/ -f 3,4,5,6,7,8,9)
  done
  for key in $(find {{.Node}}/foreignKeys -type f); do
    acbuild --debug copy $key /etc/susi/keys/$(echo $key|cut -d\/ -f 3,4,5,6,7,8,9)
  done

	cp nodes.txt .hosts
	echo "127.0.0.1 localhost" >> .hosts
	acbuild --debug copy .hosts /etc/hosts

  acbuild --debug set-exec -- {{.Start}}


  acbuild --debug write --overwrite {{.Node}}/containers/susi-serial-latest-linux-amd64.aci
	if test -f {{.Node}}/containers/susi-serial-latest-linux-amd64.aci.asc; then
		rm {{.Node}}/containers/susi-serial-latest-linux-amd64.aci.asc
	fi
  acbuild --debug end
	`

	template := template.Must(template.New("").Parse(templateString))
	buff := bytes.Buffer{}
	type templateData struct {
		Node  string
		Start string
	}
	template.Execute(&buff, templateData{node, p.StartCommand()})

	execBuildScript(buff.String())
	signContainer(node+"/containers/susi-serial-latest-linux-amd64.aci", gpgpass)
}
