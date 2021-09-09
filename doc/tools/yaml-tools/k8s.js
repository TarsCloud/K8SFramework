
// read.js
const fs = require('fs');
const yargs = require('yargs');
const yaml = require('js-yaml');
const k8s = require('@kubernetes/client-node');

//服务发布到k8s, 只修改image地址
//node k8s.js -f values.yaml -n ns -i image -d id
//node k8s.js -f values.yaml -n ns -i image -d id -u

const NAMESPACE = yargs.argv.n;

const kc = new k8s.KubeConfig();
kc.loadFromDefault();

const opts = {};
kc.applyToRequest(opts);

const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);

const GROUP = "k8s.tars.io";
const VERSION = "v1beta1";

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

const main = async () => {
    let contents = fs.readFileSync(yargs.argv.f, 'utf8');

    let data = yaml.load(contents);

    let name = `${data.app}-${data.server}`.toLowerCase();

    let id = yargs.argv.d;
    let image = yargs.argv.i;

    let server = await getObject('tservers', name);

    console.log(server);

    if (server) {

        data.repo.id = id;
        data.repo.image = image;
        // data.repo.secret = secret;

        data.replicas = server.spec.replicas;

        if (yargs.argv.u) {
            fs.writeFileSync(yargs.argv.f, yaml.dump(data));
        }

        console.log(yaml.dump(data));
    }

}
try {
    
    main();

} catch (e) {
    console.error(e);
    process.exit(-1);
}
