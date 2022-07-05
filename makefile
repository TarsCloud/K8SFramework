.DEFAULT_GOAL := help
SHELL := /bin/bash -o pipefail
ENV_DOCKER             ?= docker
ENV_DOCKER_RUN         ?= $(ENV_DOCKER) run $(if $(findstring $(DOCKER_RUN_WITHOUT_IT),1),,-it)
ENV_HELM               ?= helm
ENV_KUBECTL            ?= kubectl
ENV_OS_NAME            ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')

override PARAMS:= $(shell sh ./params.sh PARAMS)
define func_read_params
$1 ?= $(strip $(shell sh ./params.sh $1))
endef
$(foreach param,$(PARAMS),$(eval $(call func_read_params,$(param))))
$(foreach param,$(PARAMS),$(info read [ $(param) ]  =  $($(param))))

PLATFORMS ?= $(strip,$(sort $(PLATFORMS)))
override DOCKER_ENABLE_BUILDX := $(shell if [[ '$(PLATFORMS)' =~ ^(linux/amd64)?$$ ]];then echo 0; else echo 1; fi)
override DOCKER_BUILD_CMD := $(ENV_DOCKER) $(if $(findstring $(DOCKER_ENABLE_BUILDX),1),buildx) build
override DOCKER_BUILD_LOAD := $(if $(findstring $(DOCKER_ENABLE_BUILDX),1),--load)
override DOCKER_BUILD_PLATFORMS := --platform=$(shell echo $(PLATFORMS)| sed "s/ \+/,/g")
override DOCKER_BUILD_PUSH := $(if $(findstring $(DOCKER_ENABLE_BUILDX),1),--push)

define func_check_params
	$(foreach param, $1, $(if $($(param)),,\
	$(error should set $(param) param, you can set $(param) as environment, or use 'make [target] $(param)=[value]' pattern execute make command)))
endef

override TARS_COMPILER_CONTEXT_DIR := context/compiler
override TARS_COMPILER_DOCKERFILE := context/compiler/Dockerfile
define func_create_compiler
	git submodule update --init --recursive submodule/TarsCpp
	rm -rf $(TARS_COMPILER_CONTEXT_DIR)/root/root/$(TARS_CPP_DIR)
	cp -rf $(PWD)/$(TARS_CPP_DIR) $(TARS_COMPILER_CONTEXT_DIR)/root/root
	$(foreach platform,$(PLATFORMS), \
      $(ENV_DOCKER) build --platform $(platform) $(DOCKER_BUILD_LOAD) -t $(platform)/tarscompiler:$(BUILD_VERSION) --build-arg BUILD_VERSION=$(BUILD_VERSION) $(TARS_COMPILER_CONTEXT_DIR); \
	)
endef

define func_enable_buildx
	docker run --privileged --rm tonistiigi/binfmt --install all
	-docker buildx create --name tars-builx-builder
	docker buildx use tars-builx-builder
endef

### enable.buildx : enable docker buildx for build multi-platform images
.PHONY: enable.buildx
enable.buildx:
	@echo "$@ -> [ Start ]"
	$(call func_enable_buildx)
	@echo "$@ -> [ Done ]"

### disable.buildx : disable docker buildx
.PHONY: disable.buildx
disable.buildx:
	@echo "$@ -> [ Start ]"
	$(ENV_DOCKER) buildx use default
	@echo "$@ -> [ Done ]"

### compiler : build and push compiler image to registry
.PHONY: compiler
compiler:
	@echo "$@ -> [ Start ]"
	@$(call func_check_params, TARS_CPP_DIR)
	$(call func_create_compiler)
	@echo "$@ -> [ Done ]"

define func_build_base
	$(call func_check_params, REGISTRY_URL BUILD_VERSION)
	$(DOCKER_BUILD_CMD) $(DOCKER_BUILD_PLATFORMS) -t $(REGISTRY_URL)/$1:$(BUILD_VERSION) -f $2 --build-arg REGISTRY_URL=$(REGISTRY_URL) --build-arg BUILD_VERSION=$(BUILD_VERSION) $3 $(DOCKER_BUILD_PUSH)
endef

### [base name] : build and push specified base image to registry
override BASES_CONTEXT_DIR := context/bases
.PHONY: %base
%base: $(if $(findstring $(DOCKER_ENABLE_BUILDX),1),enable.buildx,disable.buildx)
	@echo "$@ -> [ Start ]"
	$(call func_build_base,tars.$@, $(BASES_CONTEXT_DIR)/$@.Dockerfile, $(BASES_CONTEXT_DIR))
	@echo "$@ -> [ Done ]"

### base : build and push base images to registry
.PHONY: base
override BASE_SUB_TARGETS :=$(BASES)
base: $(BASE_SUB_TARGETS)
	@echo "$@ -> [ Done ]"

### elasticsearch : build and push elasticsearch image to registry
.PHONY: elasticsearch
elasticsearch:
	@echo "$@ -> [ Start ]"
	@$(call func_build_image,tars.$@,context/$@/Dockerfile, context/$@)
	@echo "$@ -> [ Done ]"

define func_expand_server_param
override $1_repo :=$(shell echo $2 | tr '[:upper:]' '[:lower:]')
endef
$(foreach server, $(CONTROLLER_SERVERS), $(eval $(call func_expand_server_param,$(server),$(server))))
$(foreach server, $(FRAMEWORK_SERVERS), $(eval $(call func_expand_server_param,$(server),tars.$(server))))
define func_build_binary
	$(call func_check_params, REGISTRY_URL BUILD_VERSION)
	@mkdir -p cache/$1/go
	@mkdir -p cache/$1/tarscpp
	$(ENV_DOCKER_RUN) --platform $1 --rm -v $(PWD)/src:/src -v $(PWD)/cache/$1/tarscpp:/tarscpp -v $(PWD)/cache/$1/go:/go $(platform)/tarscompiler:$(BUILD_VERSION) $@
	mkdir -p cache/context/$@/binary
	cp cache/$1/tarscpp/bin/$@ cache/context/$@/binary/$@_$(subst /,_,$1)
endef

define func_build_image
	$(call func_check_params, REGISTRY_URL BUILD_VERSION)
	@$(if $(REGISTRY_USER), @($(ENV_DOCKER) login -u $(REGISTRY_USER) -p $(REGISTRY_PASSWORD) $(REGISTRY_URL:docker.io/%=docker.io)))
	cp -rf $3 cache/context
	$(DOCKER_BUILD_CMD) $(DOCKER_BUILD_PLATFORMS) -t $(REGISTRY_URL)/$1:$(BUILD_VERSION) -f cache/$2 --build-arg BINARY=$@ --build-arg REGISTRY_URL=$(REGISTRY_URL) --build-arg BUILD_VERSION=$(BUILD_VERSION) cache/$(strip $3) $(DOCKER_BUILD_PUSH)
endef

### [server name] : build and push specified server image to registry
.PHONY: tars%
tars%: $(if $(findstring $(DOCKER_ENABLE_BUILDX),1),enable.buildx,disable.buildx) $(if $(findstring $(WITHOUT_DEPENDS_CHECK),1),,compiler cppbase)
	@echo "$@ -> [ Start ]"
	$(foreach platform,$(PLATFORMS), $(call func_build_binary,$(platform)))
	$(call func_build_image,$($@_repo),context/$@/Dockerfile, context/$@)
	@echo "$@ -> [ Done ]"

.PHONY: tarsweb
tarsweb: $(if $(findstring $(DOCKER_ENABLE_BUILDX),1),enable.buildx,disable.buildx)
	@echo "$@ -> [ Start ]"
	git submodule update --init --recursive submodule/TarsWeb
	rm -rf cache/context/tarsweb/root/tars-web
	mkdir -p cache/context/tarsweb/root
	cp -rf $(TARS_WEB_DIR) cache/context/tarsweb/root/tars-web
	$(call func_build_image,tars.tarsweb,context/$@/Dockerfile, context/$@)
	@echo "$@ -> [ Done ]"

### controller : build and push controller servers image to registry
.PHONY: controller $(if $(findstring $(DOCKER_ENABLE_BUILDX),1),enable.buildx,disable.buildx)
override CONTROLLER_SUB_TARGETS :=$(CONTROLLER_SERVERS)
controller: $(CONTROLLER_SUB_TARGETS)
	@echo "$@ -> [ Done ]"

### framework : build and push framework servers image to registry
.PHONY: framework $(if $(findstring $(DOCKER_ENABLE_BUILDX),1),enable.buildx,disable.buildx)
override FRAMEWORK_SUB_TARGETS :=$(FRAMEWORK_SERVERS)
framework: $(FRAMEWORK_SUB_TARGETS)
	@echo "$@ -> [ Done ]"

### chart.controller: build tarscontroller chart
override CONTROLLER_CHART_TEMPLATE_DIR:=helm/tarscontroller
override CONTROLLER_CHART_CACHE_DIR:=cache/helm/tarscontroller
chart.controller: $(if $(findstring $(WITHOUT_DEPENDS_CHECK),1),,controller)
	@echo "$@ -> [ Start ]"
	$(call func_check_params, CHART_VERSION CHART_APPVERSION CHART_DST CRD_SERVED_VERSIONS CRD_STORAGE_VERSION REGISTRY_URL BUILD_VERSION)
	@mkdir -p cache/helm
	@rm -rf $(CONTROLLER_CHART_CACHE_DIR)
	@cp -rf "$(CONTROLLER_CHART_TEMPLATE_DIR)" "$(CONTROLLER_CHART_CACHE_DIR)"
	@./util/render-conroller-chart.sh "$(CONTROLLER_CHART_CACHE_DIR)" "$(CHART_VERSION)" "$(CHART_APPVERSION)" "$(CRD_SERVED_VERSIONS)" "$(CRD_STORAGE_VERSION)" "$(REGISTRY_URL)" "$(BUILD_VERSION)"
	$(ENV_HELM) package -d "$(CHART_DST)" --version "$(CHART_VERSION)" --app-version "$(CHART_APPVERSION)"  "$(CONTROLLER_CHART_CACHE_DIR)"
	@echo "$@ -> [ Done ]"

### chart.framework: build tarsframework chart
override FRAMEWORK_CHART_TEMPLATE_DIR:=helm/tarsframework
override FRAMEWORK_CHART_CACHE_DIR:=cache/helm/tarsframework
chart.framework: $(if $(findstring $(WITHOUT_DEPENDS_CHECK),1),,framework)
	@echo "$@ -> [ Start ]"
	$(call func_check_params, CHART_VERSION CHART_APPVERSION CHART_DST REGISTRY_URL BUILD_VERSION)
	@mkdir -p cache/helm
	@rm -rf $(FRAMEWORK_CHART_CACHE_DIR)
	@cp -rf "$(FRAMEWORK_CHART_TEMPLATE_DIR)" "$(FRAMEWORK_CHART_CACHE_DIR)"
	@./util/render-framework-chart.sh "$(FRAMEWORK_CHART_CACHE_DIR)" "$(CHART_VERSION)" "$(CHART_APPVERSION)" "$(REGISTRY_URL)" "$(BUILD_VERSION)"
	$(ENV_HELM) package -d "$(CHART_DST)" --version "$(CHART_VERSION)" --app-version "$(CHART_APPVERSION)"  "$(FRAMEWORK_CHART_CACHE_DIR)"
	@echo "$@ -> [ Done ]"

### chart : set of make chart.controller, chart.framework
override CHART_SUB_TARGETS :=chart.controller chart.framework
chart: $(CHART_SUB_TARGETS)
	@echo "$@ -> [ Done ]"

### all : set of make base, make controller, make framework, make chart
.PHONY: all
override ALL_SUB_TARGETS :=base controller framework chart
all: $(ALL_SUB_TARGETS)
	@echo "$@ -> [ Done ]"

CONTROLLER_CHART_PARAMS += $(if $(CONTROLLER_REGISTRY), --set controller.registry=$(CONTROLLER_REGISTRY))
CONTROLLER_CHART_PARAMS += $(if $(CONTROLLER_TAG), --set controller.tag=$(CONTROLLER_TAG))
CONTROLLER_CHART_PARAMS += $(if $(CONTROLLER_SECRET), --set controller.secret=$(CONTROLLER_SECRET))

### install.controller : install tarscontroller chart
.PHONY: install.controller
install.controller:
	@echo "$@ -> [ Start ]"
	$(call func_check_params, CHART)
	$(ENV_HELM) install tarscontroller $(CONTROLLER_CHART_PARAMS) $(CHART)
	@echo "$@ -> [ Done ]"

### upgrade.controller : upgrade tarscontroller chart
.PHONY: upgrade.controller
upgrade.controller:
	@echo "$@ -> [ Start ]"
	$(call func_check_params, CHART)
	$(ENV_HELM) upgrade tarscontroller --install $(CONTROLLER_CHART_PARAMS) $(CHART)
	@echo "$@ -> [ Done ]"

FRAMEWORK_CHART_PARAMS += $(if $(FRAMEWORK_REGISTRY), --set framework.registry=$(FRAMEWORK_REGISTRY))
FRAMEWORK_CHART_PARAMS += $(if $(FRAMEWORK_TAG), --set framework.tag=$(FRAMEWORK_TAG))
FRAMEWORK_CHART_PARAMS += $(if $(FRAMEWORK_SECRET), --set framework.secret=$(FRAMEWORK_SECRET))
FRAMEWORK_CHART_PARAMS += $(if $(UPLOAD_REGISTRY), --set upload.registry=$(UPLOAD_REGISTRY))
FRAMEWORK_CHART_PARAMS += $(if $(UPLOAD_SECRET), --set upload.secret=$(UPLOAD_SECRET))
FRAMEWORK_CHART_PARAMS += $(if $(WEB_HOST), --set web=$(WEB_HOST))

### install.framework : install tarsframework chart
.PHONY: install.framework
install.framework:
	@echo "$@ -> [ Start ]"
	$(call func_check_params, CHART NAMESPACE UPLOAD_REGISTRY)
	$(ENV_HELM) install tarsframework -n $(NAMESPACE) --create-namespace $(FRAMEWORK_CHART_PARAMS) $(CHART)
	@echo "$@ -> [ Done ]"

### upgrade.framework : upgrade tarsframework chart
.PHONY: upgrade.framework
upgrade.framework:
	@echo "$@ -> [ Start ]"
	$(call func_check_params, CHART NAMESPACE UPLOAD_REGISTRY)
	$(ENV_HELM) upgrade tarsframework -n $(NAMESPACE) --create-namespace --install $(FRAMEWORK_CHART_PARAMS) $(CHART)
	@echo "$@ -> [ Done ]"

#replace.tars%:
#   fixme
#	@echo "$@ -> [ Start ]"
#	$(call func_check_params, NAMESPACE UPLOAD_REGISTRY)
#	$(ENV_KUBECTL) patch -n $(NAMESPACE) tserver
#	@echo "$@ -> [ Done ]"

### secret : create kubernetes docker-registry secret
.PHONY: secret
secret:
	@echo "$@ -> [ Start ]"
	$(call func_check_params, NAMESPACE NAME REGISTRY_URL REGISTRY_USER REGISTRY_PASSWORD)
	$(ENV_KUBECTL) create secret docker-registry $(NAME) -n $(NAMESPACE) --docker-server=$(REGISTRY_URL) --docker-username=$(REGISTRY_USER) --docker-password=$(REGISTRY_PASSWORD)
	@echo "$@ -> [ Done ]"

### clean : clean up
.PHONY: clean
clean:
	@echo "$@ -> [ Start ]"
	@echo "$@ -> [ Done ]"

### test.controller : controller servers
.PHONY: test.controller
test.controller:
	@echo "$@ -> [ Start ]"
	@$(call func_get_compiler)
	@mkdir -p cache/go
	$(ENV_DOCKER_RUN) --rm -v $(PWD)/src:/src -v $(PWD)/t:/t -v $(HOME)/.kube/config:/root/.kube/config -v $(PWD)/cache/go:/go --net=host $(COMPILER) test.controller
	@echo "$@ -> [ Done ]"

### help : show make rules
.PHONY: help
help:
	@echo
	@echo "Make Rules:"
	@if [ '$(ENV_OS_NAME)' = 'darwin' ]; then \
		awk '{ if(match($$0, /^#{3}([^:]+):(.*)$$/)){ split($$0, res, ":"); gsub(/^#{3}[ ]*/, "", res[1]); _desc=$$0; gsub(/^#{3}([^:]+):[ \t]*/, "", _desc); printf("-- make %-24s : %-10s\n", res[1], _desc) } }' makefile; \
	else \
		awk '{ if(match($$0, /^\s*#{3}\s*([^:]+)\s*:\s*(.*)$$/, res)){ printf("-- make %-24s : %-10s\n", res[1], res[2]) } }' makefile; \
	fi
	@echo
