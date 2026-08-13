package proxy

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rancher/websocket-proxy/common"

	. "gopkg.in/check.v1"
)

func TestProxyTLSRequiresTrustedCertificate(t *testing.T) {
	if newProxyTLSConfig(nil).InsecureSkipVerify {
		t.Fatal("proxy TLS configuration disables certificate verification")
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if response, err := newProxyHTTPClient(nil).Get(server.URL); err == nil {
		response.Body.Close()
		t.Fatal("an untrusted TLS certificate was accepted")
	}

	certificate, err := x509.ParseCertificate(server.TLS.Certificates[0].Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(certificate)
	response, err := newProxyHTTPClient(roots).Get(server.URL)
	if err != nil {
		t.Fatalf("trusted TLS certificate was rejected: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("unexpected response status: %d", response.StatusCode)
	}
}

func TestInitialRequestBodyIsBounded(t *testing.T) {
	body, err := readInitialBody(bytes.NewBufferString("hello"), 5)
	if err != nil || string(body) != "hello" {
		t.Fatalf("valid request body failed: body=%q err=%v", body, err)
	}
	if _, err := readInitialBody(strings.NewReader(""), maxInitialRequestBody+1); err == nil {
		t.Fatal("oversized initial request body was accepted")
	}
	if _, err := readInitialBody(strings.NewReader("short"), 10); err == nil {
		t.Fatal("truncated initial request body was accepted")
	}
}

func TestContentLengthRejectsInvalidValues(t *testing.T) {
	request, err := http.NewRequest(http.MethodPost, "http://example.invalid", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Length", "-1")
	if _, err := setContentLength(request); err == nil {
		t.Fatal("negative Content-Length was accepted")
	}
	request.Header.Set("Content-Length", "5")
	if length, err := setContentLength(request); err != nil || length != 5 {
		t.Fatalf("valid Content-Length failed: length=%d err=%v", length, err)
	}
}

const host = "localhost:23425"

func Test(t *testing.T) {
	TestingT(t)
}

type ProxyTestSuite struct {
}

var _ = Suite(&ProxyTestSuite{})

func (s *ProxyTestSuite) TestPost(c *C) {
	input := make(chan string)
	output := make(chan common.Message)

	handler := &Handler{}
	go handler.Handle("key", "init", input, output)

	input <- marshal(c, common.HTTPMessage{
		Method: "GET",
		URL:    "http://" + host + "/foo",
		Body:   []byte("foo"),
	})
	input <- marshal(c, common.HTTPMessage{
		Body: []byte("bar"),
	})
	input <- marshal(c, common.HTTPMessage{
		EOF: true,
	})

	var response common.HTTPMessage
	unmarshal(c, <-output, &response)
	c.Assert(response.Code, Equals, 200)
	response = common.HTTPMessage{}

	//Second message will have the payload
	unmarshal(c, <-output, &response)
	c.Assert(string(response.Body), Equals, "foobar")
}

func unmarshal(c *C, msg common.Message, httpMessage *common.HTTPMessage) {
	if err := json.Unmarshal([]byte(msg.Body), httpMessage); err != nil {
		c.Fatal(err)
	}
}

func marshal(c *C, obj interface{}) string {
	bytes, err := json.Marshal(obj)
	if err != nil {
		c.Fatal(err)
	}
	return string(bytes)
}

func (s *ProxyTestSuite) SetUpSuite(c *C) {
	listener, err := net.Listen("tcp", host)
	c.Assert(err, IsNil)
	go http.Serve(listener, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.Write(bytes)
	}))
}
