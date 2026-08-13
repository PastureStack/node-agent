package handlers

import (
	"fmt"
	"github.com/PastureStack/node-agent/core/progress"
	"github.com/PastureStack/node-agent/utilities/config"
	"github.com/PastureStack/node-agent/utilities/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
	"os"
	"os/exec"
)

type ConfigUpdateHandler struct {
}

func (h *ConfigUpdateHandler) ConfigUpdate(event *revents.Event, cli *client.RancherClient) error {
	if event.Name != "config.update" || event.ReplyTo == "" {
		return nil
	}

	if len(utils.InterfaceToArray(event.Data["items"])) == 0 {
		return reply(map[string]interface{}{}, event, cli)
	}
	itemNames := []string{}

	for _, v := range utils.InterfaceToArray(event.Data["items"]) {
		item := utils.InterfaceToMap(v)
		name, ok := approvedConfigItem(utils.InterfaceToString(item["name"]))
		if !ok {
			return fmt.Errorf("unsupported configuration item")
		}
		if name != "pyagent" || config.UpdatePyagent() {
			itemNames = append(itemNames, name)
		}
	}
	home := config.Home()
	env := os.Environ()
	env = append(env, fmt.Sprintf("%v=%v", "CATTLE_ACCESS_KEY", config.AccessKey()))
	env = append(env, fmt.Sprintf("%v=%v", "CATTLE_SECRET_KEY", config.SecretKey()))
	env = append(env, fmt.Sprintf("%v=%v", "CATTLE_HOME", home))
	args := itemNames

	retcode := 0

	command := exec.Command(config.Sh(), args...)
	command.Env = env
	command.Dir = home
	output, err := command.CombinedOutput()
	if err != nil {
		retcode = utils.GetExitCode(err)
	}
	if retcode != 0 {
		pro := &progress.Progress{Request: event, Client: cli}
		pro.Update("config update failed", "no", map[string]interface{}{
			"exitCode": retcode,
			"output":   string(output),
		})
		return nil
	}
	return reply(map[string]interface{}{}, event, cli)
}

// approvedConfigItem maps every supported first-party item to a string
// literal.  The configuration script treats arguments beginning with "--"
// as options, so forwarding arbitrary control-plane input would let an event
// change the script's behavior even though exec.Command does not invoke a
// shell.
func approvedConfigItem(name string) (string, bool) {
	switch name {
	case "agent-instance-scripts":
		return "agent-instance-scripts", true
	case "agent-instance-startup":
		return "agent-instance-startup", true
	case "bootstrap":
		return "bootstrap", true
	case "configscripts":
		return "configscripts", true
	case "dnsmasq":
		return "dnsmasq", true
	case "haproxy":
		return "haproxy", true
	case "host-api":
		return "host-api", true
	case "host-config":
		return "host-config", true
	case "host-iptables":
		return "host-iptables", true
	case "host-routes":
		return "host-routes", true
	case "hosts":
		return "hosts", true
	case "ipsec":
		return "ipsec", true
	case "ipsec-hosts":
		return "ipsec-hosts", true
	case "iptables":
		return "iptables", true
	case "metadata":
		return "metadata", true
	case "metadata-answers":
		return "metadata-answers", true
	case "monit":
		return "monit", true
	case "psk":
		return "psk", true
	case "pyagent":
		return "pyagent", true
	case "python-agent":
		return "python-agent", true
	case "services":
		return "services", true
	case "system-stacks":
		return "system-stacks", true
	default:
		return "", false
	}
}
