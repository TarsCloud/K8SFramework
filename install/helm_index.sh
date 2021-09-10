
#!/bin/bash

helm package tarscontroller
mv tarscontroller*.tgz ../charts

helm package tarsframework
mv tarsframework*.tgz ../charts

helm repo index ../charts --url https://tarscloud.github.io/K8SFramework/charts/
