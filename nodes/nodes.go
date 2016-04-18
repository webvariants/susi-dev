package nodes

import (
	"fmt"
	"io/ioutil"
	"strings"
)

//Node is a node of multiple services
type Node struct {
	ID        string
	IP        string
	Fqdn      string
	PodID     string
	SystemdID string
}

//Nodes is a list of nodes
type Nodes map[string]Node

//Load loads nodes from file
func Load(file string) (Nodes, error) {
	nodes := Nodes{}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nodes, err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if len(line) > 0 {
			parts := strings.Split(line, " ")
			if len(parts) == 3 {
				node := Node{parts[1], parts[0], parts[2], "", ""}
				nodes[node.ID] = node
			} else if len(parts) == 4 {
				node := Node{parts[1], parts[0], parts[2], parts[3], ""}
				nodes[node.ID] = node
			} else if len(parts) == 5 {
				node := Node{parts[1], parts[0], parts[2], parts[3], parts[4]}
				nodes[node.ID] = node
			}
		}
	}
	return nodes, nil
}

//Save saves the nodes to file
func (nodes Nodes) Save(file string) error {
	data := ""
	for _, node := range nodes {
		data += fmt.Sprintf("%v %v %v %v %v\n", node.IP, node.ID, node.Fqdn, node.PodID, node.SystemdID)
	}
	return ioutil.WriteFile(file, []byte(data), 0644)
}

//Set adds a node
func (nodes Nodes) Set(node Node) {
	nodes[node.ID] = node
}
