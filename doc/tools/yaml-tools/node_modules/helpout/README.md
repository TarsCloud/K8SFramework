# Format Help Output

Created to save time when it comes to formatting help documentation. The output is automatically adjusted to fit the users console buffer size.

Help output is usually the first place most users look for quick overviews of program functionality and this was created to `helpout` with adding and updating new help documentation.

####Example
```javascript
var npmPackage = require('../package.json'),
    helpout = require('../lib/index.js');

process.stdout.write(helpout.version(npmPackage));
process.stdout.write(helpout.help({
    npmPackage: npmPackage,
    usage: [ // Can be either a string or an array of strings
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
        }
    }
}));
```

####Example Output
```text
helpout 0.1.1
Written by Bryan Way <bryanwayb@gmail.com>

Usage: helpout <command> [options]
       helpout [regular options]
       helpout --these --are --just --tests [this is going to be a really long
           string of text that should wrap onto the next line]

Details
  This is a test description. The content here should be wrapped to the console
  width and only on word breaks before the end of the line.

    --test, -t                      Toggles a test switch. Literally does
                                    nothing, but nice to look at when passing a
                                    command.
    -o                              Another test switch, but this time a bit
                                    more circular and minimal in appearance. All
                                    the rage with hipsters.
    --a-really-frickin-long-switch  Really pushing the limits of what is
                                    classified as a valid switch now, are we?
```
Output specifics will vary based on your console/terminal emulator width. The above output was created with a 80 character width buffer.

####Install

Install into your project via NPM
```bash
npm install helpout
```