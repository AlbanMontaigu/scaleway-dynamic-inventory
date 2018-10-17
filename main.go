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
    "regexp"
    "log"
    "github.com/scaleway/go-scaleway"
    "github.com/scaleway/go-scaleway/logger"
    "github.com/scaleway/go-scaleway/types"
)

//
// Global variables
//
var (
    scwApi api.ScalewayAPI
    l = log.New(os.Stderr, "", 0)
    allowedServerName = regexp.MustCompile("proxy[0-9]|master[[0-9]|worker+[0-9]")
)

//
// Global constants
//
const (
    MSG_PREFIX = "scw-inv:"
)

//
// Main function
//
func main() {

    // Init API
    initScwApi()

    // Vars
    var jsonResponse []byte
    var err error

    // Handling default value
    osArg1 := "--list"
    if len(os.Args) > 1 {
        osArg1 = os.Args[1]
    }

    // Process depending the flags
    switch osArg1 {
    
    // Get server list
    case "--list":
        jsonResponse, err = json.Marshal(getServers())
            
    // Get server details
    case "--host":
        if len(os.Args) < 2  {
            l.Printf("%s hostname is required (--host <hostname>)", MSG_PREFIX)
            os.Exit(1)
        }
        jsonResponse, err = json.Marshal(getServer(os.Args[2]))
        
    // No arg so do nothing exit directly
    default:
        l.Printf("%s usage: [--host|--list]", MSG_PREFIX)
        os.Exit(1)
    }

    // Cherck result and displays it if any
    if err != nil {
        l.Printf("%s failed to marshal the dynamic inventory: %s", MSG_PREFIX, err)
        os.Exit(1)
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
        l.Printf("%s required SCALEWAY_ORGANIZATION env var is not set", MSG_PREFIX)
        os.Exit(1)
    }
    scwToken := strings.TrimSpace(os.Getenv("SCALEWAY_TOKEN"))
    if strings.TrimSpace(scwToken) == "" {
        l.Printf("%s required SCALEWAY_TOKEN env var is not set", MSG_PREFIX)
        os.Exit(1)
    }

    // Init api object
    disabledLoggerFunc := func(a *api.ScalewayAPI) {
        a.Logger = logger.NewDisableLogger()
    }
    api, err := api.NewScalewayAPI(scwOrga, scwToken, "Scaleway Dynamic Inventory", "", disabledLoggerFunc)
    if err != nil {
        l.Printf("%s failed to create scaleway API instance: %s", MSG_PREFIX, err)
        os.Exit(1)
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
        l.Printf("%s failed to get servers: %s", MSG_PREFIX, err)
        os.Exit(1)
    }

    // Prepare result
    result := make(map[string][]string)

    // Build result
    for _, server := range *servers {

        // Servers to filter / skip in the loop
        if (!allowedServerName.MatchString(server.Name)) {
            continue
        }

        // Servers not filtered, adding it to the result
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
        l.Printf("%s failed to get server id with name: %s", MSG_PREFIX, err)
        os.Exit(1)
    }
    server, err := scwApi.GetServer(serverId)
    if err != nil {
        l.Printf("%s failed to get server with id: %s", MSG_PREFIX, err)
        os.Exit(1)
    }
    return server
}

//
// Get server details (with --host flag)
//
func getServer(serverName string) map[string]string {

    // Servers to filter / skip in the loop
    if (!allowedServerName.MatchString(serverName)) {
        l.Printf("%s server name allowed regexp: %s", MSG_PREFIX, allowedServerName.String())
        os.Exit(1)
    }

    // Prepare targeted server
    var server *types.ScalewayServer

    // Prepare result
    result := make(map[string]string)

    // Build generic stuff
    result["ansible_python_interpreter"] = "/usr/bin/python3"
    result["ansible_user"] = "root"

    // Get proxy0 public ip for gateway
    serverProxy0 := getScWServerByName("proxy0")

    // Common proxy conf
    result["ansible_ssh_common_args"] = "-q -C -o ControlMaster=auto -o ControlPersist=5m -o ForwardAgent=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"

    // Build specific result for proxy0
    if serverName == "proxy0" {
        server = serverProxy0
        result["proxy_inet"] = "True"
    } else {
        server = getScWServerByName(serverName)
        result["ansible_ssh_common_args"] = result["ansible_ssh_common_args"] + " -o ProxyCommand=\"ssh -W %h:%p -q root@" + serverProxy0.PublicAddress.IP + " -i ~/.ssh/scaleway.pem\""
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
                    
                case strings.Contains(server.Name, "master"):
                    result["vpn_ip"] = "192.168.66.2" + lastDigit
                    
                case strings.Contains(server.Name, "worker"):
                    result["vpn_ip"] = "192.168.66.3" + lastDigit
                    
            }
        }
    }
    return result

}
