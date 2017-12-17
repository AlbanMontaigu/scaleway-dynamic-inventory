# Scaleway dynamic inventory for Ansible

Designed to make using Ansible with Scaleway easy, this small script will
retrieve all servers associated with your account and group them by their
tags. This makes it extremely easy to associate your Ansible playbooks with
individual groups of servers (i.e. updating OpenVPN on only VPN enabled
servers).

## Installation

* Get the application: `go get github.com/chmod666org/scaleway-dynamic-inventory`
* Go install this application: `go install github.com/chmod666org/scaleway-dynamic-inventory`
* Create a new bash script and add the following to it:
```
#!/bin/bash

SCALEWAY_ORG_TOKEN='' SCALEWAY_TOKEN='' scaleway-dynamic-inventory
```
* Call your Ansible playbook, passing in the bash script created in step 2 to the -i parameter: `ansible-playbook play.yml -i scaleway-dynamic-inventory.sh`
