ifndef DEV_MK
DEV_MK:=# Prevent repeated "-include".

include ./make/verbose.mk
include ./make/git.mk

DOCKER_REPO?=quay.io/openshiftio
IMAGE_NAME?=toolchain-operator
REGISTRY_URI=quay.io

TIMESTAMP:=$(shell date +%s)
TAG?=$(GIT_COMMIT_ID_SHORT)-$(TIMESTAMP)
OPENSHIFT_VERSION?=4

DEPLOY_DIR:=deploy
NAMESPACE:=codeready-toolchain

.PHONY: push-operator-image
## Push the operator container image to a container registry
push-operator-image: build-operator-image
	@docker login -u $(QUAY_USERNAME) -p $(QUAY_PASSWORD) $(REGISTRY_URI)
	docker push $(DOCKER_REPO)/$(IMAGE_NAME):$(TAG)

.PHONY: create-resources
create-resources:
	@echo "Logging using system:admin..."
	@oc login -u system:admin
	@echo "Creating cluster scoped resources..."
	@cat $(DEPLOY_DIR)/global-manifests.yaml | sed s/\REPLACE_NAMESPACE/$(NAMESPACE)/ | oc apply -f -
	@oc apply -f $(DEPLOY_DIR)/olm-catalog/manifests/0.0.2/dsaas-cluster-admin.ClusterRole.yaml
	@oc apply -f $(DEPLOY_DIR)/olm-catalog/manifests/0.0.2/online-registration.ClusterRole.yaml
	@echo "Creating namespaced scope resources..."
	@cat $(DEPLOY_DIR)/namespace-manifests.yaml | sed s/\REPLACE_NAMESPACE/$(NAMESPACE)/ | oc apply -f -

.PHONY: create-cr
create-cr:
	@echo "Creating Custom Resource..."
	@oc create -f $(DEPLOY_DIR)/crds/codeready_v1alpha1_toolchainenabler_cr.yaml

.PHONY: build-operator-image
build-operator-image:
	docker build -t $(DOCKER_REPO)/$(IMAGE_NAME):$(TAG) -f Dockerfile.dev .

.PHONY: clean-all
clean-all:  clean-operator clean-resources delete-project

.PHONY: create-project
create-project:
	@echo "Creating project $(NAMESPACE)"
	@oc new-project $(NAMESPACE) || true

.PHONY: delete-project
delete-project:
	@echo "Deleting project $(NAMESPACE)"
	@oc delete project $(NAMESPACE) || true

.PHONY: clean-operator
clean-operator:
	@echo "Deleting Deployment for Operator"
	@cat $(DEPLOY_DIR)/operator.yaml | sed s/\:latest/:$(TAG)/ | oc delete -f - || true

.PHONY: clean-resources
clean-resources:
	@echo "Deleting sub resources..."
	@oc delete -f $(DEPLOY_DIR)/crds/codeready_v1alpha1_toolchainenabler_cr.yaml || true
	@echo "Deleting namespaced scope resources..."
	@cat $(DEPLOY_DIR)/namespace-manifests.yaml | sed s/\REPLACE_NAMESPACE/$(NAMESPACE)/ | oc delete -f - || true
	@echo "deleting cluster scoped resources"
	@cat $(DEPLOY_DIR)/global-manifests.yaml | sed s/\REPLACE_NAMESPACE/$(NAMESPACE)/ | oc delete -f - || true
	@echo "Deleting OAuthClient 'codeready-toolchain'"
	@oc delete oauthclient codeready-toolchain || true
	@echo "Deleting dsaas-cluster-admin ClusterRole"
	@oc delete -f $(DEPLOY_DIR)/olm-catalog/manifests/0.0.2/dsaas-cluster-admin.ClusterRole.yaml || true
	@echo "Deleting online-registration ClusterRole"
	@oc delete -f $(DEPLOY_DIR)/olm-catalog/manifests/0.0.2/online-registration.ClusterRole.yaml || true
	@echo "Deleting online-registration ClusterRoleBinding"
	@oc delete clusterrolebinding online-registration || true
	@echo "Deleting online-registration sa from openshift-infra namespace"
	@oc delete sa online-registration -n openshift-infra || true

.PHONY: deploy-operator
deploy-operator: build build-operator-image
	@oc project $(NAMESPACE)
	@echo "Creating Deployment for Operator"
	@cat $(DEPLOY_DIR)/operator.yaml | sed s/\:latest/:$(TAG)/ | oc create -f -

.PHONY: minishift-start
minishift-start:
	minishift start --cpus 4 --memory 8GB
	-eval `minishift docker-env` && oc login -u system:admin

.PHONY: deploy-all
deploy-all: clean-resources delete-project create-project create-resources deploy-operator create-cr

endif
