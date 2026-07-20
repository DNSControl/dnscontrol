# Short version

You received this error message:

```text
 1: ERROR: target (ghs.googlehosted.com) includes a (.), must end with a (.)
```

This means you should add a "." to the end of the target.

```diff
-   CNAME("foo", "ghs.googlehosted.com"),
+   CNAME("foo", "ghs.googlehosted.com."),
```

# Why CNAME/MX/NS targets require a trailing "dot"

People are often confused about this error message:

```text
 1: ERROR: target (ghs.googlehosted.com) includes a (.), must end with a (.)
```

What this means is that CNAME/MX/NS records (anything where the "target" is a hostname) must end with a "." to indicate that it is a FQDN. The exception to this is that if it is simply a "short name" (i.e. no dots) then DNSControl will add the domain to it.

Here are four examples:

{% code title="dnsconfig.js" %}
```javascript
    CNAME("foo", "bar")       // Permitted. (expands to bar.$DOMAIN)
    CNAME("foo", "bar.com.")  // Permitted. (we are certain what the user wants)
    CNAME("foo", "bar.com")   // ERROR (ambiguous)
    CNAME("foo", "meta.xyz")  // ERROR (ambiguous)
```
{% endcode %}

The first 2 examples are permitted. The last 2 examples are ambiguous and are therefore are considered errors.

How are they ambiguous?

 * Should $DOMAIN be added to "bar.com"? Well, obviously not, because it already ends with ".com" and we all know that "bar.com.bar.com" is probably not what they want. No, it isn't that obvious! Why? (see the next bullet point)
 * Should $DOMAIN be added to "meta.xyz"? Everyone knows that ".xyz" isn't a TLD. Obviously, yes, $DOMAIN should be appended. However, wait... ".xyz" became a TLD in June 2014. We don't want to be surprised by changes like that. Also, users should not be required to memorize all the TLDs. (In the old days it was reasonable to expect people to memorize the 7 TLDs (gov/edu/com/mil/org/net) but since 2000 that's all changed. By the way, we forgot to include "int" in the original and you didn't notice.)
 * What if the CNAME target is "www.bar.com" and the domain is "bar.com"? Then It is reasonable to infer the user's intent, right? `www.bar.com.bar.com.` would be silly, right? Maybe. What if we are copying 100 lines of `dnsconfig.js` from one `D()` to another. Buried in the middle is this one CNAME that means something entirely different when in a new $DOMAIN. That would be bad. We've seen this in production and want to prevent this kind of error.

Yes, we could layer rule upon rule upon rule. Eventually we'd get all the rules right. However, now a user would have to know all the rules to be able to use DNSControl. The point of the DNSControl DSL is to enable the casual user to be able to make DNS updates. By "casual user" we do not mean someone someone that lives and breathes DNS like you and I do. In fact, we mean someone that hasn't memorized the list of rules.

We know of no time where a human intentionally wanted "foo.example.com.domain.com" as the target of an MX record. In fact, the opposite is true. StackExchange.com had a big email outage in 2013 because MX records were updated and the "trailing dot" was forgotten. Our MX records became "aspmx.l.google.com.stackexchange.com" and due to a high TTL we lost email for a few hours. Recently (2017) we had a similar problem and it delayed a new service from working. Luckily this was a new service and didn't have existing users so the problem was unnoticed except for the fact that a project schedule slipped by 3 days.

Therefore, we prefer the rule to be "when in doubt, error out". It is less to remember and catches errors. It also doesn't remove the expressiveness of the language. One dot is better than 100 rules.

## Simple mental models are better

SRE ... the R stands for reliability.

A big source of human error is mental-model mismatch. That is, when operating a complex system, the user has a mental model of what is going on in the system. They are, essentially, emulating the software in their head to predict that the change they are making will have the result they seek. The more complex the system the less likely the mental model will match reality.

A mental model mismatch leads to confusion, frustration, and more importantly it increases the risk of operating error the creates production problems.

If the rules are simple, the mental model will be more accurate than if it is complex. "If something is ambiguous, we give an error and tell you to add a dot to the end" is *simple.* "If something is ambiguous, we follow this list of 100 rules that decide what the user had intended" is *complex.*

One could argue that your users are very smart and can memorize all the rules. Why should they have to? It's just a single keystroke!

## Future

We welcome proposals for how to resolve this ambiguity.

["Future proofing is not adding stuff. Future proofing is making sure you can easily add code/features without breaking existing functionality."](http://softwareengineering.stackexchange.com/a/79591/116123) By not solving the problem now, we open the door to upwards compatible solutions. If we created a partial solution now, we might prevent future solutions from being upward compatible. By simply giving an error we open the door to new solutions.

We should warn you, however, that any new proposals should be simpler than "add a dot".
