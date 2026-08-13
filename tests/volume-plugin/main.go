package main

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	testDriver = "convoytest"
	testSocket = "/var/run/convoytest.sock"
	testRoot   = "/tmp/convoytest"
)

type volumePlugin struct {
	driver string
	root   string
	mu     sync.Mutex
}

type volumeRequest struct {
	Name string
}

type volumeInfo struct {
	Name       string
	Mountpoint string
}

func main() {
	driver, socket, root := fixturePaths()

	must(os.MkdirAll(root, 0755))
	must(os.MkdirAll("/etc/docker/plugins", 0755))
	must(os.WriteFile(filepath.Join("/etc/docker/plugins", driver+".spec"),
		[]byte("unix://"+socket+"\n"), 0644))
	_ = os.Remove(socket)

	listener, err := net.Listen("unix", socket)
	must(err)
	must(os.Chmod(socket, 0666))

	plugin := &volumePlugin{driver: driver, root: root}
	mux := http.NewServeMux()
	mux.HandleFunc("/Plugin.Activate", plugin.activate)
	mux.HandleFunc("/VolumeDriver.Create", plugin.create)
	mux.HandleFunc("/VolumeDriver.Remove", plugin.remove)
	mux.HandleFunc("/VolumeDriver.Mount", plugin.mount)
	mux.HandleFunc("/VolumeDriver.Unmount", plugin.ok)
	mux.HandleFunc("/VolumeDriver.Path", plugin.path)
	mux.HandleFunc("/VolumeDriver.Get", plugin.get)
	mux.HandleFunc("/VolumeDriver.List", plugin.list)
	mux.HandleFunc("/VolumeDriver.Capabilities", plugin.capabilities)
	must(http.Serve(listener, mux))
}

func fixturePaths() (string, string, string) {
	return testDriver, testSocket, testRoot
}

func (p *volumePlugin) activate(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{"Implements": []string{"VolumeDriver"}})
}

func (p *volumePlugin) create(w http.ResponseWriter, r *http.Request) {
	req, err := decodeRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	writeError(w, os.MkdirAll(p.volumePath(req.Name), 0755))
}

func (p *volumePlugin) remove(w http.ResponseWriter, r *http.Request) {
	req, err := decodeRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	writeError(w, os.RemoveAll(p.volumePath(req.Name)))
}

func (p *volumePlugin) mount(w http.ResponseWriter, r *http.Request) {
	req, err := decodeRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	path := p.volumePath(req.Name)
	if err := os.MkdirAll(path, 0755); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]interface{}{"Mountpoint": path, "Err": ""})
}

func (p *volumePlugin) path(w http.ResponseWriter, r *http.Request) {
	req, err := decodeRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]interface{}{
		"Mountpoint": p.volumePath(req.Name),
		"Err":        "",
	})
}

func (p *volumePlugin) get(w http.ResponseWriter, r *http.Request) {
	req, err := decodeRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]interface{}{
		"Volume": volumeInfo{Name: req.Name, Mountpoint: p.volumePath(req.Name)},
		"Err":    "",
	})
}

func (p *volumePlugin) list(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(p.root)
	if err != nil {
		writeError(w, err)
		return
	}
	volumes := make([]volumeInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			volumes = append(volumes, volumeInfo{
				Name:       entry.Name(),
				Mountpoint: p.volumePath(entry.Name()),
			})
		}
	}
	writeJSON(w, map[string]interface{}{"Volumes": volumes, "Err": ""})
}

func (p *volumePlugin) capabilities(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"Capabilities": map[string]string{"Scope": "local"},
		"Err":          "",
	})
}

func (p *volumePlugin) ok(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{"Err": ""})
}

func (p *volumePlugin) volumePath(name string) string {
	clean := strings.TrimPrefix(filepath.Clean("/"+name), string(os.PathSeparator))
	return filepath.Join(p.root, clean)
}

func decodeRequest(r *http.Request) (volumeRequest, error) {
	defer r.Body.Close()
	var req volumeRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

func writeJSON(w http.ResponseWriter, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	writeJSON(w, map[string]string{"Err": msg})
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
