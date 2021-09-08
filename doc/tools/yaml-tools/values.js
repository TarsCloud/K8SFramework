
// read.js
const fs = require('fs');
const yargs = require('yargs');
const yaml = require('js-yaml');

//获取或者修复yaml文件中某个值
//node values.js -f values.yaml -d id -i image -u

try {

    let contents = fs.readFileSync(yargs.argv.f, 'utf8');

    let data = yaml.load(contents);

    data.repo.id = yargs.argv.d;
    data.repo.image = yargs.argv.i;
    data.user = process.env.GITLAB_USER_NAME;
    data.reason = process.env.CI_COMMIT_MESSAGE;

    if (yargs.argv.u) {
        fs.writeFileSync(yargs.argv.f, yaml.dump(data));
    }

} catch (e) {
    console.error(e);
    process.exit(-1);
}

