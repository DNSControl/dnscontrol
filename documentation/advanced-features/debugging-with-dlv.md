# Debugging Tips

- [Debugger](#debugger)
  - [Debug a particular function](#debug-a-particular-function)
  - [Debug an integration tests](#debug-an-integration-tests)
  - [Debug the `dnscontrol` command](#debug-the-dnscontrol-command)

## Debug a particular function

```shell
dlv test github.com/DNSControl/dnscontrol/v4/pkg/diff2 -- -test.run Test_analyzeByRecordSet
                                                ^^^^^^^^^
                                                Assumes you are in the pkg/diff2 directory.
```

## Debug an integration tests

```shell
dlv test github.com/DNSControl/dnscontrol/v4/integrationTest -- -test.v -test.run ^TestDNSProviders -verbose -profile BIND -start 7 -end 7
```

If you are using VSCode, the equivalent configuration is:

```json
    "configurations": [

        {
            "name": "Debug Integration Test",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "${workspaceFolder}/integrationTest",
            "cwd": "${workspaceFolder}/integrationTest",
            "envFile": "${workspaceFolder}/integrationTest/.env",
            "args": [
                "-test.v",
                "-test.run",
                "^TestDNSProviders",
                "-verbose",
                "-profile", "BIND",
                "-start", "7",
                "-end", "7"
            ],
            "buildFlags": "",
            "env": {},
            "showLog": false,
            "console": "integratedTerminal",
            "internalConsoleOptions": "neverOpen"
        }

    ]
}

```

## Debug the `dnscontrol` command

```shell
dlv debug --wd /path/to/config/dir -- preview --domains examples.com
```

VSCode equivalent configuration is:

```json
    "configurations": [

        {
            "name": "preview example.com",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/",
            "cwd": "/path/to/config/dir",
            "args": [
                "preview",
                "--domains",
                "example.com"
            ]
        }

    ]
```

## Debug `helpers.js`

Develop a function:

```
node -e "
function IP(dot) {
    var d = dot.split('.');
    return ((((((+d[0]) * 256) + (+d[1])) * 256) + (+d[2])) * 256) + (+d[3]);
}
console.log(IP('135.181.247.240'));
"
```

Debug a function within helpers.js:

```
$ node -e "
const fs = require('fs');
const vm = require('vm');
const code = fs.readFileSync('/Users/tlimoncelli/gitthings/dnscontrol/pkg/js/helpers.js', 'utf8');
const sandbox = {};
vm.createContext(sandbox);
vm.runInContext(code, sandbox);
console.log(vm.runInContext(\" IP('135.181.247.240') \", sandbox));
"
```
