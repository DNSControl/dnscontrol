{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("none"), SERVICE("bind"),
    A("@", "1.2.3.4"),
);
```
{% endcode %}
