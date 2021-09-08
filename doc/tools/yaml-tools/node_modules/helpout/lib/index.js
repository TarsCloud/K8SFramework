'use strict';

function makeWrapLine(str, newlineIndent, strOffset) {
    var ret = '',
        width = process.stdout.columns,
        shouldIndent = false,
        indentArray = new Array(newlineIndent + 1).join(' ');

    if(!strOffset) {
        strOffset = 0;
    }

    while(str.length) {
        if(shouldIndent) {
            ret += indentArray;
        }

        var newWidth = width - (shouldIndent ? indentArray.length : 0);
        if(str.length > newWidth) {
            var nextWordBreak = str.lastIndexOf(' ', newWidth);

            ret += str.slice(0, nextWordBreak) + (newWidth === nextWordBreak ? '' : '\n');
            str = str.substring(nextWordBreak + 1);
        }
        else {
            ret += str;
            break;
        }

        shouldIndent = true;
    }

    return ret;
}

module.exports = {
    help: function(options) {
        options = options || { };
        var ret = '';

        var npmPackage = options.npmPackage;
        var packageName;
        if(npmPackage) {
            packageName = npmPackage.name;
        }

        if(!packageName) {
            packageName = process.argv[1];
        }

        if(options.usage) {
            var usage = options.usage;
            if(!(usage instanceof Array)) {
                usage = [ usage ];
            }

            var usageString = 'Usage: ',
                usageLen = usageString.length,
                len = usage.length,
                indent = new Array(usageLen + 1).join(' ');

            for(var i = 0; i < len; i++) {
                ret += makeWrapLine((i === 0 ? usageString : indent) + packageName + ' ' + usage[i], indent.length + 4) + '\n';
            }
        }

        if(options.sections) {
            var sections = options.sections;
            for(var section in options.sections) {
                if(sections.hasOwnProperty(section)) {
                    var entry = sections[section];

                    ret += '\n' + section + '\n';
                    if(entry.description) {
                        ret += makeWrapLine('  ' + entry.description, 2) + '\n';
                    }

                    if(entry.options) {
                        ret += '\n';
                        var maxNameLength = 0,
                            optName;
                        for(optName in entry.options) { // First pass, get max name length
                            if(entry.options.hasOwnProperty(optName)) {
                                var nameLen = optName.length;
                                if(maxNameLength < nameLen) {
                                    maxNameLength = nameLen;
                                }
                            }
                        }

                        for(optName in entry.options) { // Now do the processing
                            if(entry.options.hasOwnProperty(optName)) {
                                ret += makeWrapLine('    ' + optName + (new Array(maxNameLength - optName.length + 3).join(' ')) + entry.options[optName], maxNameLength + 6) + '\n';
                            }
                        }
                    }
                }
            }
        }
        
        ret += '\n';

        return ret;
    },
    version: function(npmPackage) {
        if(!npmPackage) {
            throw new Error('package.json object is required');
        }

        var ret = makeWrapLine(npmPackage.name + ' ' + npmPackage.version, 4) + '\n';

        var writtenBy = 'Written by ';

        var authors = npmPackage.contributors || [ ];
        if(npmPackage.author) {
            authors.push(npmPackage.author);
        }

        var len = authors.length;
        for(var i = 0; i < len; i++) {
            var author = authors[i];
            if(author.name) {
                writtenBy += (i > 0 ? ', ' : '') + author.name;
                if(author.email) {
                    writtenBy += ' <' + author.email + '>';
                }
            }
            else if(author.email) {
                writtenBy += (i > 0 ? ', ' : '') + author.email;
            }
        }
        ret += makeWrapLine(writtenBy, 0) + '\n\n';

        return ret;
    }
};