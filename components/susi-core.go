package components

import (
	"bytes"
	"html/template"
)

type susiCoreComponent struct{}

func (p *susiCoreComponent) Config() string {
	return ""
}

func (p *susiCoreComponent) StartCommand() string {
	return "/usr/local/bin/susi-core -k /etc/susi/keys/susi-core.key -c /etc/susi/keys/susi-core.crt"
}

func (p *susiCoreComponent) ExtraShell(node string) string {
	return ""
}

func (p *susiCoreComponent) BuildContainer(node, gpgpass string) {
	buildBaseContainer()
	templateString := `
	acbuild --debug begin /var/lib/susi-dev/containers/susi-base-latest-linux-amd64.aci

  acbuild --debug set-name susi.io/susi-core

  acbuild --debug copy .build/alpine/bin/susi-core /usr/local/bin/susi-core
  acbuild --debug copy {{.Node}}/pki/pki/issued/susi-core.crt /etc/susi/keys/susi-core.crt
  acbuild --debug copy {{.Node}}/pki/pki/private/susi-core.key /etc/susi/keys/susi-core.key
  acbuild --debug copy {{.Node}}/configs/susi-core.json /etc/susi/susi-core.json || true
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


  acbuild --debug write --overwrite {{.Node}}/containers/susi-core-latest-linux-amd64.aci
	if test -f {{.Node}}/containers/susi-core-latest-linux-amd64.aci.asc; then
		rm {{.Node}}/containers/susi-core-latest-linux-amd64.aci.asc
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
	signContainer(node+"/containers/susi-core-latest-linux-amd64.aci", gpgpass)
}
