var npmPackage = require('../package.json'),
    helpout = require('../lib/index.js');

process.stdout.write(helpout.version(npmPackage));
process.stdout.write(helpout.help({
    npmPackage: npmPackage,
    usage: [
        '<command> [options]',
        '[regular options]',
        '--these --are --just --tests [this is going to be a really long string of text that should wrap onto the next line]'
    ],
    sections: {
        Details: {
            description: 'This is a test description. The content here should be wrapped to the console width and only on word breaks before the end of the line.',
            options: {
                '--test, -t': 'Toggles a test switch. Literally does nothing, but nice to look at when passing a command.',
                '-o': 'Another test switch, but this time a bit more circular and minimal in appearance. All the rage with hipsters.',
                '--a-really-frickin-long-switch': 'Really pushing the limits of what is classified as a valid switch now, are we?'
            }
        },
        'Another Section': {
            description: 'Another test description',
            options: {
                '--more-stuff': 'Just another section of more examples'
            }
        }
    }
}));