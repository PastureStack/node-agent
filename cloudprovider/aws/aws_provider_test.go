package aws

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	. "gopkg.in/check.v1"

	"github.com/PastureStack/node-agent/cloudprovider"
	"github.com/PastureStack/node-agent/core/hostinfo"
	"github.com/PastureStack/node-agent/utilities/config"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	TestingT(t)
}

type ComputeTestSuite struct {
}

var _ = Suite(&ComputeTestSuite{})

func (s *ComputeTestSuite) SetUpSuite(c *C) {
}

type fakeReplyImpl struct{}

func (f fakeReplyImpl) getInstanceIdentityDocument(context.Context) (instanceIdentity, error) {
	return instanceIdentity{Region: "fake", AvailabilityZone: "fake"}, nil
}

type errorReplyImpl struct{}

func (e errorReplyImpl) getInstanceIdentityDocument(context.Context) (instanceIdentity, error) {
	return instanceIdentity{}, errors.New("fake error")
}

func (s *ComputeTestSuite) TestGetHostInfo(c *C) {
	os.Mkdir(config.StateDir(), 0755)
	p := Provider{
		interval:   time.Second,
		retryCount: 2,
	}
	i := &hostinfo.Info{}
	i.Labels = map[string]string{
		cloudprovider.RegionLabel:           "fake",
		cloudprovider.AvailabilityZoneLabel: "fake",
		cloudprovider.CloudProviderLabel:    awsTag,
	}

	p.client = fakeReplyImpl{}
	hostInfo, err := p.GetHostInfo()
	c.Assert(err, IsNil)
	c.Assert(hostInfo, DeepEquals, i)

	p.client = errorReplyImpl{}
	hostInfo, err = p.GetHostInfo()
	c.Assert(err, ErrorMatches, "fake error")
}
