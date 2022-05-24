.DEFAULT_GOAL := help
SHELL := /bin/bash -o pipefail
ENV_DOCKER             ?= docker
ENV_DOCKER_RUN         ?= $(ENV_DOCKER) run $(if $(findstring $(DOCKER_RUN_WITHOUT_IT),1),,-it)
ENV_HELM               ?= helm
ENV_KUBECTL            ?= kubectl
ENV_OS_NAME            ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')

override PARAMS:= $(shell sh ./params.sh PARAMS)
define func_read_params
$1 ?= $(shell sh ./params.sh $1)
endef
$(foreach param,$(PARAMS),$(eval $(call func_read_params,$(param))))
$(foreach param,$(PARAMS),$(info read [ $(param) ]  =  $($(param))))

define func_check_params
	$(foreach param, $1, $(if $($(param)),,\
	$(error should set $(param) param, you can set $(param) as environment, or use 'make [target] $(param)=[value]' pattern execute make command)))
endef

override TARS_COMPILER_CONTEXT_DIR := context/compiler
override TARS_COMPILER_DOCKERFILE := context/compiler/Dockerfile
define func_create_compiler
	mkdir -p $(TARS_CPP_DIR) && cd $(TARS_CPP_DIR) && git submodule update --init --recursive
	cp -r $(PWD)/$(TARS_CPP_DIR) $(TARS_COMPILER_CONTEXT_DIR)/root/root
	$(ENV_DOCKER) build -t tarscompiler:$(BUILD_VERSION) --build-arg BUILD_VERSION=$(BUILD_VERSION) $(TARS_COMPILER_CONTEXT_DIR)
endef

define func_get_compiler
	@$(call func_check_params,BUILD_VERSION)
	$(eval COMPILER := $(shell $(ENV_DOCKER) images tarscompiler:$(BUILD_VERSION) -q))
endef

define func_build_image
	$(call func_check_params, REGISTRY_URL BUILD_VERSION)
	$(if $(findstring $1,tars.tarsweb),\
		$(call func_check_params, TARS_WEB_DIR) \
		mkdir -p $(TARS_WEB_DIR) && rm -rf $3/root/root/tars-web && cp -r $(PWD)/$(TARS_WEB_DIR) $3/root/root/tars-web && git submodule update --init --recursive \
		&& $(ENV_DOCKER) build -t $(REGISTRY_URL)/$1:$(BUILD_VERSION) -f $2 $3,\
		$(ENV_DOCKER) build -t $(REGISTRY_URL)/$1:$(BUILD_VERSION) -f $2 --build-arg REGISTRY_URL=$(REGISTRY_URL) --build-arg BUILD_VERSION=$(BUILD_VERSION) $3 \
	)
endef

define func_push_image
	$(call func_check_params, REGISTRY_URL BUILD_VERSION)
	$(if $(REGISTRY_USER), @($(ENV_DOCKER) login -u $(REGISTRY_USER) -p $(REGISTRY_PASSWORD) $(REGISTRY_URL:docker.io/%=docker.io)))
	$(ENV_DOCKER) push $(REGISTRY_URL)/$1:$(BUILD_VERSION)
endef

### compiler : build and push compiler image to registry
.PHONY: compiler
compiler:
	@echo "$@ -> [ Start ]"
	@$(call func_check_params, TARS_CPP_DIR)
	@$(call func_create_compiler)
	@echo "$@ -> [ Done ]"

### [base name] : build and push specified base image to registry
override BASES_CONTEXT_DIR := context/bases
.PHONY: %base
%base:
	@echo "$@ -> [ Start ]"
	@$(call func_build_image,tars.$@, $(BASES_CONTEXT_DIR)/$@.Dockerfile, $(BASES_CONTEXT_DIR))
	@$(call func_push_image,tars.$@)
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
	@$(call func_push_image,tars.$@, context/$@)
	@echo "$@ -> [ Done ]"

define func_expand_server_param
override $1_exec :=$2
override $1_dir :=$3
override $1_repo :=$4
endef
$(foreach server, tarscontroller tarsagent, $(eval $(call func_expand_server_param,$(server), $(server), context/$(server)/root/usr/local/app/tars/$(server)/bin,$(server))))
$(foreach server, tarsimage tarsregistry, $(eval $(call func_expand_server_param,$(server), $(server), context/$(server)/root/usr/local/app/tars/$(server)/bin,tars.$(server))))
$(foreach server, tarsconfig tarslog tarsnotify tarsstat tarsproperty tarsquerystat tarsqueryproperty tarskevent, $(eval $(call func_expand_server_param, $(server), $(server), context/$(server)/root/usr/local/server/bin,tars.$(server))))
$(foreach server, tarsnode, $(eval $(call func_expand_server_param, $(server), $(server), context/$(server)/root/tarsnode/bin,tars.$(server))))
$(foreach server, tarskaniko, $(eval $(call func_expand_server_param, $(server), $(server), context/$(server)/root/kaniko,tars.$(server))))
$(foreach server, tarsweb, $(eval $(call func_expand_server_param, $(server), tars2case, context/$(server)/root/root/usr/local/tars/cpp/tools,tars.$(server))))

### [server name] : build and push specified server image to registry
.PHONY: tars%
tars%: $(if $(findstring $(WITHOUT_DEPENDS_CHECK),1),,compiler cppbase)
	@echo "$@ -> [ Start ]"
	@$(call func_get_compiler)
	@mkdir -p cache/go
	@mkdir -p cache/tarscpp
	$(ENV_DOCKER_RUN) --rm -v $(PWD)/src:/src -v $(PWD)/cache/tarscpp:/tarscpp -v $(PWD)/cache/go:/go $(COMPILER) $($@_exec)
	mkdir -p $($@_dir)
	cp cache/tarscpp/bin/$($@_exec) $($@_dir)
	$(call func_build_image,$($@_repo),context/$@/Dockerfile, context/$@)
	$(call func_push_image,$($@_repo), context/$@)
	@echo "$@ -> [ Done ]"

### controller : build and push controller servers image to registry
.PHONY: controller
override CONTROLLER_SUB_TARGETS :=$(CONTROLLER_SERVERS)
controller: $(CONTROLLER_SUB_TARGETS)
	@echo "$@ -> [ Done ]"

### framework : build and push framework servers image to registry
.PHONY: framework
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
	@rm -rf ${CONTROLLER_CHART_CACHE_DIR}
	@cp -r "${CONTROLLER_CHART_TEMPLATE_DIR}" "${CONTROLLER_CHART_CACHE_DIR}"
	@./util/render-conroller-chart.sh "$(CONTROLLER_CHART_CACHE_DIR)" "$(CHART_VERSION)" "$(CHART_APPVERSION)" "${CRD_SERVED_VERSIONS}" "$(CRD_STORAGE_VERSION)" "$(REGISTRY_URL)" "$(BUILD_VERSION)"
	$(ENV_HELM) package -d "$(CHART_DST)" --version "$(CHART_VERSION)" --app-version "$(CHART_APPVERSION)"  "$(CONTROLLER_CHART_CACHE_DIR)"
	@echo "$@ -> [ Done ]"

### chart.framework: build tarsframework chart
override FRAMEWORK_CHART_TEMPLATE_DIR:=helm/tarsframework
override FRAMEWORK_CHART_CACHE_DIR:=cache/helm/tarsframework
chart.framework: $(if $(findstring $(WITHOUT_DEPENDS_CHECK),1),,framework)
	@echo "$@ -> [ Start ]"
	$(call func_check_params, CHART_VERSION CHART_APPVERSION CHART_DST REGISTRY_URL BUILD_VERSION)
	@mkdir -p cache/helm
	@rm -rf ${FRAMEWORK_CHART_CACHE_DIR}
	@cp -r "${FRAMEWORK_CHART_TEMPLATE_DIR}" "${FRAMEWORK_CHART_CACHE_DIR}"
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
