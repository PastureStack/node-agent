package aliyun

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

func (f fakeReplyImpl) RegionAndZone(context.Context) (string, string, error) {
	return "fake", "fake", nil
}

type errorReplyImpl struct{}

func (e errorReplyImpl) RegionAndZone(context.Context) (string, string, error) {
	return "", "", errors.New("fake error")
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
		cloudprovider.CloudProviderLabel:    aliyunTag,
	}

	p.client = fakeReplyImpl{}
	hostInfo, err := p.GetHostInfo()
	c.Assert(err, IsNil)
	c.Assert(hostInfo, DeepEquals, i)

	p.client = errorReplyImpl{}
	hostInfo, err = p.GetHostInfo()
	c.Assert(err, ErrorMatches, "fake error")
}

func (s *ComputeTestSuite) TestIMDSv2RegionAndZone(c *C) {
	requests := make([]string, 0, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/api/token":
			c.Assert(r.Method, Equals, http.MethodPut)
			c.Assert(r.Header.Get(aliyunTokenTTLHeader), Equals, aliyunTokenTTL)
			_, _ = w.Write([]byte("test-token\n"))
		case "/meta-data/region-id":
			c.Assert(r.Header.Get(aliyunTokenHeader), Equals, "test-token")
			_, _ = w.Write([]byte("cn-test\n"))
		case "/meta-data/zone-id":
			c.Assert(r.Header.Get(aliyunTokenHeader), Equals, "test-token")
			_, _ = w.Write([]byte("cn-test-a\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := metadataClientImpl{client: server.Client(), baseURL: server.URL + "/"}
	region, zone, err := client.RegionAndZone(context.Background())
	c.Assert(err, IsNil)
	c.Assert(region, Equals, "cn-test")
	c.Assert(zone, Equals, "cn-test-a")
	c.Assert(strings.Join(requests, ","), Equals,
		"PUT /api/token,GET /meta-data/region-id,GET /meta-data/zone-id")
}

func (s *ComputeTestSuite) TestIMDSv2RejectsRedirect(c *C) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/unexpected", http.StatusFound)
	}))
	defer server.Close()

	client := metadataClientImpl{
		client: &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		baseURL: server.URL + "/",
	}
	_, _, err := client.RegionAndZone(context.Background())
	c.Assert(err, ErrorMatches, "get Aliyun IMDSv2 token: unexpected HTTP status 302")
}
