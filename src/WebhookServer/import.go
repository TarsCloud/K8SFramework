package main

import (
	_ "tarswebhook/webhook/conversion/tars/v1beta2"
	_ "tarswebhook/webhook/conversion/tars/v1beta3"
	_ "tarswebhook/webhook/mutating/tars/v1beta2"
	_ "tarswebhook/webhook/mutating/tars/v1beta3"
	_ "tarswebhook/webhook/validating/apps/v1"
	_ "tarswebhook/webhook/validating/core/v1"
	_ "tarswebhook/webhook/validating/tars/v1beta2"
	_ "tarswebhook/webhook/validating/tars/v1beta3"
)
