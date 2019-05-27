#!/usr/bin/env bash

oc login -u system:admin

export minishift=true
export AUTH_URL=https://auth.openshift.io/
export PROJECT=toolchain-manager
export SA_NAME=toolchain-sre
export OAUTH_SECRET=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)
# setup cluster scoped resources
oc apply -f deploy/olm-catalog/manifests/0.0.2/dsaas-cluster-admin.ClusterRole.yaml
oc apply -f deploy/olm-catalog/manifests/0.0.2/online-registration.ClusterRole.yaml

# setup namespace
oc new-project $PROJECT

# remove self-provisioner cluster role
oc adm policy remove-cluster-role-from-group self-provisioner system:authenticated system:authenticated:oauth

# create sa and create required bindings
oc create sa $SA_NAME
oc adm policy add-cluster-role-to-user self-provisioner -z $SA_NAME
oc adm policy add-cluster-role-to-user dsaas-cluster-admin -z $SA_NAME

export SA_TOKEN="echo $(cut -d':' -f2 <<< $(oc describe secret $(cut -d':' -f2 <<< $(oc get sa toolchain-sre -o yaml | grep toolchain-sre-token)) | grep token:))"

# create oauthclient
oc create -f <(echo '
kind: OAuthClient
apiVersion: oauth.openshift.io/v1
metadata:
 name: codeready-toolchain
secret: '$OAUTH_SECRET'
accessTokenMaxAgeSeconds: 0
redirectURIs:
 - '$(echo $AUTH_URL)'
grantMethod: auto
')

# create route and find routing subdomain
oc create route edge my-route --service=frontend --port 8080
export ROUTE_SUBDOMAIN=$(cut -d'.' -f 2- <<<"$(oc get routes my-route -o custom-columns=HOST/PORT:spec.host --no-headers)")
oc delete route my-route

if !minishift; then
    # todo for openshift 4.x cluster
    export CLUSTER_NAME=""
    export PUBLIC_URL=""
else
    export CLUSTER_NAME=minishift
    export PUBLIC_URL=https://$(minishift ip):8443
fi

# cluster configuration
echo '
{
    "name": "minishift",
    "api-url": "'$PUBLIC_URL'",
    "app-dns": "'$ROUTE_SUBDOMAIN'",
    "service-account-token": "'$SA_TOKEN'",
    "service-account-username": "'$(echo "system:serviceaccount:$PROJECT:$SA_NAME")'",
    "token-provider-id": "zba1a021-7b215-4e17-9162-e21db8b5ee67",
    "auth-client-id": "codeready-toolchain",
    "auth-client-secret": "'$OAUTH_SECRET'",
    "auth-client-default-scope": "user:full",
    "type": "OSD"
}
'