SRC_DIR     := /src
TEST_DIR    := /t
BUILD_DIR   := /build

cmake:
	@cd $(BUILD_DIR) && cmake $(SRC_DIR)

override SERVER_TARGETS := tarscontroller tarswebhook tarsagent tarskaniko tarsAdminRegistry tarsimage tarsnode tarsregistry tarsconfig tarslog tarsnotify tarsstat tarsproperty tarsquerystat tarsqueryproperty tarskevent
define func_expand_server_targets
ifeq ($1,tarskaniko)
$1: export CGO_ENABLED=0
endif
.PHONY: $1
$1 : cmake
ifeq ($1,$(findstring $1,tarsquerystat tarsqueryproperty))
	@cd $(BUILD_DIR) && make tarsqueryserver -j1
	@cp $(BUILD_DIR)/bin/tarsqueryserver $(BUILD_DIR)/bin/$1
else
	@cd $(BUILD_DIR) && make $1 -j1
endif
endef
$(foreach server,$(SERVER_TARGETS),$(eval $(call func_expand_server_targets,$(server))))

.PHONY: e2e.controller
e2e.controller :
	cd $(TEST_DIR)/controller/e2e && go test -c
	mkdir -p $(BUILD_DIR)/bin
	mv $(TEST_DIR)/controller/e2e/e2e.test $(BUILD_DIR)/bin/e2e.controller

.PHONY: e2e.framework
e2e.framework :
	cd $(TEST_DIR)/framework/e2e && go test -c
	mkdir -p $(BUILD_DIR)/bin
	mv $(TEST_DIR)/framework/e2e/e2e.test $(BUILD_DIR)/bin/e2e.framework

clean:
	@rm -rf $(BUILD_DIR)
