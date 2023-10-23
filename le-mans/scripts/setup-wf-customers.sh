#!/usr/bin/env bash

/wf-cli/wfcli.sh auth -create -identifier dev+master@wavefront.com -customer master -credentials $(cat /tmp/wf_password) -group browse -grant &&
/wf-cli/wfcli.sh auth -identifier dev+master@wavefront.com -customer master -group user_management -grant &&
/wf-cli/wfcli.sh auth -create -identifier localdev@wavefront.com -customer csp-customer-2 -credentials $(cat /tmp/wf_password) -group browse -grant &&
/wf-cli/wfcli.sh auth -identifier localdev@wavefront.com -customer csp-customer-2 -group user_management -grant &&
/wf-cli/wfcli.sh csp -customer csp-customer-2 -setCspData -cspOrgId eeb0c55e-c84c-4267-8ee2-d6aac76a6180 -cspServInstId d3e5bacf-346a-486b-ac21-74736b42b018
