package exec

import (
	"encoding/base64"
	"io"
	"net/url"
	"runtime"
	"strconv"
	"strings"

	"github.com/PastureStack/node-agent/service/hostapi/auth"
	"github.com/PastureStack/node-agent/service/hostapi/events"
	"github.com/docker/distribution/context"
	"github.com/docker/docker/api/types"
	"github.com/rancher/log"
	"github.com/rancher/websocket-proxy/backend"
	"github.com/rancher/websocket-proxy/common"
)

type Handler struct {
}

const maxTTYDimension = 65535

func (h *Handler) Handle(key string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {
	defer backend.SignalHandlerClosed(key, response)

	requestURL, err := url.Parse(initialMessage)
	if err != nil {
		log.Errorf("Couldn't parse url=%v error=%v", initialMessage, err)
		return
	}
	tokenString := requestURL.Query().Get("token")
	token, valid := auth.GetAndCheckToken(tokenString)
	if !valid {
		return
	}

	execMap, ok := auth.GetClaimMap(token, "exec")
	if !ok {
		log.Errorf("Token missing exec claim.")
		return
	}
	execConfig, id := convert(execMap)

	client, err := events.NewDockerClient()
	if err != nil {
		log.Errorf("Couldn't get docker client. error=%v", err)
		return
	}

	execObj, err := client.ContainerExecCreate(context.Background(), id, execConfig)
	if err != nil {
		return
	}
	hijackResp, err := client.ContainerExecAttach(context.Background(), execObj.ID, execConfig)
	if err != nil {
		return
	}

	go func(w io.WriteCloser) {
		for {
			msg, ok := <-incomingMessages
			if !ok {
				if _, err := w.Write([]byte("\x04")); err != nil {
					log.Errorf("Error writing EOT message. error=%v", err)
				}
				w.Close()
				return
			}
			if strings.HasPrefix(msg, ":resizeTTY:") {
				resizeOptions, err := parseTTYResize(msg)
				if err != nil {
					log.Errorf("Error decoding TTY dimensions. error=%v", err)
					continue
				}
				client.ContainerExecResize(context.Background(), execObj.ID, resizeOptions)
				continue
			}
			data, err := base64.StdEncoding.DecodeString(msg)
			if err != nil {
				log.Errorf("Error decoding message. error=%v", err)
				continue
			}
			w.Write([]byte(data))
		}
	}(hijackResp.Conn)

	buffer := make([]byte, 4096, 4096)
	for {
		c, err := hijackResp.Reader.Read(buffer)
		if c > 0 {
			text := base64.StdEncoding.EncodeToString(buffer[:c])
			message := common.Message{
				Key:  key,
				Type: common.Body,
				Body: text,
			}
			response <- message
		}
		if err != nil {
			break
		}
	}
}

func parseTTYResize(message string) (types.ResizeOptions, error) {
	const prefix = ":resizeTTY:"
	if !strings.HasPrefix(message, prefix) {
		return types.ResizeOptions{}, strconv.ErrSyntax
	}
	parts := strings.Split(strings.TrimPrefix(message, prefix), ",")
	if len(parts) != 2 {
		return types.ResizeOptions{}, strconv.ErrSyntax
	}
	width, err := parseTTYDimension(parts[0])
	if err != nil {
		return types.ResizeOptions{}, err
	}
	height, err := parseTTYDimension(parts[1])
	if err != nil {
		return types.ResizeOptions{}, err
	}
	return types.ResizeOptions{Width: width, Height: height}, nil
}

func parseTTYDimension(value string) (uint, error) {
	dimension, err := strconv.ParseUint(value, 10, 16)
	if err != nil || dimension == 0 || dimension > maxTTYDimension {
		if err != nil {
			return 0, err
		}
		return 0, strconv.ErrRange
	}
	return uint(dimension), nil
}

func convert(execMap map[string]interface{}) (types.ExecConfig, string) {
	// Not fancy at all
	config := types.ExecConfig{}
	containerID := ""

	if param, ok := execMap["AttachStdin"]; ok {
		if val, ok := param.(bool); ok {
			config.AttachStdin = val
		}
	}

	if param, ok := execMap["AttachStdout"]; ok {
		if val, ok := param.(bool); ok {
			config.AttachStdout = val
		}
	}

	if param, ok := execMap["AttachStderr"]; ok {
		if val, ok := param.(bool); ok {
			config.AttachStderr = val
		}
	}

	if param, ok := execMap["Tty"]; ok {
		if val, ok := param.(bool); ok {
			config.Tty = val
		}
	}

	if param, ok := execMap["Container"]; ok {
		if val, ok := param.(string); ok {
			containerID = val
		}
	}

	if param, ok := execMap["Cmd"]; ok {
		cmd := []string{}
		if list, ok := param.([]interface{}); ok {
			for _, item := range list {
				if val, ok := item.(string); ok {
					cmd = append(cmd, val)
				}
			}
		}
		config.Cmd = cmd
	}

	if runtime.GOOS == "windows" {
		config.Cmd = []string{"powershell"}
	}

	return config, containerID
}
