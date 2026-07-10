package powerdns

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
	pdns "github.com/mittwald/go-powerdns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDNSSECCorrectionsSkipsCryptokeysWhenAutoDNSSECUnset(t *testing.T) {
	dsp := &powerdnsProvider{}

	dc, _ := models.NewDomainConfig("example.com")
	corrections, err := dsp.getDNSSECCorrections(dc)

	require.NoError(t, err)
	assert.Empty(t, corrections)
}

func TestGetDNSSECCorrectionsIgnoresMissingCryptokeysEndpoint(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(server.Close)

	client, err := pdns.New(
		pdns.WithBaseURL(server.URL),
		pdns.WithAPIKeyAuthentication("secret"),
	)
	require.NoError(t, err)

	dsp := &powerdnsProvider{
		client:     client,
		ServerName: "localhost",
	}

	dc, _ := models.NewDomainConfig("example.com")
	dc.AutoDNSSEC = "on"
	corrections, err := dsp.getDNSSECCorrections(dc)

	require.NoError(t, err)
	assert.Empty(t, corrections)
}
