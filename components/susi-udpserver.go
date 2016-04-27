package components

import (
	"bytes"
	"html/template"
)

type susiUDPServerComponent struct{}

func (p *susiUDPServerComponent) Config() string {
	return `{
    "susi-addr": "localhost",
    "susi-port": 4000,
    "cert": "/etc/susi/keys/susi-udpserver.crt",
    "key": "/etc/susi/keys/susi-udpserver.key",
    "component": {
      "port": 4001
	  }
  }`
}

func (p *susiUDPServerComponent) StartCommand() string {
	return "/usr/local/bin/susi-udpserver -c /etc/susi/susi-udpserver.json"
}

func (p *susiUDPServerComponent) ExtraShell(node string) string {
	return ""
}

func (p *susiUDPServerComponent) BuildContainer(node, gpgpass string) {
	buildBaseContainer()
	templateString := `
	acbuild --debug begin /var/lib/susi-dev/containers/susi-base-latest-linux-amd64.aci

  acbuild --debug set-name susi.io/susi-udpserver

  acbuild --debug copy .build/alpine/bin/susi-udpserver /usr/local/bin/susi-udpserver
  acbuild --debug copy {{.Node}}/pki/pki/issued/susi-udpserver.crt /etc/susi/keys/susi-udpserver.crt
  acbuild --debug copy {{.Node}}/pki/pki/private/susi-udpserver.key /etc/susi/keys/susi-udpserver.key
  acbuild --debug copy {{.Node}}/configs/susi-udpserver.json /etc/susi/susi-udpserver.json || true
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


  acbuild --debug write --overwrite {{.Node}}/containers/susi-udpserver-latest-linux-amd64.aci
	if test -f {{.Node}}/containers/susi-udpserver-latest-linux-amd64.aci.asc; then
		rm {{.Node}}/containers/susi-udpserver-latest-linux-amd64.aci.asc
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
	signContainer(node+"/containers/susi-udpserver-latest-linux-amd64.aci", gpgpass)
}
