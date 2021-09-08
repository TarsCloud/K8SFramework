'use strict';

var jsHtml = require('./jshtml.js'),
    compile = require('./compile.js');

var cacheEntry = { };

module.exports = {
    script: jsHtml,
    compile: compile,
    cached: function(cache, script, options) {
        if(!options && typeof script === 'object') {
            options = script;
            script = null;
        }

        if(!script) {
            script = cache;
        }

        var c = cacheEntry[cache];
        if(!c) {
            c = cacheEntry[cache] = jsHtml(script, options);
        }
        return c;
    },
    render: function(script, options, callback) {
        if(!callback && options && typeof options === 'function') {
            callback = options;
            options = null;
        }
        return jsHtml(script, options).render(callback);
    }
};

require.extensions['.jshtml'] = function(m, filename) {
    var script = jsHtml();
    script.setScriptFile(filename);
    script.makeFunction();
    script._context.module = m;
    m.exports = script;
};