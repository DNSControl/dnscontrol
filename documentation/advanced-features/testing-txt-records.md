# TXT record testing

We recently discovered a strange bug with processing TXT records and double-quotes. Sadly we haven't been able to determine a way to test this automatically. Therefore, I've written up this methodology.

# The problem

The problem relates to TXT records that have a string with quotes in them.

If a user creates a TXT record whose contents are `"something"` (yes, with double quotes), some APIs get confused.

This bug is most likely to appear in a provider that uses `RecordConfig.PopulateFromString()` (see `models/t_parse.go`) to create TXT records. That function assumes the string should always have the quotes stripped, though it is more likely that the string should be taken verbatim.

# The test

To complete this test, you will need a test domain that you can add records to. It won't be modified otherwise.

This bug has to do with double-quotes at the start and end of TXT records. If your provider doesn't permit double-quotes in TXT records (you'd be surprised!) you don't have to worry about this bug because those records are banned in your `auditrecords.go` file.

## Step 1: Create the test records

Log into your DNS provider's web UI (portal) and create these 4 TXT records. (Don't use DNSControl!) Yes, include the double-quotes on test 2 and 3!

| Hostname      | TXT       |
|---------------|-----------|
| t0            | test0     |
| t1            | test1     |
| t2            | "test2"   |
| t3            | "test3"   |

## Step 2: Update `dnsconfig.js`

Now in your `dnsconfig.js` file, add these records to the domain:

    TXT("t0", "test0"),
    TXT("t1", "\"test1\""),
    TXT("t2", "test2"),
    TXT("t3", "\"test3\""),

## Step 3: Preview

When you do a `dnscontrol preview`, you should see changes for t1 and t2.

```text
#1: MODIFY TXT t1.example.com: ("test1" ttl=1) -> ("\"test1\"" ttl=1)
#2: MODIFY TXT t2.example.com: ("\"test2\"" ttl=1) -> ("test2" ttl=1)
```

If you don't see those changes, that's a bug. For example, we found that Cloudflare left t2 alone but would try to add double-quotes to t3! This was fixed in [PR#1543](https://github.com/DNSControl/dnscontrol/pull/1543).

## Step 4: Push

Let's assume you DO see the changes. Push them using `dnscontrol push` then check the webui to see that the changes are correct.

```text
2 corrections
#1: MODIFY TXT t1.stackoverflow.help: ("test1" ttl=1) -> ("\"test1\"" ttl=1)
SUCCESS!
#2: MODIFY TXT t2.stackoverflow.help: ("\"test2\"" ttl=1) -> ("test2" ttl=1)
SUCCESS!
```

Refresh your provider's web UI and you should see the changes as expected: t1 should have double-quotes and t2 shouldn't. If the change wasn't correctly done, that's a bug.

## Step 5: That's it!

Remove the lines from `dnsconfig.js` and run `dnscontrol push` to clean up.
