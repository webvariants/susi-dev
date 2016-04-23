package components

import (
	"bytes"
	"html/template"
)

type susiAuthenticatorComponent struct{}

func (p *susiAuthenticatorComponent) Config() string {
	return `{
    "susi-addr": "localhost",
    "susi-port": 4000,
    "cert": "/etc/susi/keys/susi-authenticator.crt",
    "key": "/etc/susi/keys/susi-authenticator.key",
    "component": {
      "file": "/usr/share/susi/authenticator.json"
    }
  }`
}

func (p *susiAuthenticatorComponent) StartCommand() string {
	return "/usr/local/bin/susi-authenticator -c /etc/susi/susi-authenticator.json"
}

func (p *susiAuthenticatorComponent) BuildContainer(node, gpgpass string) {
	buildBaseContainer()

	templateString := `
	acbuild --debug begin .containers/susi-base-latest-linux-amd64.aci

  acbuild --debug set-name susi.io/susi-authenticator

  acbuild --debug copy .build/alpine/bin/susi-authenticator /usr/local/bin/susi-authenticator
  acbuild --debug copy {{.Node}}/pki/pki/issued/susi-authenticator.crt /etc/susi/keys/susi-authenticator.crt
  acbuild --debug copy {{.Node}}/pki/pki/private/susi-authenticator.key /etc/susi/keys/susi-authenticator.key
  acbuild --debug copy {{.Node}}/configs/susi-authenticator.json /etc/susi/susi-authenticator.json || true
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


  acbuild --debug write --overwrite {{.Node}}/containers/susi-authenticator-latest-linux-amd64.aci
	if test -f {{.Node}}/containers/susi-authenticator-latest-linux-amd64.aci.asc; then
		rm {{.Node}}/containers/susi-authenticator-latest-linux-amd64.aci.asc
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
	signContainer(node+"/containers/susi-authenticator-latest-linux-amd64.aci", gpgpass)
}
