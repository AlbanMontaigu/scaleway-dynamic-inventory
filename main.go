package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
  "strconv"

	"github.com/scaleway/scaleway-cli/pkg/api"
)

func main() {
	orgToken := os.Getenv("SCALEWAY_ORGANIZATION")
	token := os.Getenv("SCALEWAY_TOKEN")

	if strings.TrimSpace(orgToken) == "" || strings.TrimSpace(token) == "" {
		panic("required environmental variables are not set")
	}


  if os.Args[1] == "--list" {
	  dynamicInventory := make(map[string][]string)
	  getServers(dynamicInventory, token, orgToken)

	  body, err := json.Marshal(dynamicInventory)
	  if err != nil {
		  panic("failed to marshal the dynamic inventory")
	  }

	  fmt.Println(string(body))
  } else if os.Args[1] == "--host" {
	  dynamicInventory := make(map[string]string)
	  getServersDetails(dynamicInventory, token, orgToken, os.Args[2])

    jsonString, err := json.Marshal(dynamicInventory)
	  if err != nil {
		  panic("failed to marshal the dynamic inventory")
	  }

    fmt.Println(string(jsonString))
  }
}

func getServers(dict map[string][]string, token, orgToken string) {
	disabledLoggerFunc := func(a *api.ScalewayAPI) {
		a.Logger = api.NewDisableLogger()
	}

	api, err := api.NewScalewayAPI(orgToken, token, "Scaleway Dynamic Inventory",
		"", disabledLoggerFunc)
	if err != nil {
		panic(fmt.Sprintf("failed to create API instance: %s", err))
	}

	servers, err := api.GetServers(true, 0)
	if err != nil {
		panic(fmt.Sprintf("failed to get servers: %s", err))
	}

	for _, server := range *servers {
		for _, tag := range server.Tags {
			if _, ok := dict[tag]; !ok {
				dict[tag] = make([]string, 0)
			}
      dict[tag] = append(dict[tag], server.Name)
		}
	}
}

func getServersDetails(dict map[string]string, token, orgToken string, servername string) {
	disabledLoggerFunc := func(a *api.ScalewayAPI) {
		a.Logger = api.NewDisableLogger()
	}

	api, err := api.NewScalewayAPI(orgToken, token, "Scaleway Dynamic Inventory",
		"", disabledLoggerFunc)
	if err != nil {
		panic(fmt.Sprintf("failed to create API instance: %s", err))
	}

	servers, err := api.GetServers(true, 0)
	if err != nil {
		panic(fmt.Sprintf("failed to get servers: %s", err))
	}
  
  var count int = 0
  var publicip string = "none"
	for _, server := range *servers {
    count++ 
    if server.Name == "proxy0" {
      publicip = server.PublicAddress.IP
    }
    if server.Name == servername {
      if server.PublicAddress.IP != "" {
        dict["ansible_host"] = server.PublicAddress.IP
      } else {
        dict["ansible_host"] = server.PrivateIP
      }
      dict["ansible_user"] = "root"
      var vpn_ip string = "192.168.66." + strconv.Itoa(count)
      dict["vpn_ip"] = vpn_ip
		}
    if ! strings.Contains(servername, "proxy0") {
      dict["ansible_ssh_common_args"] = "-o ProxyCommand=\"ssh -W %h:%p -q root@" + publicip + " -i ~/.ssh/scaleway.pem\""
    }
	}
}
