package components

import (
	"bytes"
	"html/template"
	"os"
)

type susiMQTTComponent struct{}

func (p *susiMQTTComponent) Config() string {
	return `{
    "susi-addr": "localhost",
    "susi-port": 4000,
    "cert": "/etc/susi/keys/susi-mqtt.crt",
    "key": "/etc/susi/keys/susi-mqtt.key",
    "component": {
	      "mqtt-addr": "localhost",
	      "mqtt-port": 1883,
	      "forward": [".*@mqtt"],
	      "subscribe": ["susi/#"]
	  }
  }`
}

func (p *susiMQTTComponent) StartCommand() string {
	return "/usr/local/bin/susi-mqtt -c /etc/susi/susi-mqtt.json"
}

func (p *susiMQTTComponent) buildBaseContainer() {
	buildBaseContainer()
	script := `
	  acbuild --debug begin .containers/susi-base-latest-linux-amd64.aci
	  acbuild --debug set-name susi.io/susi-mqtt-base
    acbuild --debug run -- /bin/sh -c "echo -en 'http://dl-4.alpinelinux.org/alpine/v3.3/main\n' > /etc/apk/repositories"
	  acbuild --debug run -- apk update
	  acbuild --debug run -- apk add mosquitto-libs mosquitto-libs++


	  acbuild --debug write --overwrite .containers/susi-mqtt-base-latest-linux-amd64.aci
	  acbuild --debug end
	`
	if _, err := os.Stat(".containers/susi-mqtt-base-latest-linux-amd64.aci"); err != nil {
		execBuildScript(script)
	}
}

func (p *susiMQTTComponent) BuildContainer(node, gpgpass string) {
	p.buildBaseContainer()
	templateString := `

	acbuild --debug begin .containers/susi-mqtt-base-latest-linux-amd64.aci

  acbuild --debug set-name susi.io/susi-mqtt

  acbuild --debug copy .build/alpine/bin/susi-mqtt /usr/local/bin/susi-mqtt
  acbuild --debug copy {{.Node}}/pki/pki/issued/susi-mqtt.crt /etc/susi/keys/susi-mqtt.crt
  acbuild --debug copy {{.Node}}/pki/pki/private/susi-mqtt.key /etc/susi/keys/susi-mqtt.key
  acbuild --debug copy {{.Node}}/configs/susi-mqtt.json /etc/susi/susi-mqtt.json || true
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


  acbuild --debug write --overwrite {{.Node}}/containers/susi-mqtt-latest-linux-amd64.aci
	if test -f {{.Node}}/containers/susi-mqtt-latest-linux-amd64.aci.asc; then
		rm {{.Node}}/containers/susi-mqtt-latest-linux-amd64.aci.asc
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
	signContainer(node+"/containers/susi-mqtt-latest-linux-amd64.aci", gpgpass)
}
