package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/terraform"
)

const routeTableName = `internet`

func main() {
	err := munge(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

func munge(in io.Reader, out io.Writer) error {
	state := &terraform.State{}

	err := json.NewDecoder(in).Decode(state)
	if err != nil {
		return err
	}

	root := state.RootModule()
	if !needsMunging(root) {
		_, err = io.Copy(out, in)
		return err
	}

	newResources := make(map[string]*terraform.ResourceState)
	for key, resource := range root.Resources {
		if !strings.Contains(key, "aws_route_table."+routeTableName) {
			continue
		}
		routeName, route := extractRouteResource(key, resource)
		newResources[routeName] = route
	}
	for key, resource := range newResources {
		root.Resources[key] = resource
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "    ")
	err = encoder.Encode(state)
	return err
}

func needsMunging(root *terraform.ModuleState) bool {
	hasRouteTable := false
	for key, _ := range root.Resources {
		if strings.Contains(key, "aws_route."+routeTableName) {
			return false
		}
		if strings.Contains(key, "aws_route_table."+routeTableName) {
			hasRouteTable = true
		}
	}
	return hasRouteTable
}

func extractRouteResource(name string, table *terraform.ResourceState) (string, *terraform.ResourceState) {

	if table.Primary.Attributes["route.#"] != "1" {
		panic(fmt.Sprintf("Route table %s has %s routes", name, table.Primary.Attributes["route.#"]))
	}

	var cidrBlock, natGatewayID string
	keysToRemove := make([]string, 0)
	for k, v := range table.Primary.Attributes {
		if strings.Contains(k, "cidr_block") {
			cidrBlock = v
		}
		if strings.Contains(k, "nat_gateway_id") {
			natGatewayID = v
		}
		if strings.Contains(k, "route") {
			keysToRemove = append(keysToRemove, k)
		}
	}
	for _, k := range keysToRemove {
		delete(table.Primary.Attributes, k)
	}
	table.Primary.Attributes["route.#"] = "0"
	table.Dependencies = []string{}

	if cidrBlock == "" || natGatewayID == "" {
		panic("Faild to extract cidr_block and nat_gateway_id from" + name)
	}

	id := routeIDHash(table.Primary.ID, cidrBlock)
	route := &terraform.ResourceState{
		Type: "aws_route",
		Dependencies: []string{
			"aws_nat_gateway.cf",
			"aws_route_table." + routeTableName,
		},
		Primary: &terraform.InstanceState{
			ID: id,
			Attributes: map[string]string{
				"destination_cidr_block":     cidrBlock,
				"destination_prefix_list_id": "",
				"gateway_id":                 "",
				"id":                         id,
				"instance_id":                "",
				"instance_owner_id":          "",
				"nat_gateway_id":             natGatewayID,
				"network_interface_id":       "",
				"origin":                     "CreateRoute",
				"route_table_id":             table.Primary.ID,
				"state":                      "active",
				"vpc_peering_connection_id":  "",
			},
			Meta: map[string]string{},
		},
		Deposed: []*terraform.InstanceState{},
	}

	nameElements := strings.Split(name, ".")
	resourceIndex := nameElements[len(nameElements)-1]
	return "aws_route." + routeTableName + "." + resourceIndex, route
}

// Create an ID for a route
func routeIDHash(routeTableID, CIDRBlock string) string {
	return fmt.Sprintf("r-%s%d", routeTableID, hashcode.String(CIDRBlock))
}
