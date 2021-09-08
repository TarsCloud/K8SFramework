#!/bin/bash

# PUSH=`helm plugin list | grep push | awk '{print $1}'`

# if [ "$PUSH" != "push" ]; then
#     helm plugin install https://github.com/chartmuseum/helm-push
#     helm repo add --username admin --password Upchina@999 upchina-charts http://harbor.12345up.com/chartrepo/charts
# fi


helm package tars-server 
mv tserver*.tgz charts

# helm repo index charts --url http://taf.gitlab.whup.io/taf-deploy-platform/charts

# helm push taf-server/ upchina-charts 

