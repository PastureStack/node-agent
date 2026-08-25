package config

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/PastureStack/node-agent/model"
	"github.com/PastureStack/node-agent/utilities/constants"
	googleUUID "github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rancher/log"
)

const maxIdentityFileSize = 4096

func URL() string {
	ret := DefaultValue("CONFIG_URL", "")
	if len(ret) == 0 {
		return APIURL("")
	}
	return ret
}

func stripSchemas(url string) string {
	if len(url) == 0 {
		return ""
	}

	if strings.HasSuffix(url, "/schemas") {
		return url[0 : len(url)-len("schemas")]
	}

	return url
}

func APIURL(df string) string {
	return stripSchemas(DefaultValue("URL", df))
}

func APIProxyListenPort() int {
	ret, _ := strconv.Atoi(DefaultValue("API_PROXY_LISTEN_PORT", "9342"))
	return ret
}

func Builds() string {
	switch filepath.Clean(strings.TrimSpace(DefaultValue("BUILD_DIR", ""))) {
	case "/var/lib/pasturestack/builds":
		return "/var/lib/pasturestack/builds"
	case "/var/lib/cattle/builds":
		return "/var/lib/cattle/builds"
	case "/var/lib/rancher/builds":
		return "/var/lib/rancher/builds"
	case `c:\ProgramData\pasturestack\builds`:
		return `c:\ProgramData\pasturestack\builds`
	case `c:\ProgramData\cattle\builds`:
		return `c:\ProgramData\cattle\builds`
	default:
		return defaultBuildDir(Home())
	}
}

func StateDir() string {
	return managedStateDir(DefaultValue("STATE_DIR", Home()))
}

func KeyFile() string {
	switch filepath.Clean(strings.TrimSpace(DefaultValue("HOST_KEY_FILE", ""))) {
	case "/var/lib/pasturestack/etc/ssl/host.key":
		return "/var/lib/pasturestack/etc/ssl/host.key"
	case "/var/lib/cattle/etc/ssl/host.key":
		return "/var/lib/cattle/etc/ssl/host.key"
	case "/var/lib/rancher/etc/ssl/host.key":
		return "/var/lib/rancher/etc/ssl/host.key"
	case "/var/lib/etc/ssl/host.key":
		return "/var/lib/etc/ssl/host.key"
	case `c:\ProgramData\pasturestack\etc\ssl\host.key`:
		return `c:\ProgramData\pasturestack\etc\ssl\host.key`
	case `c:\ProgramData\cattle\etc\ssl\host.key`:
		return `c:\ProgramData\cattle\etc\ssl\host.key`
	case `c:\ProgramData\etc\ssl\host.key`:
		return `c:\ProgramData\etc\ssl\host.key`
	}
	preferredPath, legacyPath := hostKeyPaths(StateDir())
	if legacyPath != "" {
		if _, err := os.Stat(preferredPath); os.IsNotExist(err) {
			if _, legacyErr := os.Stat(legacyPath); legacyErr == nil {
				return legacyPath
			}
		}
	}
	return preferredPath
}

func physicalHostUUIDFile() string {
	return managedIdentityFile(
		DefaultValue("PHYSICAL_HOST_UUID_FILE", ".physical_host_uuid"),
		".physical_host_uuid",
	)
}

func PhysicalHostUUID(forceWrite bool) (string, error) {
	return GetUUIDFromFile("PHYSICAL_HOST_UUID", physicalHostUUIDFile(), forceWrite)
}

func Home() string {
	if runtime.GOOS == "windows" {
		configured := filepath.Clean(strings.TrimSpace(DefaultValue("HOME", `c:\ProgramData\pasturestack`)))
		for _, candidate := range []string{`c:\ProgramData\pasturestack`, `c:\ProgramData\cattle`} {
			if samePath(configured, candidate) {
				return candidate
			}
		}
		return `c:\ProgramData\pasturestack`
	}
	switch filepath.Clean(strings.TrimSpace(DefaultValue("HOME", "/var/lib/pasturestack"))) {
	case "/var/lib/pasturestack":
		return "/var/lib/pasturestack"
	case "/var/lib/cattle":
		return "/var/lib/cattle"
	case "/var/lib/rancher":
		return "/var/lib/rancher"
	default:
		return "/var/lib/pasturestack"
	}
}

func defaultBuildDir(home string) string {
	switch home {
	case "/var/lib/cattle":
		return "/var/lib/cattle/builds"
	case "/var/lib/rancher":
		return "/var/lib/rancher/builds"
	case `c:\ProgramData\cattle`:
		return `c:\ProgramData\cattle\builds`
	case `c:\ProgramData\pasturestack`:
		return `c:\ProgramData\pasturestack\builds`
	default:
		return "/var/lib/pasturestack/builds"
	}
}

func hostKeyPaths(stateDir string) (string, string) {
	switch stateDir {
	case "/var/lib/rancher/state":
		return "/var/lib/rancher/etc/ssl/host.key", ""
	case "/var/lib/cattle":
		return "/var/lib/cattle/etc/ssl/host.key", "/var/lib/etc/ssl/host.key"
	case `c:\ProgramData\cattle`:
		return `c:\ProgramData\cattle\etc\ssl\host.key`, `c:\ProgramData\etc\ssl\host.key`
	case `c:\ProgramData\pasturestack`:
		return `c:\ProgramData\pasturestack\etc\ssl\host.key`, `c:\ProgramData\etc\ssl\host.key`
	default:
		return "/var/lib/pasturestack/etc/ssl/host.key", "/var/lib/etc/ssl/host.key"
	}
}

func managedStateDir(configured string) string {
	cleaned := filepath.Clean(strings.TrimSpace(configured))
	if runtime.GOOS != "windows" && cleaned == "/var/lib/rancher" {
		return "/var/lib/rancher/state"
	}
	for _, candidate := range approvedStateDirs() {
		if samePath(cleaned, candidate) {
			return candidate
		}
	}
	return defaultStateDir()
}

func defaultStateDir() string {
	if runtime.GOOS == "windows" {
		return `c:\ProgramData\pasturestack`
	}
	return "/var/lib/pasturestack"
}

func approvedStateDirs() []string {
	if runtime.GOOS == "windows" {
		return []string{
			`c:\ProgramData\pasturestack`,
			`c:\ProgramData\cattle`,
		}
	}
	return []string{
		"/var/lib/pasturestack",
		"/var/lib/cattle",
		"/var/lib/rancher/state",
	}
}

func samePath(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func managedIdentityFile(configured, filename string) string {
	if filename != ".physical_host_uuid" && filename != ".docker_uuid" {
		return ""
	}
	cleaned := filepath.Clean(strings.TrimSpace(configured))
	if cleaned == filename || cleaned == "." || cleaned == "" {
		return filepath.Join(StateDir(), filename)
	}
	for _, root := range approvedStateDirs() {
		candidate := filepath.Join(root, filename)
		if samePath(cleaned, candidate) {
			return candidate
		}
	}
	return filepath.Join(StateDir(), filename)
}

func resolveIdentityFile(uuidFilePath string) (string, string, error) {
	cleaned := filepath.Clean(strings.TrimSpace(uuidFilePath))
	for _, root := range approvedStateDirs() {
		if samePath(cleaned, filepath.Join(root, ".physical_host_uuid")) {
			return root, ".physical_host_uuid", nil
		}
		if samePath(cleaned, filepath.Join(root, ".docker_uuid")) {
			return root, ".docker_uuid", nil
		}
	}
	return "", "", errors.New("identity file is outside an approved state directory")
}

func openIdentityFile(uuidFilePath string, flags int) (*os.File, error) {
	root, filename, err := resolveIdentityFile(uuidFilePath)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	directory, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	return directory.OpenFile(filename, flags, 0600)
}

func removeIdentityFile(uuidFilePath string) error {
	root, filename, err := resolveIdentityFile(uuidFilePath)
	if err != nil {
		return err
	}
	directory, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Remove(filename)
}

func getUUIDFromFile(uuidFilePath string) (string, error) {
	uuid := ""

	file, err := openIdentityFile(uuidFilePath, os.O_RDONLY)
	if err == nil {
		fileBuffer, readErr := io.ReadAll(io.LimitReader(file, maxIdentityFileSize+1))
		_ = file.Close()
		if readErr != nil {
			return "", errors.Wrap(readErr, constants.ReadUUIDFromFileError+"failed to read uuid file")
		}
		if len(fileBuffer) > maxIdentityFileSize {
			return "", errors.New(constants.ReadUUIDFromFileError + "uuid file exceeds size limit")
		}
		uuid = strings.TrimSpace(string(fileBuffer))
	} else if !os.IsNotExist(err) {
		return "", errors.Wrap(err, constants.ReadUUIDFromFileError+"failed to read uuid file")
	}
	if uuid == "" {
		newUUID, err := googleUUID.NewRandom()
		if err != nil {
			return "", errors.Wrap(err, constants.ReadUUIDFromFileError+"failed to generate uuid")
		}
		uuid = newUUID.String()
		file, err := openIdentityFile(uuidFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
		if err != nil {
			return "", errors.Wrap(err, constants.ReadUUIDFromFileError+"failed to create uuid file")
		}
		if _, err := file.WriteString(uuid); err != nil {
			_ = file.Close()
			return "", errors.Wrap(err, constants.ReadUUIDFromFileError+"failed to write uuid to file")
		}
		if err := file.Close(); err != nil {
			return "", errors.Wrap(err, constants.ReadUUIDFromFileError+"failed to close uuid file")
		}
	}
	return uuid, nil
}

func GetUUIDFromFile(envName string, uuidFilePath string, forceWrite bool) (string, error) {
	uuid := DefaultValue(envName, "")
	if uuid != "" {
		if forceWrite {
			if err := removeIdentityFile(uuidFilePath); err != nil && !os.IsNotExist(err) {
				return "", errors.Wrap(err, constants.GetUUIDFromFileError+"failed to remove uuid file")
			}
			file, err := openIdentityFile(uuidFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
			if err != nil {
				return "", errors.Wrap(err, constants.GetUUIDFromFileError+"failed to create uuid file")
			}
			if _, err := file.WriteString(uuid); err != nil {
				_ = file.Close()
				return "", errors.Wrap(err, constants.GetUUIDFromFileError+"failed to write uuid to file")
			}
			if err := file.Close(); err != nil {
				return "", errors.Wrap(err, constants.GetUUIDFromFileError+"failed to close uuid file")
			}
		}
		return uuid, nil
	}
	return getUUIDFromFile(uuidFilePath)
}

func DoPing() bool {
	return DefaultValue("PING_ENABLED", "true") == "true"
}

func DetectCloudProvider() bool {
	return DefaultValue("DETECT_CLOUD_PROVIDER", "true") == "true"
}

func SetSecretKey(value string) {
	constants.ConfigOverride["SECRET_KEY"] = value
}

func SecretKey() string {
	return DefaultValue("SECRET_KEY", "adminpass")
}

func SetAccessKey(value string) {
	constants.ConfigOverride["ACCESS_KEY"] = value
}

func AccessKey() string {
	return DefaultValue("ACCESS_KEY", "admin")
}

func SetAPIURL(value string) {
	constants.ConfigOverride["URL"] = value
}

func agentIP() string {
	return DefaultValue("AGENT_IP", "")
}

func Sh() string {
	switch filepath.Clean(strings.TrimSpace(DefaultValue("CONFIG_SCRIPT", ""))) {
	case "/var/lib/pasturestack/config.sh":
		return "/var/lib/pasturestack/config.sh"
	case "/var/lib/cattle/config.sh":
		return "/var/lib/cattle/config.sh"
	case "/var/lib/rancher/config.sh":
		return "/var/lib/rancher/config.sh"
	case `c:\ProgramData\pasturestack\config.sh`:
		return `c:\ProgramData\pasturestack\config.sh`
	case `c:\ProgramData\cattle\config.sh`:
		return `c:\ProgramData\cattle\config.sh`
	default:
		return defaultConfigScript(Home())
	}
}

func defaultConfigScript(home string) string {
	switch home {
	case "/var/lib/cattle":
		return "/var/lib/cattle/config.sh"
	case "/var/lib/rancher":
		return "/var/lib/rancher/config.sh"
	case `c:\ProgramData\cattle`:
		return `c:\ProgramData\cattle\config.sh`
	case `c:\ProgramData\pasturestack`:
		return `c:\ProgramData\pasturestack\config.sh`
	default:
		return "/var/lib/pasturestack/config.sh"
	}
}

func PhysicalHost() (model.PingResource, error) {
	uuid, err := PhysicalHostUUID(false)
	if err != nil {
		return model.PingResource{}, errors.Wrap(err, constants.PhysicalHostError+"failed to get physical host uuid")
	}
	hostname, err := Hostname()
	if err != nil {
		return model.PingResource{}, errors.Wrap(err, constants.PhysicalHostError+"failed to get hostname")
	}
	return model.PingResource{
		UUID: uuid,
		Type: "physicalHost",
		Kind: "physicalHost",
		Name: hostname,
	}, nil
}

func UpdatePyagent() bool {
	return DefaultValue("CONFIG_UPDATE_PYAGENT", "true") == "true"
}

func HostAPIIP() string {
	return DefaultValue("HOST_API_IP", "0.0.0.0")
}

func HostAPIPort() string {
	return DefaultValue("HOST_API_PORT", "9345")
}

func JwtPublicKeyFile() string {
	preferredPath, legacyPath := approvedPublicKeyPaths(Home())
	if configured := strings.TrimSpace(DefaultValue("CONSOLE_HOST_API_PUBLIC_KEY", "")); configured != "" {
		if validated := approvedPublicKeyPath(configured); validated != "" {
			return validated
		}
		return preferredPath
	}
	if _, err := os.Stat(preferredPath); os.IsNotExist(err) {
		if _, legacyErr := os.Stat(legacyPath); legacyErr == nil {
			return legacyPath
		}
	}
	return preferredPath
}

func approvedPublicKeyPaths(home string) (string, string) {
	if runtime.GOOS == "windows" {
		if samePath(home, `c:\ProgramData\cattle`) {
			return `c:\ProgramData\cattle\etc\platform\api.crt`, `c:\ProgramData\cattle\etc\cattle\api.crt`
		}
		return `c:\ProgramData\pasturestack\etc\platform\api.crt`, `c:\ProgramData\pasturestack\etc\cattle\api.crt`
	}
	switch home {
	case "/var/lib/cattle":
		return "/var/lib/cattle/etc/platform/api.crt", "/var/lib/cattle/etc/cattle/api.crt"
	case "/var/lib/rancher":
		return "/var/lib/rancher/etc/platform/api.crt", "/var/lib/rancher/etc/cattle/api.crt"
	default:
		return "/var/lib/pasturestack/etc/platform/api.crt", "/var/lib/pasturestack/etc/cattle/api.crt"
	}
}

func approvedPublicKeyPath(configured string) string {
	cleaned := filepath.ToSlash(filepath.Clean(configured))
	for _, home := range []string{
		"/var/lib/pasturestack", "/var/lib/cattle", "/var/lib/rancher",
		`c:\ProgramData\pasturestack`, `c:\ProgramData\cattle`,
	} {
		preferred, legacy := approvedPublicKeyPaths(home)
		if cleaned == filepath.ToSlash(filepath.Clean(preferred)) {
			return preferred
		}
		if cleaned == filepath.ToSlash(filepath.Clean(legacy)) {
			return legacy
		}
	}
	return ""
}

func HostProxy() string {
	return DefaultValue("HOST_API_PROXY", "")
}

func Labels() map[string]string {
	val := DefaultValue("HOST_LABELS", "")
	ret := map[string]string{}
	if val != "" {
		m, err := url.ParseQuery(val)
		if err != nil {
			log.Error(err)
		}
		for k, v := range m {
			ret[strings.TrimSpace(k)] = strings.TrimSpace(v[0])
		}
	}
	return ret
}

func DockerEnable() bool {
	return DefaultValue("DOCKER_ENABLED", "true") == "true"
}

func DockerHostIP() string {
	return DefaultValue("DOCKER_HOST_IP", agentIP())
}

func SetDockerUUID() (string, error) {
	return GetUUIDFromFile("DOCKER_UUID", dockerUUIDFile(), true)
}

func DockerUUID() (string, error) {
	return GetUUIDFromFile("DOCKER_UUID", dockerUUIDFile(), false)
}

func dockerUUIDFile() string {
	return managedIdentityFile(
		DefaultValue("DOCKER_UUID_FILE", ".docker_uuid"),
		".docker_uuid",
	)
}

func CadvisorIP() string {
	return DefaultValue("CADVISOR_IP", "127.0.0.1")
}

func CadvisorPort() string {
	return DefaultValue("CADVISOR_PORT", "9344")
}

func DefaultValue(name string, df string) string {
	if value, ok := constants.ConfigOverride[name]; ok {
		return value
	}
	if result := os.Getenv(fmt.Sprintf("PLATFORM_%s", name)); result != "" {
		return result
	}
	if result := os.Getenv(fmt.Sprintf("CATTLE_%s", name)); result != "" {
		return result
	}
	return df
}

func Stamp() string {
	switch filepath.Clean(strings.TrimSpace(DefaultValue("STAMP_FILE", ""))) {
	case "/var/lib/pasturestack/.pyagent-stamp":
		return "/var/lib/pasturestack/.pyagent-stamp"
	case "/var/lib/cattle/.pyagent-stamp":
		return "/var/lib/cattle/.pyagent-stamp"
	case "/var/lib/rancher/.pyagent-stamp":
		return "/var/lib/rancher/.pyagent-stamp"
	case `c:\ProgramData\pasturestack\.pyagent-stamp`:
		return `c:\ProgramData\pasturestack\.pyagent-stamp`
	case `c:\ProgramData\cattle\.pyagent-stamp`:
		return `c:\ProgramData\cattle\.pyagent-stamp`
	default:
		return defaultStampFile(Home())
	}
}

func defaultStampFile(home string) string {
	switch home {
	case "/var/lib/cattle":
		return "/var/lib/cattle/.pyagent-stamp"
	case "/var/lib/rancher":
		return "/var/lib/rancher/.pyagent-stamp"
	case `c:\ProgramData\cattle`:
		return `c:\ProgramData\cattle\.pyagent-stamp`
	case `c:\ProgramData\pasturestack`:
		return `c:\ProgramData\pasturestack\.pyagent-stamp`
	default:
		return "/var/lib/pasturestack/.pyagent-stamp"
	}
}
