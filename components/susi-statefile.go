package components

import (
	"bytes"
	"html/template"
)

type susiStatefileComponent struct{}

func (p *susiStatefileComponent) Config() string {
	return `{
    "susi-addr": "localhost",
    "susi-port": 4000,
    "cert": "/etc/susi/keys/susi-statefile.crt",
    "key": "/etc/susi/keys/susi-statefile.key",
    "component": {
      "file": "/usr/share/susi/statefile.json"
	  }
  }`
}

func (p *susiStatefileComponent) StartCommand() string {
	return "/usr/local/bin/susi-statefile -c /etc/susi/susi-statefile.json"
}

func (p *susiStatefileComponent) ExtraShell(node string) string {
	return ""
}

func (p *susiStatefileComponent) BuildContainer(node, gpgpass string) {
	buildBaseContainer()
	templateString := `
	acbuild --debug begin .containers/susi-base-latest-linux-amd64.aci

  acbuild --debug set-name susi.io/susi-statefile

  acbuild --debug copy .build/alpine/bin/susi-statefile /usr/local/bin/susi-statefile
  acbuild --debug copy {{.Node}}/pki/pki/issued/susi-statefile.crt /etc/susi/keys/susi-statefile.crt
  acbuild --debug copy {{.Node}}/pki/pki/private/susi-statefile.key /etc/susi/keys/susi-statefile.key
  acbuild --debug copy {{.Node}}/configs/susi-statefile.json /etc/susi/susi-statefile.json || true
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


  acbuild --debug write --overwrite {{.Node}}/containers/susi-statefile-latest-linux-amd64.aci
	if test -f {{.Node}}/containers/susi-statefile-latest-linux-amd64.aci.asc; then
		rm {{.Node}}/containers/susi-statefile-latest-linux-amd64.aci.asc
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
	signContainer(node+"/containers/susi-statefile-latest-linux-amd64.aci", gpgpass)
}
