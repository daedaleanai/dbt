package netrc

import (
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/daedaleanai/dbt/v2/log"
)

type BasicAuth struct {
	User     string
	Password string
}

type netrc struct {
	machines map[string]BasicAuth
}

var usersNetrcFile netrc = netrc{
	machines: make(map[string]BasicAuth),
}

func init() {
	parseNetrc()
}

func parseNetrc() {
	var homeEnvVar string
	for _, v := range os.Environ() {
		parts := strings.Split(v, "=")
		if len(parts) == 2 {
			key := parts[0]
			if key == "HOME" {
				homeEnvVar = parts[1]
			}
		}
	}
	if homeEnvVar == "" {
		log.Warning("Unable to find HOME environment variable. netrc not parsed.\n")
		return
	}

	netrcPath := path.Join(homeEnvVar, ".netrc")
	netrcContents, err := os.ReadFile(netrcPath)
	if err != nil {
		log.Warning("Error reading %q.\n", netrcPath)
		return
	}

	netrcLines := strings.Split(string(netrcContents), "\n")
	currentMachine := ""

	saveUsername := func(machine, login string) {
		if machine != "" {
			if m, ok := usersNetrcFile.machines[machine]; ok {
				usersNetrcFile.machines[machine] = BasicAuth{User: login, Password: m.Password}
			} else {
				usersNetrcFile.machines[machine] = BasicAuth{User: login}
			}
		}
	}

	savePassword := func(machine, passwd string) {
		if machine != "" {
			if m, ok := usersNetrcFile.machines[machine]; ok {
				usersNetrcFile.machines[machine] = BasicAuth{User: m.User, Password: passwd}
			} else {
				usersNetrcFile.machines[machine] = BasicAuth{Password: passwd}
			}
		}
	}

	for _, line := range netrcLines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "machine") {
			currentMachine = strings.TrimSpace(strings.TrimPrefix(line, "machine"))
		} else if strings.HasPrefix(line, "login") {
			saveUsername(currentMachine, strings.TrimSpace(strings.TrimPrefix(line, "login")))
		} else if strings.HasPrefix(line, "password") {
			savePassword(currentMachine, strings.TrimSpace(strings.TrimPrefix(line, "password")))
		}

	}
}

func GetAuthForUrl(urlString string) *BasicAuth {
	url, err := url.Parse(urlString)
	if err != nil {
		log.Warning("Invalid URL %q.\n", urlString)
		return nil
	}

	if auth, ok := usersNetrcFile.machines[url.Hostname()]; ok {
		return &auth
	}

	return nil
}
