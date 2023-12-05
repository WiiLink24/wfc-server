package serverbrowser

import (
	"github.com/logrusorgru/aurora/v3"
	"regexp"
	"wwfc/logging"
	"wwfc/serverbrowser/filter"
)

// DWC makes requests in the following formats:
// Matching ver 03: dwc_mver = %d and dwc_pid != %u and maxplayers = %d and numplayers < %d and dwc_mtype = %d and dwc_hoststate = %u and dwc_suspend = %u and (%s)
// Matching ver 90: dwc_mver = %d and dwc_pid != %u and maxplayers = %d and numplayers < %d and dwc_mtype = %d and dwc_mresv != dwc_pid and (%s)
// ...OR
// Self Lookup: dwc_pid = %u

// Example: dwc_mver = 90 and dwc_pid != 43 and maxplayers = 11 and numplayers < 11 and dwc_mtype = 0 and dwc_hoststate = 2 and dwc_suspend = 0 and (rk = 'vs' and ev >= 4250 and ev <= 5750 and p = 0)

var regexSelfLookup = regexp.MustCompile(`^dwc_pid ?= ?(\d{1,10})$`)

func filterServers(servers []map[string]string, queryGame string, expression string, publicIP string) []map[string]string {
	if match := regexSelfLookup.FindStringSubmatch(expression); match != nil {
		dwc_pid := match[1]

		filtered := []map[string]string{}

		// Search for where the profile ID matches
		for _, server := range servers {
			if server["gamename"] != queryGame {
				continue
			}

			if server["dwc_pid"] == dwc_pid {
				if server["publicip"] != publicIP {
					logging.Error(ModuleName, "Self lookup", aurora.Cyan(dwc_pid), "from wrong IP")
					return []map[string]string{}
				}

				logging.Info(ModuleName, "Self lookup from", aurora.Cyan(dwc_pid), "ok")
				return []map[string]string{server}
			}

			// Alternatively, if the server hasn't set its dwc_pid field yet, we return servers matching the request's public IP.
			// If multiple servers exist with the same public IP then the client will use the one with the matching port.
			// This is a bit of a hack to speed up server creation.
			if _, ok := server["dwc_pid"]; !ok && server["publicip"] == publicIP {
				// Create a copy of the map with some values changed
				newServer := map[string]string{}
				for k, v := range server {
					newServer[k] = v
				}
				newServer["dwc_pid"] = dwc_pid
				newServer["dwc_mtype"] = "0"
				newServer["dwc_mver"] = "0"
				filtered = append(filtered, newServer)
			}
		}

		if len(filtered) == 0 {
			logging.Error(ModuleName, "Could not find server with dwc_pid", aurora.Cyan(dwc_pid))
			return []map[string]string{}
		}

		logging.Info(ModuleName, "Self lookup for", aurora.Cyan(dwc_pid), "matched", aurora.BrightCyan(len(filtered)), "servers via public IP")
		return filtered
	}

	// Matchmaking search
	tree, err := filter.Parse(expression)
	if err != nil {
		logging.Error(ModuleName, "Error parsing filter:", err.Error())
		return []map[string]string{}
	}

	filtered := []map[string]string{}

	for _, server := range servers {
		if server["gamename"] != queryGame {
			continue
		}

		if server["dwc_hoststate"] != "0" && server["dwc_hoststate"] != "2" {
			continue
		}

		ret, err := filter.Eval(tree, server, queryGame)
		if err != nil {
			logging.Error(ModuleName, "Error evaluating filter:", err.Error())
			return []map[string]string{}
		}

		if ret != 0 {
			filtered = append(filtered, server)
		}
	}

	logging.Info(ModuleName, "Matched", aurora.BrightCyan(len(filtered)), "servers")
	return filtered
}
