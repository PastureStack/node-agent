module github.com/PastureStack/node-agent

go 1.26.0

toolchain go1.27.0

require (
	github.com/PastureStack/websocket-proxy v0.0.0-00010101000000-000000000000
	github.com/aws/aws-sdk-go-v2 v1.43.7
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.38
	github.com/containerd/errdefs v1.0.0
	github.com/docker/go-connections v0.8.1
	github.com/docker/go-units v0.5.0
	github.com/go-viper/mapstructure/v2 v2.5.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/golang/glog v1.2.5
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/moby/moby/api v1.55.0
	github.com/moby/moby/client v0.5.1
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/rancher/event-subscriber v0.0.0-20170627155511-cdcd1659ec46
	github.com/rancher/go-rancher v0.1.1-0.20161130212115-f4560b58215d
	github.com/rancher/log v0.1.0-u2
	github.com/shirou/gopsutil/v4 v4.26.7
	github.com/sirupsen/logrus v1.10.1
	github.com/vishvananda/netlink v1.3.1
	github.com/vishvananda/netns v0.0.5
	golang.org/x/sys v0.47.0
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/aws/smithy-go v1.27.8 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/ebitengine/purego v0.10.2 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/tklauser/go-sysconf v0.3.16 // indirect
	github.com/tklauser/numcpus v0.11.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.70.0 // indirect
	go.opentelemetry.io/otel v1.45.0 // indirect
	go.opentelemetry.io/otel/metric v1.45.0 // indirect
	go.opentelemetry.io/otel/trace v1.45.0 // indirect
)

replace github.com/PastureStack/websocket-proxy => ../websocket-proxy

replace github.com/rancher/event-subscriber => ./third_party/event-subscriber

replace github.com/rancher/go-rancher => ./third_party/go-rancher

replace github.com/rancher/log => ./third_party/log
