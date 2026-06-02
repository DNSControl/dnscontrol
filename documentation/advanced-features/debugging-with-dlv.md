## Debugger

### Debug a particular function:

```shell
dlv test github.com/DNSControl/dnscontrol/v4/pkg/diff2 -- -test.run Test_analyzeByRecordSet
                                                ^^^^^^^^^
                                                Assumes you are in the pkg/diff2 directory.
```

### Debug an integration tests:

```shell
dlv test github.com/DNSControl/dnscontrol/v4/integrationTest -- -test.v -test.run ^TestDNSProviders -verbose -profile BIND -start 7 -end 7
```

If you are using VSCode, the equivalent configuration is:

```
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

### Debug the `dnscontrol` command

```shell
dlv debug --wd /path/to/config/dir -- preview --domains examples.com
```

VSCode equivalent configuration is:

```
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

