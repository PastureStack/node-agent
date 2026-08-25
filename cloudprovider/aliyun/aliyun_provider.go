package aliyun

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PastureStack/node-agent/cloudprovider"
	"github.com/PastureStack/node-agent/core/hostinfo"
)

const (
	aliyunTag                 = "aliyun"
	aliyunMetadataBaseURL     = "http://100.100.100.200/latest/"
	aliyunTokenPath           = "api/token"
	aliyunRegionPath          = "meta-data/region-id"
	aliyunZonePath            = "meta-data/zone-id"
	aliyunTokenTTLHeader      = "X-aliyun-ecs-metadata-token-ttl-seconds"
	aliyunTokenHeader         = "X-aliyun-ecs-metadata-token"
	aliyunTokenTTL            = "60"
	aliyunMetadataTimeout     = 5 * time.Second
	aliyunMetadataMaxResponse = 4096
)

type Provider struct {
	client     metadataClient
	interval   time.Duration
	retryCount int
}

type metadataClient interface {
	RegionAndZone(context.Context) (string, string, error)
}

type metadataClientImpl struct {
	client  *http.Client
	baseURL string
}

func init() {
	cloudprovider.AddCloudProvider(aliyunTag, &Provider{
		retryCount: 2, // aliyun sdk itself will also retry 5 times for some error, like timeout
		interval:   time.Second * 30,
	})
}

func (m metadataClientImpl) RegionAndZone(ctx context.Context) (string, string, error) {
	token, err := m.read(ctx, http.MethodPut, aliyunTokenPath, "")
	if err != nil {
		return "", "", fmt.Errorf("get Aliyun IMDSv2 token: %w", err)
	}
	region, err := m.read(ctx, http.MethodGet, aliyunRegionPath, token)
	if err != nil {
		return "", "", fmt.Errorf("get Aliyun region: %w", err)
	}
	zone, err := m.read(ctx, http.MethodGet, aliyunZonePath, token)
	if err != nil {
		return "", "", fmt.Errorf("get Aliyun zone: %w", err)
	}
	return region, zone, nil
}

func (m metadataClientImpl) read(ctx context.Context, method, path, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, method, m.baseURL+path, nil)
	if err != nil {
		return "", err
	}
	if method == http.MethodPut {
		req.Header.Set(aliyunTokenTTLHeader, aliyunTokenTTL)
	} else {
		req.Header.Set(aliyunTokenHeader, token)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, aliyunMetadataMaxResponse+1))
	if err != nil {
		return "", err
	}
	if len(body) > aliyunMetadataMaxResponse {
		return "", fmt.Errorf("response exceeds %d bytes", aliyunMetadataMaxResponse)
	}
	value := strings.TrimSpace(string(body))
	if value == "" {
		return "", fmt.Errorf("empty metadata response")
	}
	return value, nil
}

func (p *Provider) Init() error {
	p.client = metadataClientImpl{
		baseURL: aliyunMetadataBaseURL,
		client: &http.Client{
			Timeout: aliyunMetadataTimeout,
			Transport: &http.Transport{
				Proxy: nil,
			},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	return nil
}

func (p *Provider) Name() string {
	return aliyunTag
}

func (p *Provider) GetHostInfo() (i *hostinfo.Info, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), aliyunMetadataTimeout)
	defer cancel()

	region, zone, err := p.client.RegionAndZone(ctx)
	if err != nil {
		return
	}
	i = &hostinfo.Info{}
	i.Labels = map[string]string{}
	i.Labels[cloudprovider.RegionLabel] = region
	i.Labels[cloudprovider.AvailabilityZoneLabel] = zone
	i.Labels[cloudprovider.CloudProviderLabel] = aliyunTag
	return
}

func (p *Provider) RetryCount() int {
	return p.retryCount
}

func (p *Provider) Interval() time.Duration {
	return p.interval
}
