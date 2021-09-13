const fs = require('fs');
const yargs = require('yargs');
const yaml = require('js-yaml');
const k8s = require('@kubernetes/client-node');

//测试用

//node image.js -n tars-dev -a test -s testserver

const NAMESPACE = yargs.argv.n;

const kc = new k8s.KubeConfig();
kc.loadFromDefault();

const opts = {};
kc.applyToRequest(opts);

const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);

const GROUP = "k8s.tars.io";
const VERSION = "v1beta1";
const TImageTypeLabel = "tars.io/ImageType"
const TServerAppLabel = "tars.io/ServerApp"
const TServerNameLabel = "tars.io/ServerName"

const getObject = async (plural, name) => {

    let data = null;
    try {
        data = await k8sApi.getNamespacedCustomObject(GROUP, VERSION, NAMESPACE, plural, name);
    } catch (e) {
        if (e.statusCode == 404) {
            return null;
        }
        throw e;
    }
    return data.body
}

const createObject = async (plural, object) => {
    return await k8sApi.createNamespacedCustomObject(GROUP, VERSION, NAMESPACE, plural, object);
}


const createImage = async (ServerApp, ServerName, namespace) => {

    image = (ServerApp + '-' + ServerName).toLowerCase();

    let result = await getObject("timages", image);

    if (!result) {

        let tImage = {
            apiVersion: GROUP + '/' + VERSION,
            kind: 'TImage',
            metadata: {
                namespace: namespace,
                name: image,
                labels: {}
            },
            imageType: 'server',
            releases: []
        }

        tImage.metadata.labels[`${TImageTypeLabel}`] = 'server';
        tImage.metadata.labels[`${TServerAppLabel}`] = ServerApp;
        tImage.metadata.labels[`${TServerNameLabel}`] = ServerName;

        // console.log(tImage.metadata.labels);

        result = await createObject("timages", tImage);
    }
}

const app = yargs.argv.a;
const server = yargs.argv.s;
const namespace = yargs.argv.n;

createImage(app, server, namespace);