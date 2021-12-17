// read.js
const fs = require('fs');
const yargs = require('yargs');
const yaml = require('js-yaml');

//获取或者修复yaml文件中某个值
//node index.js -f values.yaml -g app
//node index.js -f values.yaml -s app -v base 
//node index.js -f values.yaml -s app -v base  -u

try {

    let contents = fs.readFileSync(yargs.argv.f, 'utf8');

    let data = yaml.load(contents);

    if (yargs.argv.g) {
        let value = eval(`data.${yargs.argv.g}`);
        if (Array.isArray(value)) {
            console.log(value.join(" "));
        } else {
            eval(`console.log(data.${yargs.argv.g}.toLowerCase()`);
        }
    } else if (yargs.argv.s) {
        eval(`data.${yargs.argv.s} = yargs.argv.v`);
        if (yargs.argv.u) {
            fs.writeFileSync(yargs.argv.f, yaml.dump(data));
        }
    }

} catch (e) {
    console.error(e);
    process.exit(-1);
}