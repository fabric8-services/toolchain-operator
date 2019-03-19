package cluster

import (
	"context"
	clusterclient "github.com/fabric8-services/fabric8-cluster-client/cluster"
	"github.com/fabric8-services/fabric8-common/auth"
	"github.com/fabric8-services/fabric8-common/goasupport"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	goaclient "github.com/goadesign/goa/client"
	"github.com/pkg/errors"
	"net/http"
	"net/url"
)

type Config interface {
	GetClusterServiceURL() string
	GetAuthServiceURL() string
	GetClientID() string
	GetClientSecret() string
	GetClusterName() string
}

type clusterService struct {
	config Config
}

func NewClusterService(config Config) *clusterService {
	return &clusterService{config}
}

// CreateCluster adds cluster configuration in cluster service
func (s clusterService) CreateCluster(ctx context.Context, data *clusterclient.CreateClusterData, options ...httpsupport.HTTPClientOption) error {
	signer := newJWTSASigner(ctx, s.config, options...)
	remoteClusterService, err := signer.createSignedClient()
	if err != nil {
		return errors.Wrapf(err, "failed to create JWT signer for cluster service")
	}

	clusterURL := s.config.GetClusterServiceURL()
	clusterData := &clusterclient.CreateClustersPayload{Data: data}

	res, err := remoteClusterService.CreateClusters(goasupport.ForwardContextRequestID(ctx), clusterclient.CreateClustersPath(), clusterData)
	if err != nil {
		return errors.Wrapf(err, "failed to add cluster configuration for cluster %s", clusterURL)
	}
	defer func() {
		if err := httpsupport.CloseResponse(res); err != nil {
			log.Error(err, "error during closing response body when adding cluster configuration")
		}
	}()

	bodyString, err := httpsupport.ReadBody(res.Body)
	if err != nil {
		return errors.Wrapf(err, "unable to read response while saving cluster configuration")
	}
	if res.StatusCode != http.StatusCreated {
		err := errors.Errorf("received unexpected response code while adding cluster configuration in cluster management service. Response status: %s. Response body: %s", res.Status, bodyString)
		log.Error(err, "failed to add cluster configuration in cluster management service",
			"cluster_url", clusterURL, "response_status", res.Status, "response_body", bodyString)
		return err
	}
	return nil
}

type saSigner interface {
	createSignedClient() (*clusterclient.Client, error)
}

type jwtSASigner struct {
	ctx     context.Context
	config  Config
	options []httpsupport.HTTPClientOption
}

func newJWTSASigner(ctx context.Context, config Config, options ...httpsupport.HTTPClientOption) saSigner {
	return &jwtSASigner{ctx, config, options}
}

// createSignedClient creates a client with a JWT signer which uses the Auth Service Account token
func (c jwtSASigner) createSignedClient() (*clusterclient.Client, error) {
	cln, err := c.createClient(c.ctx)
	if err != nil {
		return nil, err
	}
	token, err := auth.ServiceAccountToken(c.ctx, c.config, c.config.GetClientID(), c.config.GetClientSecret(), c.options...)
	if err != nil {
		return nil, err
	}

	cln.SetJWTSigner(
		&goaclient.JWTSigner{
			TokenSource: &goaclient.StaticTokenSource{
				StaticToken: &goaclient.StaticToken{
					Value: token,
					Type:  "Bearer"}}})

	return cln, nil
}

func (c jwtSASigner) createClient(ctx context.Context) (*clusterclient.Client, error) {
	u, err := url.Parse(c.config.GetClusterServiceURL())
	if err != nil {
		return nil, err
	}

	httpClient := http.DefaultClient

	if c.options != nil {
		for _, opt := range c.options {
			opt(httpClient)
		}
	}
	cln := clusterclient.New(goaclient.HTTPClientDoer(httpClient))

	cln.Host = u.Host
	cln.Scheme = u.Scheme
	return cln, nil
}
