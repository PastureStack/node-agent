package aws

import (
	"context"
	"time"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"

	"github.com/PastureStack/node-agent/cloudprovider"
	"github.com/PastureStack/node-agent/core/hostinfo"
)

const (
	awsTag          = "aws"
	metadataTimeout = 5 * time.Second
)

type Provider struct {
	client     metadataClient
	interval   time.Duration
	retryCount int
}

type metadataClient interface {
	getInstanceIdentityDocument(context.Context) (instanceIdentity, error)
}

type metadataClientImpl struct {
	client *imds.Client
}

type instanceIdentity struct {
	Region           string
	AvailabilityZone string
}

func init() {
	cloudprovider.AddCloudProvider(awsTag, &Provider{
		retryCount: 6,
		interval:   time.Second * 30,
	})
}

func (m metadataClientImpl) getInstanceIdentityDocument(ctx context.Context) (instanceIdentity, error) {
	document, err := m.client.GetInstanceIdentityDocument(ctx, nil)
	if err != nil {
		return instanceIdentity{}, err
	}
	return instanceIdentity{
		Region:           document.Region,
		AvailabilityZone: document.AvailabilityZone,
	}, nil
}

func (p *Provider) Init() error {
	p.client = metadataClientImpl{client: imds.New(imds.Options{
		EnableFallback: awsv2.FalseTernary,
	})}
	return nil
}

func (p *Provider) Name() string {
	return awsTag
}

func (p *Provider) GetHostInfo() (i *hostinfo.Info, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), metadataTimeout)
	defer cancel()

	document, err := p.client.getInstanceIdentityDocument(ctx)
	if err != nil {
		return
	}
	i = &hostinfo.Info{}
	i.Labels = map[string]string{}
	i.Labels[cloudprovider.RegionLabel] = document.Region
	i.Labels[cloudprovider.AvailabilityZoneLabel] = document.AvailabilityZone
	i.Labels[cloudprovider.CloudProviderLabel] = awsTag
	return
}

func (p *Provider) RetryCount() int {
	return p.retryCount
}

func (p *Provider) Interval() time.Duration {
	return p.interval
}
