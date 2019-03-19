package cluster

import (
	"context"
	clusterclient "github.com/fabric8-services/fabric8-cluster-client/cluster"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/toolchain-operator/pkg/config"
	"github.com/fabric8-services/toolchain-operator/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	"net/http"
	"os"
	"testing"
)

const TOKEN = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjlNTG5WaWFSa2hWajFHVDlrcFdVa3dISXdVRC13WmZVeFItM0Nwa0UtWHMiLCJ0eXAiOiJKV1QifQ.eyJpYXQiOjE1NTEyNjA3NTIsImlzcyI6Imh0dHA6Ly9sb2NhbGhvc3QiLCJqdGkiOiI2OTE1MmU1Mi05ZmNiLTQ3MjEtYjhlZC04MTgxY2UyOTY4ZDgiLCJzY29wZXMiOlsidW1hX3Byb3RlY3Rpb24iXSwic2VydmljZV9hY2NvdW50bmFtZSI6InRvb2xjaGFpbi1vcGVyYXRvciIsInN1YiI6ImJiNmQwNDNkLWYyNDMtNDU4Zi04NDk4LTJjMThhMTJkY2Y0NyJ9.D-t7lrfJ-nd4P62t6oXOrYY364h2yGxw23-2qoRMERdBED2E8pMAOk1yZeCk18FUn1TFslxL2nuYOE9bRL7i8qUQCGTzgFIk8QtIOw8iLSkRRPVHJGSraUSVZqsePgcU4Y_dCEZlEBkR_SPEZ5l5lm7QdfWd-JaCLnQVTW5oRPhEx0B6455UyX6Giy68ySO5WuBl0WHIvEHr6N3rSIZ7cptRAatvb9PEKxyajfBE1uC60jEE5iJwEfzv2BYBr07lhskTxQqno05In21_rRcBMjaLStVLHRVmb62hPw4FC3OGOU1wn9MmhlZVo9VYuVMjpl4qerX1l8ZkhIZpRXCpEg"

func TestSaveClusterOK(t *testing.T) {
	// given
	defer gock.OffAll()

	setupGockForAuth()
	gock.New("http://cluster").
		Post("api/clusters").
		MatchHeader("Authorization", "Bearer "+TOKEN).
		BodyString(`{"data":{"api-url":"https://api.dsaas-stage.openshift.com/","app-dns":"8a09.starter-us-east-2.openshiftapps.com","auth-client-default-scope":"user:full","auth-client-id":"codeready-toolchain","auth-client-secret":"oauthsecret","name":"dsaas-stage","service-account-token":"mysatoken","service-account-username":"system:serviceaccount:config-test:toolchain-sre","token-provider-id":"3d7b75e3-7053-4846-9b64-26cf42717692","type":"OSD"}}`).
		Reply(201)

	reset := setEnv()
	defer reset()
	c, err := config.NewConfiguration()
	require.NoError(t, err)

	i := dummyClusterConfigInformer{}
	clusterData, err := i.Inform()
	require.NoError(t, err)

	clusterService := NewClusterService(c)

	// when
	err = clusterService.CreateCluster(context.Background(), clusterData, httpsupport.WithRoundTripper(http.DefaultTransport))

	// then
	assert.NoError(t, err)
}

func TestSaveClusterFail(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		// given
		defer gock.OffAll()
		// to retrieve sa token
		gock.New("http://auth").
			Post("api/token").
			MatchHeader("Content-Type", "application/x-www-form-urlencoded").
			BodyString(`client_id=bb6d043d-f243-458f-8498-2c18a12dcf47&client_secret=secret&grant_type=client_credentials`).
			Reply(200).
			BodyString(`{"access_token": "bearer_token","token_type":"Bearer"}`)
		gock.New("http://cluster").
			Post("api/clusters").
			MatchHeader("Authorization", "Bearer "+"bearer_token").
			BodyString(`{"data":{"api-url":"https://api.dsaas-stage.openshift.com/","app-dns":"8a09.starter-us-east-2.openshiftapps.com","auth-client-default-scope":"user:full","auth-client-id":"codeready-toolchain","auth-client-secret":"oauthsecret","name":"dsaas-stage","service-account-token":"mysatoken","service-account-username":"system:serviceaccount:config-test:toolchain-sre","token-provider-id":"3d7b75e3-7053-4846-9b64-26cf42717692","type":"OSD"}}`).
			Reply(401).
			BodyString("unauthorized access")

		reset := setEnv()
		defer reset()

		c, err := config.NewConfiguration()
		require.NoError(t, err)

		i := dummyClusterConfigInformer{}
		clusterData, err := i.Inform()
		require.NoError(t, err)

		clusterService := NewClusterService(c)

		// when
		err = clusterService.CreateCluster(context.Background(), clusterData, httpsupport.WithRoundTripper(http.DefaultTransport))

		// then
		assert.EqualError(t, err, "received unexpected response code while adding cluster configuration in cluster management service. Response status: 401 Unauthorized. Response body: unauthorized access")
	})

	t.Run("bad request", func(t *testing.T) {
		// given
		defer gock.OffAll()
		setupGockForAuth()
		gock.New("http://cluster").
			Post("api/clusters").
			MatchHeader("Authorization", "Bearer "+TOKEN).
			BodyString(`{"data":{"api-url":"https://api.dsaas-stage.openshift.com/","app-dns":"8a09.starter-us-east-2.openshiftapps.com","auth-client-default-scope":"user:full","auth-client-id":"codeready-toolchain","auth-client-secret":"oauthsecret","name":"dsaas-stage","service-account-token":"mysatoken","service-account-username":"system:serviceaccount:config-test:toolchain-sre","token-provider-id":"3d7b75e3-7053-4846-9b64-26cf42717692","type":"OSD"}}`).
			Reply(400).
			BodyString("something bad happened")

		reset := setEnv()
		defer reset()

		c, err := config.NewConfiguration()
		require.NoError(t, err)

		i := dummyClusterConfigInformer{}
		clusterData, err := i.Inform()
		require.NoError(t, err)

		clusterService := NewClusterService(c)

		// when
		err = clusterService.CreateCluster(context.Background(), clusterData, httpsupport.WithRoundTripper(http.DefaultTransport))

		// then
		assert.EqualError(t, err, "received unexpected response code while adding cluster configuration in cluster management service. Response status: 400 Bad Request. Response body: something bad happened")
	})

	t.Run("internal server error", func(t *testing.T) {
		// given
		defer gock.OffAll()
		setupGockForAuth()
		gock.New("http://cluster").
			Post("api/clusters").
			MatchHeader("Authorization", "Bearer "+TOKEN).
			BodyString(`{"data":{"api-url":"https://api.dsaas-stage.openshift.com/","app-dns":"8a09.starter-us-east-2.openshiftapps.com","auth-client-default-scope":"user:full","auth-client-id":"codeready-toolchain","auth-client-secret":"oauthsecret","name":"dsaas-stage","service-account-token":"mysatoken","service-account-username":"system:serviceaccount:config-test:toolchain-sre","token-provider-id":"3d7b75e3-7053-4846-9b64-26cf42717692","type":"OSD"}}`).
			Reply(500).
			BodyString("something went wrong")

		reset := setEnv()
		defer reset()

		c, err := config.NewConfiguration()
		require.NoError(t, err)

		i := dummyClusterConfigInformer{}
		clusterData, err := i.Inform()
		require.NoError(t, err)

		clusterService := NewClusterService(c)

		// when
		err = clusterService.CreateCluster(context.Background(), clusterData, httpsupport.WithRoundTripper(http.DefaultTransport))

		// then
		assert.EqualError(t, err, "received unexpected response code while adding cluster configuration in cluster management service. Response status: 500 Internal Server Error. Response body: something went wrong")
	})
}

type dummyClusterConfigInformer struct {
}

func (d dummyClusterConfigInformer) Inform(options ...SASecretOption) (*clusterclient.CreateClusterData, error) {
	tokenID := "3d7b75e3-7053-4846-9b64-26cf42717692"
	return &clusterclient.CreateClusterData{
		Name:                   os.Getenv("CLUSTER_NAME"),
		APIURL:                 `https://api.` + os.Getenv("CLUSTER_NAME") + `.openshift.com/`,
		AppDNS:                 "8a09.starter-us-east-2.openshiftapps.com",
		ServiceAccountToken:    "mysatoken",
		ServiceAccountUsername: "system:serviceaccount:config-test:toolchain-sre",
		AuthClientID:           "codeready-toolchain",
		AuthClientSecret:       "oauthsecret",
		AuthClientDefaultScope: "user:full",
		TokenProviderID:        &tokenID,
		Type:                   "OSD",
	}, nil
}

func setupGockForAuth() {
	// to retrieve sa token
	gock.New("http://auth").
		Post("api/token").
		MatchHeader("Content-Type", "application/x-www-form-urlencoded").
		BodyString(`client_id=bb6d043d-f243-458f-8498-2c18a12dcf47&client_secret=secret&grant_type=client_credentials`).
		Reply(200).
		BodyString(`{"access_token":"` + TOKEN + `","token_type":"Bearer"}`)
}

func setEnv() func() {
	return test.SetEnv(
		test.Env("CLUSTER_NAME", "dsaas-stage"),
		test.Env("AUTH_URL", "http://auth"),
		test.Env("CLUSTER_URL", "http://cluster"),
		test.Env("TC_CLIENT_ID", "bb6d043d-f243-458f-8498-2c18a12dcf47"),
		test.Env("TC_CLIENT_SECRET", "secret"),
	)
}
