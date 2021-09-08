'use strict';

var jshint = require('jshint').JSHINT,
    fs = require('fs'),
    path = require('path'),
    colors = require('colors'),
    JSCS = require('jscs');

var argv = require('minimist')(process.argv.slice(2));

var allowFix = argv.fix || argv.f;
var totalFiles = 0;
var totalErrorsInFiles = 0;
var totalFixableErrorsInFiles = 0;
var returnCode = 0;

var jshintConfig = JSON.parse(fs.readFileSync(path.resolve(__dirname, '../.jshintrc')).toString());
var jscsConfig = JSON.parse(fs.readFileSync(path.resolve(__dirname, '../.jscsrc')).toString());

function recursiveCheckDirectory(dir) {
    var dirList = fs.readdirSync(dir);
    for(var i in dirList) {
        if(dirList.hasOwnProperty(i)) {
            var pathname = path.join(dir, dirList[i]);
            if(fs.statSync(pathname).isDirectory()) {
                recursiveCheckDirectory(pathname);
            }
            else if(path.extname(pathname).toLowerCase() === '.js') {
                var fileContents = fs.readFileSync(pathname).toString();
                var totalErrors = 0;

                jshint(fileContents, jshintConfig);
                totalErrors += jshint.errors.length;

                var jscsChecker = new JSCS();
                jscsChecker.registerDefaultRules();
                jscsChecker.configure(jscsConfig);
                var jscsErrors = jscsChecker.checkString(fileContents).getErrorList();
                totalErrors += jscsErrors.length;

                var jscsFixableErrors = 0;
                for(var o in jscsErrors) {
                    if(jscsErrors.hasOwnProperty(o) && jscsErrors[o].fixed) {
                        jscsFixableErrors++;
                        totalFixableErrorsInFiles++;
                    }
                }

                if(allowFix) {
                    totalErrors -= jscsFixableErrors;
                }

                var success = totalErrors === 0;
                totalErrorsInFiles += totalErrors;

                var printColor = success ? colors.green : colors.red;
                console.log(printColor('[' + (success ? 'PASS' : 'FAIL') + '] ') + path.relative(process.cwd(), pathname));

                totalFiles++;
                if(!success || jscsFixableErrors > 0) {
                    console.log('  ' + totalErrors + ' error' + (totalErrors === 1 ? '' : 's'));

                    if(jscsFixableErrors > 0) {
                        console.log('  ' + jscsFixableErrors + ' fixable error' + (jscsFixableErrors === 1 ? '' : 's'));
                    }

                    for(var n in jshint.errors) {
                        if(jshint.errors.hasOwnProperty(n)) {
                            console.log(colors.red('    code  [' + jshint.errors[n].line + ', ' + jshint.errors[n].character + ']: ' + jshint.errors[n].reason));
                        }
                    }

                    for(var p in jscsErrors) {
                        if(jscsErrors.hasOwnProperty(p)) {
                            console.log((jscsErrors[p].fixed && allowFix ? colors.green : colors.yellow)('    style [' + jscsErrors[p].line + ', ' + jscsErrors[p].column + ']: ' + jscsErrors[p].message + (jscsErrors[p].fixed ? ' (fix' + (allowFix ? 'ed' : 'able') + ')' : '')));
                        }
                    }

                    if(jscsFixableErrors > 0 && allowFix) {
                        fs.writeFileSync(pathname, jscsChecker.fixString(fileContents).output);
                    }

                    if(!success) {
                        returnCode++;
                    }
                }
            }
        }
    }
}

recursiveCheckDirectory(path.resolve(__dirname, '../lib'));
recursiveCheckDirectory(path.resolve(__dirname, '../test'));

console.log('\n' + (totalFiles - returnCode) + '/' + totalFiles + ' files passed (' + totalErrorsInFiles + ' error' + (totalErrorsInFiles === 1 ? '' : 's') + ', ' + totalFixableErrorsInFiles + ' fix' + (allowFix ? 'ed' : 'able') + ')');

process.exit(allowFix ? 0 : returnCode);