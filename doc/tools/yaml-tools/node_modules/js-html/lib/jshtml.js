'use strict';

var vm = require('vm'),
    fs = require('fs'),
    compile = require('./compile.js'),
    acorn = require('acorn'),
    escodegen = require('escodegen'),
    esmangle = require('esmangle');

var createContextObject = require('./context.js'),
    defaultOptions = {
        syntaxCheck: true,
        format: false,
        mangle: false,
        optimize: false,
        minify: false,
        isolate: true
    };

function JsHtml(script, options) {
    if(!options && typeof script !== 'string') {
        options = script;
    }
    else if(script) {
        this.setScript(script);
    }

    this.setOptions(options);
}

JsHtml.prototype.setOptions = function(options) {
    if(this._options && (options.isolate || false) !== (this._options.isolate || false)) {
        delete this._context; // Execution context will need to be recreated
    }

    this._options = options = (function recurse(opts, def) {
        if(opts == null) {
            opts = { };
        }
        for(var entry in def) {
            if(def.hasOwnProperty(entry) && !opts.hasOwnProperty(entry)) {
                opts[entry] = def[entry];
            }
        }
        return opts;
    })(options, defaultOptions);

    if(options.minify) {
        options.mangle = true;
        options._escodegen = {
            format: {
                compact: true,
                semicolons: false,
                parentheses: false
            }
        };
    }

    if(options.mangle || options.optimize) {
        options.format = true;
    }

    if(options.format) {
        options.syntaxCheck = true;
    }

    if(options.syntaxCheck) {
        options._acorn = {
            allowReturnOutsideFunction: true,
            sourceType: 'module'
        };
    }
};

JsHtml.prototype.setScript = function(script) {
    this.clear();
    this._script = script;
    this._sourceCompiled = false;
};

JsHtml.prototype.setScriptFile = function(filepath) {
    this.setScript(fs.readFileSync(filepath).toString());
    if(!this._options.filename) {
        this._options.filename = filepath;
    }
};

JsHtml.prototype.clear = function() {
    this._script = this._sourceCompiled = this._function =
        this._context = undefined;
};

JsHtml.prototype.compile = function() {
    var script = this._script;
    if(!this._sourceCompiled && script) {
        script = compile(script);

        if(this._options.syntaxCheck) {
            var ast = acorn.parse(script, this._options._acorn);
            if(this._options.optimize) {
                ast = esmangle.optimize(ast, null);
            }
            if(this._options.mangle) {
                ast = esmangle.mangle(ast);
            }
            if(this._options.format) {
                script = escodegen.generate(ast, this._options._escodegen);
            }
        }

        this._script = script;
        this._sourceCompiled = true;
    }
    return script;
};

JsHtml.prototype._makeContext = function() {
    var context = this._context;
    if(!context) {
        var isolate = this._options.isolate;
        context = createContextObject(isolate);

        var additionalContext = this._options.context;
        if(additionalContext) {
            for(var property in additionalContext) {
                if(additionalContext.hasOwnProperty(property)) {
                    context[property] = additionalContext[property];
                }
            }
        }

        if(isolate) {
            context = vm.createContext(context);
        }

        this._context = context;
    }
    return context;
};

JsHtml.prototype.makeFunction = function() {
    var func = this._function;
    if(!func) {
        var script = this.compile();
        if(typeof script === 'string') {
            var context = this._makeContext(),
                exec;
            if(this._options.isolate) {
                exec = vm.runInContext('(function(){' + script + '});', context);
                context.__init_script(this._options.filename || __filename);

                this._function = func = function(thisObject, callback) {
                    if(typeof thisObject === 'function') {
                        callback = thisObject;
                        thisObject = null;
                    }

                    if(callback) {
                        context.__complete = function() {
                            callback(context.__render_end());
                        };
                        exec.call(thisObject);
                    }
                    else {
                        exec.call(thisObject);
                        return context.__render_end();
                    }
                };
            }
            else {
                var genScript = '(function(';
                var args = [ ];
                if(context) {
                    var first = true;
                    for(var property in context) {
                        if(context.hasOwnProperty(property)) {
                            genScript += (first ? '' : ',') + property;
                            args.push(context[property]);
                            first = false;
                        }
                    }
                }
                genScript += '){' + script + '});';
                exec = vm.runInThisContext(genScript);
                context.__init_script(this._options.filename || __filename);

                this._function = func = function(thisObject, callback) {
                    if(typeof thisObject === 'function') {
                        callback = thisObject;
                        thisObject = null;
                    }

                    if(callback) {
                        context.__complete = function() {
                            callback(context.__render_end());
                        };
                        exec.apply(thisObject, args);
                    }
                    else {
                        exec.apply(thisObject, args);
                        return context.__render_end();
                    }
                };
            }
        }
    }
    return func;
};

JsHtml.prototype.render = function(callback) {
    var func = this.makeFunction(),
        ret;
    if(func) {
        ret = func(callback);
    }
    return ret;
};

module.exports = function(script, options) {
    return new JsHtml(script, options);
};