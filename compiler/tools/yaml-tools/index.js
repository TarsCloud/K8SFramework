// read.js
const fs = require('fs');
const yargs = require('yargs');
const yaml = require('js-yaml');

//获取或者修复yaml文件中某个值, 获取app, 并转换成小写
//node index.js -f values.yaml -g app
//获取readme属性, 且保持大小写不变
//node index.js -f values.yaml -n -g readme
//node index.js -f values.yaml -s app -v base 
//node index.js -f values.yaml -s app -v base  -u

try {

    let contents = fs.readFileSync(yargs.argv.f, 'utf8');

    let data = yaml.load(contents);

    if (yargs.argv.g) {
        let value = eval(`data.${yargs.argv.g}`);
        if (Array.isArray(value)) {
            console.log(value.join(" "));
        } else if (!value) {
            console.log(``);
        } else {
            if (yargs.argv.n) {
                console.log(value);
            } else {
                console.log(value.toLowerCase());
            }

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