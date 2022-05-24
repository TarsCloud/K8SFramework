SRC_DIR     := /src
TEST_DIR    := /t
BUILD_DIR   := /tarscpp

cmake:
	@cd $(BUILD_DIR) && cmake $(SRC_DIR)

override SERVER_TARGETS := tarscontroller tarsagent tarskaniko tarsimage tarsnode tarsregistry tarsconfig tarslog tarsnotify tarsstat tarsproperty tarsquerystat tarsqueryproperty tarskevent
define func_expand_server_targets
ifeq ($1,tarskaniko)
$1: export CGO_ENABLED=0
endif
.PHONY: $1
$1 : cmake
ifeq ($1,$(findstring $1,tarsquerystat tarsqueryproperty))
	@cd $(BUILD_DIR) && make tarsqueryserver -j4
	@cp $(BUILD_DIR)/bin/tarsqueryserver $(BUILD_DIR)/bin/$1
else
	@cd $(BUILD_DIR) && make $1 -j4
endif
endef
$(foreach server,$(SERVER_TARGETS),$(eval $(call func_expand_server_targets,$(server))))

.PHONY: tars2case
tars2case :
	@cp /usr/local/tars/cpp/tools/tars2case $(BUILD_DIR)/bin/tars2case

.PHONY: test.controller
test.controller :
	cd $(TEST_DIR)/controller/e2e && go test --timeout 90m

clean:
	@rm -rf $(BUILD_DIR)
