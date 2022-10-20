package e2e

import (
	_ "e2e/taccount/v1beta2"
	_ "e2e/taccount/v1beta3"
	_ "e2e/tconfig/v1beta2"
	_ "e2e/tconfig/v1beta3"
	_ "e2e/timage/v1beta2"
	_ "e2e/timage/v1beta3"
	_ "e2e/tserver/v1beta2"
	_ "e2e/tserver/v1beta3"
	_ "e2e/ttemplate/v1beta2"
	_ "e2e/ttemplate/v1beta3"
	_ "e2e/ttree/v1beta2"
	_ "e2e/ttree/v1beta3"
)

func runE2E() {}
