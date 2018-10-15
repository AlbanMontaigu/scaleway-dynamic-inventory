package main

//
// Dependencies
//
import (
    "encoding/json"
    "fmt"
    "os"
    "strings"
    "strconv"
    "github.com/scaleway/go-scaleway"
    "github.com/scaleway/go-scaleway/logger"
    "github.com/scaleway/go-scaleway/types"
)

//
// Global variables
//
var scwApi api.ScalewayAPI

//
// Main function
//
func main() {

    // Init API
    initScwApi()

    // Vars
    var jsonResponse []byte
    var err error

    // Process depending the flags
    switch os.Args[1] {
    
    // Get server list
    default:
    case "--list":
        jsonResponse, err = json.Marshal(getServers())
    
    // Get server details
    case "--host":
        if os.Args[2] == "" {
            panic("Hostname is required (--host <hostname>)")
        }
        jsonResponse, err = json.Marshal(getServer(os.Args[2]))
    }

        
    // Cherck result and displays it if any
    if err != nil {
        panic("Failed to marshal the dynamic inventory")
    }
    fmt.Println(string(jsonResponse))

}

//
// Initialize the common scaleway API object
//
func initScwApi () {

    // Get and control scaleway tokens
    scwOrga := strings.TrimSpace(os.Getenv("SCALEWAY_ORGANIZATION"))
    if strings.TrimSpace(scwOrga) == ""  {
        panic("Required SCALEWAY_ORGANIZATION env var is not set")
    }
    scwToken := strings.TrimSpace(os.Getenv("SCALEWAY_TOKEN"))
    if strings.TrimSpace(scwToken) == "" {
        panic("Required SCALEWAY_TOKEN env var is not set")
    }

    // Init api object
    disabledLoggerFunc := func(a *api.ScalewayAPI) {
        a.Logger = logger.NewDisableLogger()
    }
    api, err := api.NewScalewayAPI(scwOrga, scwToken, "Scaleway Dynamic Inventory", "", disabledLoggerFunc)
    if err != nil {
        panic(fmt.Sprintf("Failed to create scaleway API instance: %s", err))
    }
    scwApi = *api
}

//
// Get servers list (--list flag)
//
func getServers() map[string][]string {

    // API call
    servers, err := scwApi.GetServers(true, 0)
    if err != nil {
        panic("Failed to get servers")
    }

    // Prepare result
    result := make(map[string][]string)

    // Build result
    for _, server := range *servers {
        for _, tag := range server.Tags {
            if _, ok := result[tag]; !ok {
                result[tag] = make([]string, 0)
            }
            result[tag] = append(result[tag], server.Name)
        }
    }
    return result
}

//
// Get server by name (scw whants id)
//
func getScWServerByName(serverName string) *types.ScalewayServer {

    // API call
    serverId, err := scwApi.GetServerID(serverName)
    if err != nil {
        panic(fmt.Sprintf("Failed to get server id with name: %s", serverName))
    }
    server, err := scwApi.GetServer(serverId)
    if err != nil {
        panic(fmt.Sprintf("Failed to get server with id: %s", serverId))
    }
    return server
}

//
// Get server details (with --host flag)
//
func getServer(serverName string) map[string]string {

    // Prepare targeted server
    var server *types.ScalewayServer

    // Prepare result
    result := make(map[string]string)

    // Build generic stuff
    result["ansible_python_interpreter"] = "/usr/bin/python3"
    result["ansible_user"] = "root"

    // Get proxy0 public ip for gateway
    serverProxy0 := getScWServerByName("proxy0")

    // Build specific result for proxy0
    if serverName == "proxy0" {
        server = serverProxy0
        result["proxy_inet"] = "True"
    } else {
        server = getScWServerByName(serverName)
        result["ansible_ssh_common_args"] = "-o ProxyCommand=\"ssh -W %h:%p -q root@" + serverProxy0.PublicAddress.IP + " -i ~/.ssh/scaleway.pem\""
    }

    // Build ansible hosts and takes care about public / private ip
    if server.PublicAddress.IP != "" {
        result["ansible_host"] = server.PublicAddress.IP
    } else {
        result["ansible_host"] = server.PrivateIP
    }

    // Build the vpn_ip
    digitPos := len(server.Name)-1
    if digitPos > 1 {
        lastDigit := string(server.Name[digitPos:])
        if _, err := strconv.Atoi(lastDigit); err == nil {
            switch {
                case strings.Contains(server.Name, "proxy"):
                    result["vpn_ip"] = "192.168.66.1" + lastDigit
                    break
                case strings.Contains(server.Name, "master"):
                    result["vpn_ip"] = "192.168.66.2" + lastDigit
                    break
                case strings.Contains(server.Name, "worker"):
                    result["vpn_ip"] = "192.168.66.3" + lastDigit
                    break
            }
        }
    }
    return result

}
