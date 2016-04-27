package components

import (
	"bytes"
	"html/template"
)

type susiWebstackComponent struct{}

func (p *susiWebstackComponent) Config() string {
	return ""
}

func (p *susiWebstackComponent) StartCommand() string {
	return "/usr/local/bin/susi-gowebstack -susiaddr 127.0.0.1:4000 -assets /usr/share/susi/webroot/ -cert /etc/susi/keys/susi-gowebstack.crt -key /etc/susi/keys/susi-gowebstack.key -webaddr=:80"
}

func (p *susiWebstackComponent) ExtraShell(node string) string {
	return ""
}

func (p *susiWebstackComponent) BuildContainer(node, gpgpass string) {
	buildBaseContainer()
	templateString := `
	acbuild --debug begin /var/lib/susi-dev/containers/susi-base-latest-linux-amd64.aci

  acbuild --debug set-name susi.io/susi-gowebstack

  acbuild --debug copy .build/alpine/bin/susi-gowebstack /usr/local/bin/susi-gowebstack
  acbuild --debug copy {{.Node}}/pki/pki/issued/susi-gowebstack.crt /etc/susi/keys/susi-gowebstack.crt
  acbuild --debug copy {{.Node}}/pki/pki/private/susi-gowebstack.key /etc/susi/keys/susi-gowebstack.key
  acbuild --debug copy {{.Node}}/configs/susi-gowebstack.json /etc/susi/susi-gowebstack.json || true
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


  acbuild --debug write --overwrite {{.Node}}/containers/susi-gowebstack-latest-linux-amd64.aci
	if test -f {{.Node}}/containers/susi-gowebstack-latest-linux-amd64.aci.asc; then
		rm {{.Node}}/containers/susi-gowebstack-latest-linux-amd64.aci.asc
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
	signContainer(node+"/containers/susi-gowebstack-latest-linux-amd64.aci", gpgpass)
}
