package components

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
)

type susiGoComponent struct{}

func (p *susiGoComponent) Config() string {
	return ""
}

func (p *susiGoComponent) StartCommand() string {
	return "/usr/bin/go run /usr/share/susi/golang-program.go"
}

func (p *susiGoComponent) buildBaseContainer() {
	script := `
	  acbuild --debug begin
	  acbuild --debug set-name susi.io/susi-go-base
		acbuild --debug dep add quay.io/coreos/alpine-sh
		acbuild --debug run -- /bin/sh -c "echo -en 'http://dl-4.alpinelinux.org/alpine/v3.3/main\n@community http://dl-4.alpinelinux.org/alpine/v3.3/community\n' > /etc/apk/repositories"
	  acbuild --debug run -- apk update
	  acbuild --debug run -- apk add go@community git
		acbuild --debug run -- mkdir /root/go
		acbuild --debug environment add GOPATH /root/go
		acbuild --debug run -- go get github.com/webvariants/susigo
	  acbuild --debug write --overwrite /var/lib/susi-dev/containers/susi-go-base-latest-linux-amd64.aci
	  acbuild --debug end
	`
	if _, err := os.Stat("/var/lib/susi-dev/containers/susi-go-base-latest-linux-amd64.aci"); err != nil {
		execBuildScript(script)
	}
}

func (p *susiGoComponent) ExtraShell(node string) string {
	return fmt.Sprintf(`echo -en "package main\n\
\n\
import (\n\
	\"log\"\n\
	\"time\"\n\
	\"fmt\"\n\
	\"github.com/webvariants/susigo\"\n\
)\n\
\n\
func main() {\n\
	susi, err := susigo.NewSusi(\"[::1]:4000\", \"/etc/susi/keys/susi-go.crt\", \"/etc/susi/keys/susi-go.key\")\n\
	if err != nil {\n\
		log.Fatal(err)\n\
	}\n\
\n\
	susi.RegisterProcessor(\"the-answer\", func(event *susigo.Event) {\n\
		event.Payload = 42\n\
		susi.Ack(event)\n\
	})\n\
\n\
	event := susigo.Event{\n\
		Topic: \"the-answer\",\n\
	}\n\
\n\
	time.Sleep(1 * time.Second)
	susi.Publish(event, func(event *susigo.Event) {\n\
		fmt.Println(\"The answer is\", event.Payload)\n\
	})\n\
\n\
	select {}\n\
}\n\
" > %v/assets/golang-program.go
	`, node)
}

func (p *susiGoComponent) BuildContainer(node, gpgpass string) {
	p.buildBaseContainer()
	templateString := `
	acbuild --debug begin /var/lib/susi-dev/containers/susi-go-base-latest-linux-amd64.aci

  acbuild --debug set-name susi.io/susi-go

  acbuild --debug copy {{.Node}}/pki/pki/issued/susi-go.crt /etc/susi/keys/susi-go.crt
  acbuild --debug copy {{.Node}}/pki/pki/private/susi-go.key /etc/susi/keys/susi-go.key
  acbuild --debug copy {{.Node}}/configs/susi-go.json /etc/susi/susi-go.json || true
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

  acbuild --debug write --overwrite {{.Node}}/containers/susi-go-latest-linux-amd64.aci
	if test -f {{.Node}}/containers/susi-go-latest-linux-amd64.aci.asc; then
		rm {{.Node}}/containers/susi-go-latest-linux-amd64.aci.asc
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
	signContainer(node+"/containers/susi-go-latest-linux-amd64.aci", gpgpass)
}
