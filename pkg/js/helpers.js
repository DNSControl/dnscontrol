'use strict';

// How to keep this file clean:
// 1. Add new functions in alphabetical order when it makes sense.
// 2. Run [prettier](https://github.com/prettier/prettier) on the file to ensure
//    your code conforms to our coding standard:
//        npm install prettier
//        node_modules/.bin/prettier --write pkg/js/helpers.js

// Development tip:
// This file is embeded in the binary via "go build". If you are
// debugging/developing this code, it may be faster to specify the
// -dev file to have helpers.js read from the file instead.

// If this javascript interpreter doesn't have a .endsWith() function on strings, add one.
if (typeof String.prototype.endsWith !== 'function') {
    String.prototype.endsWith = function (suffix) {
        return this.indexOf(suffix, this.length - suffix.length) !== -1;
    };
}

var conf = {
    registrars: [],
    dns_providers: [],
    domains: [],
    domain_names: [],
};

var defaultArgs = [];

function initialize() {
    conf = {
        registrars: [],
        dns_providers: [],
        domains: [],
    };
    defaultArgs = [];
}

function _isDomain(d) {
    return (
        _.isArray(d.nameservers) &&
        _.isArray(d.rawrecords) &&
        _.isString(d.name)
    );
}

// Returns an array of domains which were registered so far during runtime
// Own function for compatibility reasons or if some day special processing would be required.
function getConfiguredDomains() {
    return conf.domain_names;
}

// NewRegistrar returns an registrar object.
// For backwards compatibility, it accepts (name), (name, meta),
// (name, type), (name, type, meta).
function NewRegistrar() {
    // For backwards compatibility, this is a wrapper around the legacy
    // version of this function.
    switch (arguments.length) {
        case 1:
            return oldNewRegistrar(arguments[0], '-');
        case 2:
            // x = NewRegistrar("myThing", "THING")
            // x = NewRegistrar("myThing", { metakey: metavalue } )
            if (typeof arguments[1] === 'object') {
                return oldNewRegistrar(arguments[0], '-', arguments[1]);
            }
            break;
        default: // do nothing
    }
    return oldNewRegistrar.apply(null, arguments);
}
function oldNewRegistrar(name, type, meta) {
    if (type) {
        type == 'MANUAL';
    }
    var reg = { name: name, type: type, meta: meta };
    conf.registrars.push(reg);
    return name;
}

function NewDnsProvider(name, type, meta) {
    // For backwards compatibility, this is a wrapper around the legacy
    // version of this function.
    switch (arguments.length) {
        case 1:
            return oldNewDnsProvider(arguments[0], '-');
        case 2:
            // x = NewDnsProvider("myThing", "THING")
            // x = NewDnsProvider("myThing", { metakey: metavalue } )
            if (typeof arguments[1] === 'object') {
                return oldNewDnsProvider(arguments[0], '-', arguments[1]);
            }
            break;
        default: // do nothing
    }
    return oldNewDnsProvider.apply(null, arguments);
}
function oldNewDnsProvider(name, type, meta) {
    if (typeof meta === 'object' && 'ip_conversions' in meta) {
        meta.ip_conversions = format_tt(meta.ip_conversions);
    }
    var dsp = { name: name, type: type, meta: meta };
    conf.dns_providers.push(dsp);
    return name;
}

function newDomain(name, registrar) {
    return {
        name: name,
        subdomain: '',
        registrar: registrar,
        meta: {},
        rawrecords: [],
        recordsabsent: [],
        dnsProviders: {},
        defaultTTL: 0,
        nameservers: [],
        unmanaged: [],
    };
}

function processDargs(m, domain) {
    // for each modifier, if it is a...
    // function: call it with domain
    // array: process recursively
    // object: merge it into metadata
    if (_.isFunction(m)) {
        m(domain);
    } else if (_.isArray(m)) {
        for (var j in m) {
            processDargs(m[j], domain);
        }
    } else if (_.isObject(m)) {
        _.extend(domain.meta, m);
    } else {
        throw (
            'WARNING: domain modifier type unsupported: ' +
            typeof m +
            ' Domain: ' +
            domain.name
        );
    }
}

// D(name,registrar): Create a DNS Domain. Use the parameters as records and mods.
function D(name, registrar) {
    var domain = newDomain(name, registrar);
    for (var i = 0; i < defaultArgs.length; i++) {
        processDargs(defaultArgs[i], domain);
    }
    for (var i = 2; i < arguments.length; i++) {
        var m = arguments[i];
        processDargs(m, domain);
    }

    // handle the empty tag ("example.com!" -> "example.com")
    // replace name with result of removing the empty tag if it exists
    // keep track so we can explain the situation in the error message
    var withoutEmptyTag = _removeEmptyTag(name);
    name = withoutEmptyTag[0];
    var tagWasRemoved = withoutEmptyTag[1];

    if (conf.domain_names.indexOf(name) !== -1) {
        var message = name + ' is declared more than once';
        if (tagWasRemoved) {
            message += ' (check empty tags)';
        }
        throw message;
    }
    conf.domains.push(domain);
    conf.domain_names.push(name);
}

function INCLUDE(name) {
    var domain = _getDomainObject(name);
    if (domain == null) {
        throw (
            name +
            ' was not declared yet and therefore cannot be updated. Use D() before.'
        );
    }
    return function (d) {
        // New-style record types live in rawrecords (processed in Go), so they
        // must be copied too. Each domain re-serializes these objects to its own
        // IR, so sharing the references here is safe.
        d.rawrecords.push.apply(d.rawrecords, domain.obj.rawrecords);
    };
}

// D_EXTEND(name): Update a DNS Domain already added with D(), or subdomain thereof
function D_EXTEND(name) {
    var domain = _getDomainObject(name);
    if (domain == null) {
        throw (
            name +
            ' was not declared yet and therefore cannot be updated. Use D() before.'
        );
    }

    // Handle weird REV() case.
    if (name.indexOf('/') !== -1) {
        name = name.substring(name.indexOf('.') + 1);
    }

    domain.obj.subdomain = name.substr(
        0,
        name.length - domain.obj.name.length - 1
    );

    for (var i = 1; i < arguments.length; i++) {
        var m = arguments[i];
        processDargs(m, domain.obj);
    }
    conf.domains[domain.id] = domain.obj; // let's overwrite the object.
}

// _removeEmptyTag(domain): Remove empty tag.
function _removeEmptyTag(name) {
    var tagWasRemoved = false;
    if (name.slice(-1) === '!') {
        name = name.slice(0, name.length - 1);
        tagWasRemoved = true;
    }
    return [name, tagWasRemoved];
}

// _getDomainObject(name): This implements the domain matching
// algorithm used by D_EXTEND(). Candidate matches are an exact match
// of the domain's name, or if name is a proper subdomain of the
// domain's name. The longest match is returned.
function _getDomainObject(name) {
    var nameTrimmedTag = _removeEmptyTag(name);
    name = nameTrimmedTag[0];
    var domain = null;
    var domainLen = 0;
    for (var i = 0; i < conf.domains.length; i++) {
        var thisName = conf.domains[i]['name'];
        // check for empty tag
        var thisNameTrimmedTag = _removeEmptyTag(thisName);
        thisName = thisNameTrimmedTag[0];

        var desiredSuffix = '.' + thisName;
        var foundSuffix = name.substr(-desiredSuffix.length);
        // If this is an exact match or the suffix matches...
        if (name === thisName || foundSuffix === desiredSuffix) {
            // If this match is a longer match than our current best match...
            if (thisName.length > domainLen) {
                domainLen = thisName.length;
                domain = { id: i, obj: conf.domains[i] };
            }
        }
    }
    return domain;
}

// DEFAULTS provides a set of default arguments to apply to all future domains.
// Each call to DEFAULTS will clear any previous values set.
function DEFAULTS() {
    defaultArgs = [];
    for (var i = 0; i < arguments.length; i++) {
        defaultArgs.push(arguments[i]);
    }
}

// TTL(v): Set the TTL for a DNS record.
function TTL(v) {
    if (_.isString(v)) {
        v = stringToDuration(v);
    }
    return function (r) {
        r.ttl = v;
    };
}

function stringToDuration(v) {
    var matches = v.match(/^(\d+)([smhdwny]?)$/);
    if (matches == null) {
        throw v + ' is not a valid duration string';
    }
    unit = 's';
    if (matches[2]) {
        unit = matches[2];
    }
    v = parseInt(matches[1]);
    var u = { s: 1, m: 60, h: 3600 };
    u['d'] = u.h * 24;
    u['w'] = u.d * 7;
    u['n'] = u.d * 30;
    u['y'] = u.d * 365;
    v *= u[unit];
    return v;
}

// DefaultTTL(v): Set the default TTL for the domain.
function DefaultTTL(v) {
    if (_.isString(v)) {
        v = stringToDuration(v);
    }
    return function (d) {
        d.defaultTTL = v;
    };
}

function makeCAAFlag(value) {
    return function (record) {
        record.caaflag |= value;
    };
}

// CAA_CRITICAL: Critical CAA flag
var CAA_CRITICAL = makeCAAFlag(1 << 7);

// caaOptions injects the CAA flag as the first rdata argument so that MakeCAA
// (Go) always receives (flag, tag, value). processedArgs is [label, tag, value];
// the flag is accumulated on record.caaflag by the CAA_CRITICAL modifier and
// defaults to 0.
function caaOptions(record, processedArgs) {
    return [
        processedArgs[0],
        record.caaflag || 0,
        processedArgs[1],
        processedArgs[2],
    ];
}

// DnsProvider("providerName", 0)
// nsCount of 0 means don't use or register any nameservers.
// nsCount not provider means use all.
function DnsProvider(name, nsCount) {
    if (typeof nsCount === 'undefined') {
        nsCount = -1;
    }
    return function (d) {
        d.dnsProviders[name] = nsCount;
    };
}

// // AKAMAITLC(name, answer_type, target, recordModifiers...)
// var AKAMAITLC = recordBuilder('AKAMAITLC', {
//     args: [
//         ['name', _.isString],
//         [
//             'answer_type',
//             function (value) {
//                 return (
//                     _.isString(value) &&
//                     ['DUAL', 'A', 'AAAA'].indexOf(value) !== -1
//                 );
//             },
//         ],
//         ['target', _.isString],
//     ],
//     transform: function (record, args, modifier) {
//         record.name = args.name;
//         record.answer_type = args.answer_type;
//         record.target = args.target;
//     },
// });

// AZURE_ALIAS(name, type, target, recordModifiers...)
// var AZURE_ALIAS = recordBuilder('AZURE_ALIAS', {
//     args: [
//         ['name', _.isString],
//         ['type', validateAzureAliasType],
//         ['target', _.isString],
//     ],
//     transform: function (record, args, modifier) {
//         record.name = args.name;
//         record.target = args.target;
//         if (_.isObject(record.azure_alias)) {
//             record.azure_alias['type'] = args.type;
//         } else {
//             record.azure_alias = { type: args.type };
//         }
//     },
// });

function validateAzureAliasType(value) {
    if (!_.isString(value)) {
        return false;
    }
    return ['A', 'AAAA', 'CNAME'].indexOf(value) !== -1;
}

// R53_ZONE(zone_id)
function R53_ZONE(zone_id) {
    return function (r) {
        if (_isDomain(r)) {
            r.meta.zone_id = zone_id;
            //console.debug("R53_Opt store zoneid in domainmeta", zone_id);
        } else if (_.isObject(r.r53_alias)) {
            r.r53_alias['zone_id'] = zone_id;
            //console.debug("R53_Opt store zoneid in recordr53alias", zone_id);
        } else {
            //console.debug("R53_Opt MAKE  zoneid in recordr53alias", zone_id);
            r.r53_alias = { zone_id: zone_id };
        }
    };
}

// R53_EVALUATE_TARGET_HEALTH(enabled)
function R53_EVALUATE_TARGET_HEALTH(enabled) {
    return function (r) {
        if (_.isObject(r.r53_alias)) {
            r.r53_alias['evaluate_target_health'] = enabled.toString();
        } else {
            r.r53_alias = { evaluate_target_health: enabled.toString() };
        }
    };
}

function r53AliasOptions(record, processedArgs, processedMetas) {
    // The args are [label, aliasType, target, evaluate_target_health, zone_id].
    var replacement = [
        processedArgs[0],
        processedArgs[1],
        processedArgs[2],
        '',
        '',
    ];

    if (_.isObject(record.r53_alias)) {
        replacement[3] = record.r53_alias['evaluate_target_health'] || 'false';
        replacement[4] = record.r53_alias['zone_id'] || '';
    }

    return replacement;
}

// R53_WEIGHT(weight, set_identifier) configures Route 53 weighted routing.
// weight: integer 0-255, set_identifier: unique string within the weighted group.
function R53_WEIGHT(weight, set_identifier) {
    if (!_.isNumber(weight) || weight < 0 || weight > 255) {
        throw 'R53_WEIGHT: weight must be a number between 0 and 255';
    }
    if (!_.isString(set_identifier) || set_identifier === '') {
        throw 'R53_WEIGHT: set_identifier must be a non-empty string';
    }
    return function (r) {
        if (!_.isObject(r.meta)) {
            r.meta = {};
        }
        r.meta['r53_weight'] = weight.toString();
        r.meta['r53_set_identifier'] = set_identifier;
    };
}

// R53_HEALTH_CHECK_ID(health_check_id) associates a Route 53 health check with the record.
function R53_HEALTH_CHECK_ID(health_check_id) {
    if (!_.isString(health_check_id) || health_check_id === '') {
        throw 'R53_HEALTH_CHECK_ID: health_check_id must be a non-empty string';
    }
    return function (r) {
        if (!_.isObject(r.meta)) {
            r.meta = {};
        }
        r.meta['r53_health_check_id'] = health_check_id;
    };
}

function validateR53AliasType(value) {
    if (!_.isString(value)) {
        return false;
    }
    return (
        [
            'SOA',
            'A',
            'TXT',
            'CNAME',
            'MX',
            'NAPTR',
            'PTR',
            'SRV',
            'SPF',
            'AAAA',
            'CAA',
            'DS',
            'TLSA',
            'SSHFP',
            'SVCB',
            'HTTPS',
        ].indexOf(value) !== -1
    );
}

// AUTOSPLIT is deprecated. It is now a no-op.
var AUTOSPLIT = {};

// Parses coordinates of the form 41°24'12.2"N 2°10'26.5"E
function parseDMSCoordinatesString(inputString) {
    var lat = inputString.match(/(-?\d+).(\d+).([\d\.]+).?\ ?([NS])/);
    var lon = inputString.match(/(-?\d+).(\d+).([\d\.]+).?\ ?([EW])/);

    if (!lat || !lon) {
        return null;
    }

    return {
        lati: {
            dg: parseInt(lat[1]),
            mn: parseInt(lat[2]),
            sc: parseFloat(lat[3]),
            hemi: lat[4],
        },
        long: {
            dg: parseInt(lon[1]),
            mn: parseInt(lon[2]),
            sc: parseFloat(lon[3]),
            hemi: lon[4],
        },
    };
}

// // Parses coordinates of the form 25.24°S 153.15°E
function parseDMMCoordinatesString(inputString) {
    var lat = inputString.match(/(-?\d+(\.\d+)?)°?\s*([NS])/i);
    var lon = inputString.match(/(-?\d+(\.\d+)?)°?\s*([EW])/i);

    if (!lat || !lon) {
        return null;
    }

    var latDeg = Math.floor(parseFloat(lat[1]));
    var latMin = Math.floor((parseFloat(lat[1]) - latDeg) * 60);
    var latSec = Math.round(((parseFloat(lat[1]) - latDeg) * 60 - latMin) * 60);
    var lonDeg = Math.floor(parseFloat(lon[1]));
    var lonMin = Math.floor((parseFloat(lon[1]) - lonDeg) * 60);
    var lonSec = Math.round(((parseFloat(lon[1]) - lonDeg) * 60 - lonMin) * 60);

    var lati = {
        dg: latDeg,
        mn: latMin,
        sc: latSec,
        hemi: lat[3],
    };
    var long = {
        dg: lonDeg,
        mn: lonMin,
        sc: lonSec,
        hemi: lon[3],
    };

    return {
        lati: lati,
        long: long,
    };
}

// builds a uint8 with 4 bit mantissa, 4 bit exponent from a float.
function getENotationInt(x) {
    /*
       9000000000cm = 9e9 == 153 (9^4 + 9) or 9<<4 + 9
       800000000cm = 8e8 == 136 (8^4 + 8) or 8<<4 + 8
       70000000cm = 7e7 == 119 (7^4 + 7) or 7<<4 + 7
       6000000cm = 6e6 == 102 (6^4 + 6) or 6<<4 + 6
       1000000cm = 1e6 == 22 (1^4 + 6) or 1<<4 + 6
       500000cm = 5e5 == 85 (5^4 + 5) or 5<<4 + 5
       40000cm = 4e4 == 68 (4^4 + 4) or 4<<4 + 4
       3000cm = 3e3 == 51 (3^4 + 3) or 3<<4 + 3
       1000cm = 1e3 == 19 (1^4 + 3) or 1<<4 + 1
       200cm = 2e2 == 34 (2^4 + 2) or 2<<4 + 2
       10cm = 1e1 == 17 (1^4 + 1) or 1<<4 + 1
       1cm = 1e0 == 16 (1^4 + 0) or 0<<4 + 0
       0cm = 0e0 == 0
    */
    size = x * 100; // get cm value

    // Convert the number to scientific notation
    var exp = Math.floor(Math.log10(size)); // Get the exponent (base 10)
    var mantissa = size / Math.pow(10, exp); // Normalize mantissa

    // Normalize the mantissa to fit into 4 bits (between 0 and 15)
    while (mantissa < 1 && exp > 0) {
        mantissa *= 10;
        exp -= 1;
    }

    /* Four-bit values greater than 9 are undefined, as are values with a base
    of zero and a non-zero exponent.
    */

    // Ensure mantissa and exponent are within 4-bit range (0-15) but no greater than 9
    mantissa = Math.floor(mantissa); // We truncate decimals
    if (mantissa > 9) {
        mantissa = 9; // Cap mantissa at 9
    }
    if (exp < 0) {
        exp = 0; // We cap negative exponents at 0
    } else if (exp > 9) {
        exp = 9; // Cap exponent at 9
    }
    // convert it to 4bit:4bit uint8
    m_e = (mantissa << 4) | (exp & 0xf);
    return m_e;
}

// Checks LOC parameters and if all is well, renders them into a 'target' string.
// The LOC record has no target string parameter. It only renders one via String().
function locStringBuilder(record, args) {
    record.name = args.name;

    // technically, we don't need this part to build the text target, but
    // it is a good sanity check to compare with later on down the chain
    // when you're in the weeds with maths.
    // Tests depend on it being present. Changes here must reflect in tests.
    nsstring = '';
    ewstring = '';
    precisionbuffer = '';
    ns = args.ns.toUpperCase();
    ew = args.ew.toUpperCase();

    // Handle N/S coords - can use also s1.toFixed(3)
    nsstring =
        args.d1.toString() +
        ' ' +
        args.m1.toString() +
        ' ' +
        args.s1.toString() +
        ' ';
    var nsmatches = args.ns.match(/^([NnSs])$/);
    if (nsmatches == null) {
        throw v + ' is not a valid latitude modifier';
    } else {
        nsstring += ns + ' ';
    }
    // Handle E/W coords - can use also s2.toFixed(3)
    ewstring =
        args.d2.toString() +
        ' ' +
        args.m2.toString() +
        ' ' +
        args.s2.toString() +
        ' ';
    var nsmatches = args.ew.match(/^([EeWw])$/);
    if (nsmatches == null) {
        throw v + ' is not a valid longitude modifier';
    } else {
        ewstring += ew + ' ';
    }

    // handle altitude, size, horizontal precision, vertical precision
    // alt -100000.00 .. 42849672.95m
    // size, horizontal precision, vertical precision 0 .. 90000000.00m
    precisionbuffer =
        (args.alt < -100000
            ? -100000
            : args.alt > 42849672.95
              ? 42849672.95
              : args.alt.toString()) + 'm';
    precisionbuffer +=
        ' ' +
        (args.siz > 90000000
            ? 90000000
            : args.siz < 0
              ? 0
              : args.siz.toString()) +
        'm';
    precisionbuffer +=
        ' ' +
        (args.hp > 90000000 ? 90000000 : args.hp < 0 ? 0 : args.hp.toString()) +
        'm';
    precisionbuffer +=
        ' ' +
        (args.vp > 90000000 ? 90000000 : args.vp < 0 ? 0 : args.vp.toString()) +
        'm';

    record.target = nsstring + ewstring + precisionbuffer;

    return record;
}

// Renders LOC type internal properties from D˚M'S" parameters.
// Change anything here at your peril.
function locDMSBuilder(record, args) {
    LOCEquator = Math.pow(2, 31); // RFC 1876, Section 2.
    LOCPrimeMeridian = Math.pow(2, 31); // RFC 1876, Section 2.
    LOCHours = 60 * 1000;
    LOCDegrees = 60 * LOCHours;
    LOCAltitudeBase = 100000;

    lat = args.d1 * LOCDegrees + args.m1 * LOCHours + args.s1 * 1000;
    lon = args.d2 * LOCDegrees + args.m2 * LOCHours + args.s2 * 1000;
    if (ns == 'N') record.loclatitude = LOCEquator + lat;
    // S
    else record.loclatitude = LOCEquator - lat;
    if (ew == 'E') record.loclongitude = LOCPrimeMeridian + lon;
    // W
    else record.loclongitude = LOCPrimeMeridian - lon;
    // Altitude
    record.localtitude = parseInt((args.alt + LOCAltitudeBase) * 100);
    // 'cast' altitude to fit 'uint32'
    record.localtitude =
        record.localtitude > 4294967295
            ? 4294967295
            : record.localtitude < 0
              ? 0
              : record.localtitude;
    // Size
    record.locsize = getENotationInt(args.siz);
    // Horizontal Precision
    m_e = args.hp;
    record.lochorizpre = getENotationInt(args.hp);

    // Vertical Precision
    m_e = args.vp;
    record.locvertpre = getENotationInt(args.vp);
}

// Post-validation function for LOC that checks if degrees and minutes are integers
function validateIntegers(args) {
    if (args.d1 % 1 !== 0) {
        throw (
            "Degrees N/S shall be an integer: record '" +
            args.name +
            "': *" +
            args.d1 +
            '*, ' +
            args.m1 +
            ', ' +
            args.s1 +
            ', ...'
        );
    }
    if (args.m1 % 1 !== 0) {
        throw (
            "Minutes N/S shall be an integer: record '" +
            args.name +
            "': " +
            args.d1 +
            ', *' +
            args.m1 +
            '*, ' +
            args.s1 +
            ', ...'
        );
    }
    if (args.d2 % 1 !== 0) {
        throw (
            "Degrees E/W shall be an integer: record '" +
            args.name +
            "': *" +
            args.d2 +
            '*, ' +
            args.m2 +
            ', ' +
            args.s2 +
            ', ...'
        );
    }
    if (args.m2 % 1 !== 0) {
        throw (
            "Minutes E/W shall be an integer: record '" +
            args.name +
            "': " +
            args.d2 +
            ', *' +
            args.m2 +
            '*, ' +
            args.s2 +
            ', ...'
        );
    }
}

function ConvertDDToDMS(D, longitude) {
    //stackoverflow, baby. do not re-order the rows.
    return {
        hemi: D < 0 ? (longitude ? 'W' : 'S') : longitude ? 'E' : 'N',
        dg: 0 | (D < 0 ? (D = -D) : D),
        mn: 0 | (((D += 1e-9) % 1) * 60),
        sc: (0 | (((D * 60) % 1) * 60000)) / 1000,
    };
}

// NAMESERVER(name,target)
function NAMESERVER(name) {
    if (arguments.length != 1) {
        throw 'NAMESERVER only accepts one argument for name.';
    }
    return function (d) {
        d.nameservers.push({ name: name });
    };
}

// NAMESERVER_TTL(v): Set the TTL for NAMESERVER records.
function NAMESERVER_TTL(v) {
    if (_.isString(v)) {
        v = stringToDuration(v);
    }
    return { ns_ttl: v.toString() };
}

function format_tt(transform_table) {
    // Turn [[low: 1, high: 2, newBase: 3], [low: 4, high: 5, newIP: 6]]
    // into "1 ~ 2 ~ 3 ~; 4 ~ 5 ~  ~ 6"
    var lines = [];
    for (var i = 0; i < transform_table.length; i++) {
        var ip = transform_table[i];
        var newIP = ip.newIP;
        if (newIP) {
            if (_.isArray(newIP)) {
                newIP = _.map(newIP, function (i) {
                    return num2dot(i);
                }).join(',');
            } else {
                newIP = num2dot(newIP);
            }
        }
        var newBase = ip.newBase;
        if (newBase) {
            if (_.isArray(newBase)) {
                newBase = _.map(newBase, function (i) {
                    return num2dot(i);
                }).join(',');
            } else {
                newBase = num2dot(newBase);
            }
        }
        var row = [num2dot(ip.low), num2dot(ip.high), newBase, newIP];
        lines.push(row.join(' ~ '));
    }
    return lines.join(' ; ');
}

//function UNMANAGED(label_pattern, rType_pattern, target_pattern) {
//    return function (d) {
//        d.unmanaged.push({
//            label_pattern: label_pattern,
//            rType_pattern: rType_pattern,
//            target_pattern: target_pattern,
//        });
//    };
//}

function DISABLE_IGNORE_SAFETY_CHECK(d) {
    // This disables a safety check intended to prevent DNSControl and
    // another system getting into a battle as they both try to update
    // the same record over and over, back and forth.  However, people
    // kept asking for it so... caveat emptor!
    // It only affects the current domain.
    d.unmanaged_disable_safety_check = true;
}

// IGNORE(labelPattern, rtypePattern, targetPattern)
function IGNORE(labelPattern, rtypePattern, targetPattern) {
    if (labelPattern === undefined) {
        labelPattern = '*';
    }
    if (rtypePattern === undefined) {
        rtypePattern = '*';
    }
    if (targetPattern === undefined) {
        targetPattern = '*';
    }
    return function (d) {
        d.unmanaged.push({
            label_pattern: labelPattern,
            rType_pattern: rtypePattern,
            target_pattern: targetPattern,
        });
    };
}

// IGNORE_NAME(name, rTypes)
function IGNORE_NAME(name, rTypes) {
    return IGNORE(name, rTypes);
}

function IGNORE_TARGET(target, rType) {
    return IGNORE('*', rType, target);
}

// IMPORT_TRANSFORM(translation_table, domain, ttl)
function importTransformOptions(record, processedArgs) {
    return [
        processedArgs[0],
        processedArgs[1],
        processedArgs[3],
        processedArgs.length === 5 ? processedArgs[4] : '',
        processedArgs[2],
    ];
}

var importTransformRawBuilder = rawrecordBuilder(
    'IMPORT_TRANSFORM',
    true,
    importTransformOptions
);
function importTransformBuilder(translation_table) {
    arguments[0] = format_tt(translation_table);
    return importTransformRawBuilder.apply(null, arguments);
}

var IMPORT_TRANSFORM = importTransformBuilder;

// IMPORT_TRANSFORM_STRIP(translation_table, domain, ttl, suffixstrip)
var IMPORT_TRANSFORM_STRIP = importTransformBuilder;

// PURGE()
function PURGE(d) {
    d.KeepUnknown = false;
}

// NO_PURGE()
function NO_PURGE(d) {
    d.KeepUnknown = true;
}

// IGNORE_EXTERNAL_DNS(prefix)
// When enabled, DNSControl will automatically detect TXT records created by
// Kubernetes external-dns and ignore both the TXT records and the corresponding
// DNS records they manage. External-dns creates TXT records with content like:
// "heritage=external-dns,external-dns/owner=<owner-id>,external-dns/resource=<resource>"
// This allows DNSControl to coexist with external-dns in the same zone.
//
// Optional prefix parameter: If your external-dns is configured with a custom
// --txt-prefix (e.g., "extdns-"), pass it here to detect those records.
// Without a prefix, it detects the default format ("%{record_type}-" prefixes like "a-", "cname-").
//
// Usage:
//   IGNORE_EXTERNAL_DNS()           // Use default detection (a-, cname-, etc.)
//   IGNORE_EXTERNAL_DNS("extdns-") // Custom prefix
function IGNORE_EXTERNAL_DNS(prefix) {
    return function (d) {
        d.ignore_external_dns = true;
        if (prefix) {
            d.external_dns_prefix = prefix;
        }
    };
}

// ENSURE_ABSENT_REC()
// Usage: A("foo", "1.2.3.4", ENSURE_ABSENT_REC())
function ENSURE_ABSENT_REC() {
    return function (r) {
        r.ensure_absent = true;
    };
}

// ENSURE_ABSENT()
// Usage: ENSURE_ABSENT(A("foo", "1.2.3.4"))
// (BROKEN. COMMENTED OUT UNTIL IT IS FIXED.)
// function ENSURE_ABSENT(r) {
//   //console.log(r);
//   return r;
// }

// AUTODNSSEC
// Permitted values are:
// ""  Do not modify the setting (the default)
// "on"   Enable AUTODNSSEC for this domain
// "off"  Disable AUTODNSSEC for this domain
function AUTODNSSEC_ON(d) {
    d.auto_dnssec = 'on';
}
function AUTODNSSEC_OFF(d) {
    d.auto_dnssec = 'off';
}
function AUTODNSSEC(d) {
    console.log(
        'WARNING: AUTODNSSEC is deprecated. It is now a no-op.  Please use AUTODNSSEC_ON or AUTODNSSEC_OFF. The default is to make no modifications. This message will disappear in a future release.'
    );
}

/**
 * @deprecated
 */
function getModifiers(args, start) {
    var mods = [];
    for (var i = start; i < args.length; i++) {
        mods.push(args[i]);
    }
    return mods;
}

// ip conversion functions from http://stackoverflow.com/a/8105740/121660
// via http://javascript.about.com/library/blipconvert.htm
function IP(dot) {
    var d = dot.split('.');
    // prettier-ignore
    return ((((((+d[0]) * 256) + (+d[1])) * 256) + (+d[2])) * 256) + (+d[3]);
}

function num2dot(num) {
    if (num === undefined) {
        return '';
    }
    if (_.isString(num)) {
        return num;
    }
    var d = num % 256;
    for (var i = 3; i > 0; i--) {
        num = Math.floor(num / 256);
        d = (num % 256) + '.' + d;
    }
    return d;
}

// Cloudflare aliases:

// Meta settings for individual records.
var CF_PROXY_OFF = { cloudflare_proxy: 'off' }; // Proxy disabled.
var CF_PROXY_ON = { cloudflare_proxy: 'on' }; // Proxy enabled.
var CF_PROXY_FULL = { cloudflare_proxy: 'full' }; // Proxy+Railgun enabled.
var CF_CNAME_FLATTEN_OFF = { cloudflare_cname_flatten: 'off' }; // CNAME flattening disabled (default).
var CF_CNAME_FLATTEN_ON = { cloudflare_cname_flatten: 'on' }; // CNAME flattening enabled (paid plans only).
// Per-domain meta settings:
// Proxy default off for entire domain (the default):
var CF_PROXY_DEFAULT_OFF = { cloudflare_proxy_default: 'off' };
// Proxy default on for entire domain:
var CF_PROXY_DEFAULT_ON = { cloudflare_proxy_default: 'on' };
// UniversalSSL off for entire domain:
var CF_UNIVERSALSSL_OFF = { cloudflare_universalssl: 'off' };
// UniversalSSL on for entire domain:
var CF_UNIVERSALSSL_ON = { cloudflare_universalssl: 'on' };
// Per-record comment (works on all plans):
function CF_COMMENT(comment) {
    return { cloudflare_comment: comment };
}
// Per-record tags (requires paid plan):
function CF_TAGS() {
    return { cloudflare_tags: Array.prototype.slice.call(arguments).join(',') };
}
// Enable comment management for domain (opt-in to sync comments):
var CF_MANAGE_COMMENTS = { cloudflare_manage_comments: 'true' };
// Enable tag management for domain (opt-in to sync tags, requires paid plan):
var CF_MANAGE_TAGS = { cloudflare_manage_tags: 'true' };

// Hurricane Electric DNS (HEDNS) aliases:

// Enable Dynamic DNS on a record (preserves existing DDNS key):
var HEDNS_DYNAMIC_ON = { hedns_dynamic: 'on' };
// Disable Dynamic DNS on a record (WARNING: clears the associated DDNS key):
var HEDNS_DYNAMIC_OFF = { hedns_dynamic: 'off' };
// Set a specific DDNS key on a dynamic record (implies HEDNS_DYNAMIC_ON):
function HEDNS_DDNS_KEY(key) {
    return { hedns_dynamic: 'on', hedns_ddns_key: key };
}

// Gidinet aliases:

// GIDINET_PREMIUM_NS(): Emit NAMESERVER records for Gidinet premium DNS
// (dns1..dns5.gidinet.com). Use together with DnsProvider(DNS_GIDINET, 0)
// so the free-tier defaults from GetNameservers are skipped.
//
// Usage:
//   D("premium.example", REG_GIDINET,
//     DnsProvider(DNS_GIDINET, 0),
//     GIDINET_PREMIUM_NS(),
//     A("www", "1.2.3.4"),
//   );
function GIDINET_PREMIUM_NS() {
    return [
        NAMESERVER('dns1.gidinet.com.'),
        NAMESERVER('dns2.gidinet.com.'),
        NAMESERVER('dns3.gidinet.com.'),
        NAMESERVER('dns4.gidinet.com.'),
        NAMESERVER('dns5.gidinet.com.'),
    ];
}

// LOC_BUILDER_DD takes an object:
// label: The DNS label for the LOC record. (default: '@')
// x: Decimal X coordinate.
// y: Decimal Y coordinate.
// alt: Altitude in m. You imperial measurement system people are suckers for punishment.
// ttl: The time for TTL, integer or string. (default: not defined, using DefaultTTL)

function LOC_BUILDER_DD(value) {
    if (!value.x && !value.y) {
        throw 'LOC_BUILDER_DD requires x and y elements';
    }

    if (!value.label) {
        value.label = '@';
    }

    var lati = ConvertDDToDMS(value.x, false);
    var long = ConvertDDToDMS(value.y, true);

    dms = { lati: lati, long: long };

    return LOC_builder_push(value, dms);
}

// LOC_BUILDER_DMM_STR takes an object:
// label: The DNS label for the LOC record. (default: '@')
// str: Input string of Degrees and decimal minutes (DMM) coordinates in the form: 25.24°S 153.15°E
// alt: Altitude in m. You imperial measurement system people are suckers for punishment.
// ttl: The time for TTL, integer or string. (default: not defined, using DefaultTTL)

function LOC_BUILDER_DMM_STR(value) {
    if (!value.str) {
        throw 'LOC_BUILDER_DMM_STR requires a string of the form 25.24°S 153.15°E';
    }

    if (!value.label) {
        value.label = '@';
    }

    var dms = parseDMMCoordinatesString(value.str);

    return LOC_builder_push(value, dms);
}

// LOC_BUILDER_DMS_STR takes an object:
// label: The DNS label for the LOC record. (default: '@')
// str: Input string of degrees, minutes, and seconds (DMS) coordinates in the form: 41°24'12.2"N 2°10'26.5"E
// alt: Altitude in m. You imperial measurement system people are suckers for punishment.
// ttl: The time for TTL, integer or string. (default: not defined, using DefaultTTL)

function LOC_BUILDER_DMS_STR(value) {
    if (!value.str) {
        throw 'LOC_BUILDER_DMS_STR requires a string of the form 33°51′31″S 151°12′51″Es (or 33°51\'31"S 151°12\'51"Es)';
    }

    if (!value.label) {
        value.label = '@';
    }

    var dms = parseDMSCoordinatesString(value.str);

    return LOC_builder_push(value, dms);
}

// LOC_BUILDER_STR takes an object:
// label: The DNS label for the LOC record. (default: '@')
// str: Input string of degrees, minutes, and seconds (DMS) coordinates in the form: 41°24'12.2"N 2°10'26.5"E
// alt: Altitude in m. You imperial measurement system people are suckers for punishment.
// ttl: The time for TTL, integer or string. (default: not defined, using DefaultTTL)

function LOC_BUILDER_STR(value) {
    if (!value.str) {
        throw 'LOC_BUILDER_STR requires a string';
    }

    if (!value.label) {
        value.label = '@';
    }

    var dms = parseDMMCoordinatesString(value.str);
    if (!dms) dms = parseDMSCoordinatesString(value.str);

    return LOC_builder_push(value, dms);
}

function LOC_builder_push(value, dms) {
    r = []; // The list of records to return.
    p = {}; // The metaparameters to set on the LOC record.
    // rawloc = "";

    // Generate a LOC record with the metaparameters.
    if (value.ttl) {
        if (value.alt)
            r.push(
                LOC(
                    value.label,
                    dms.lati.dg,
                    dms.lati.mn,
                    dms.lati.sc,
                    dms.lati.hemi,
                    dms.long.dg,
                    dms.long.mn,
                    dms.long.sc,
                    dms.long.hemi,
                    value.alt,
                    0,
                    0,
                    0,
                    p,
                    TTL(value.ttl)
                )
            );
        else
            r.push(
                LOC(
                    value.label,
                    dms.lati.dg,
                    dms.lati.mn,
                    dms.lati.sc,
                    dms.lati.hemi,
                    dms.long.dg,
                    dms.long.mn,
                    dms.long.sc,
                    dms.long.hemi,
                    0,
                    0,
                    0,
                    0,
                    p,
                    TTL(value.ttl)
                )
            );
    } else {
        if (value.alt)
            r.push(
                LOC(
                    value.label,
                    dms.lati.dg,
                    dms.lati.mn,
                    dms.lati.sc,
                    dms.lati.hemi,
                    dms.long.dg,
                    dms.long.mn,
                    dms.long.sc,
                    dms.long.hemi,
                    value.alt,
                    0,
                    0,
                    0,
                    p
                )
            );
        else
            r.push(
                LOC(
                    value.label,
                    dms.lati.dg,
                    dms.lati.mn,
                    dms.lati.sc,
                    dms.lati.hemi,
                    dms.long.dg,
                    dms.long.mn,
                    dms.long.sc,
                    dms.long.hemi,
                    0,
                    0,
                    0,
                    0,
                    p
                )
            );
    }

    return r;
}

// SPF_BUILDER takes an object:
// parts: The parts of the SPF record (to be joined with ' ').
// label: The DNS label for the primary SPF record. (default: '@')
// raw: Where (which label) to store an unaltered version of the SPF settings.
// ttl: The time for TTL, integer or string. (default: not defined, using DefaultTTL)
// split: The template for additional records to be created (default: '_spf%d')
// flatten: A list of domains to be flattened.
// overhead1: Amout of "buffer room" to reserve on the first item in the spf chain.
// txtMaxSize: The maximum size for each TXT string. Values over 255 will result in multiple strings (default: '255')

function SPF_BUILDER(value) {
    if (!value.parts || value.parts.length < 2) {
        throw 'SPF_BUILDER requires at least 2 elements';
    }
    if (!value.label) {
        value.label = '@';
    }
    if (!value.raw && value.raw !== '') {
        value.raw = '_rawspf';
    }

    r = []; // The list of records to return.
    p = {}; // The metaparameters to set on the main TXT record.
    rawspf = value.parts.join(' '); // The unaltered SPF settings.

    // If flattening is requested, generate a TXT record with the raw SPF settings.
    if (value.flatten && value.flatten.length > 0) {
        p.flatten = value.flatten.join(',');
        // Only add the raw spf record if it isn't an empty string
        if (value.raw !== '') {
            rp = {};
            if (value.ttl) {
                r.push(TXT(value.raw, rawspf, rp, TTL(value.ttl)));
            } else {
                r.push(TXT(value.raw, rawspf, rp));
            }
        }
    }

    // If overflow is specified, enable splitting.
    if (value.overflow) {
        p.split = value.overflow;
    }

    if (value.overhead1) {
        p.overhead1 = value.overhead1;
    }

    if (value.txtMaxSize) {
        p.txtMaxSize = value.txtMaxSize;
    }

    // Generate a TXT record with the metaparameters.
    if (value.ttl) {
        r.push(TXT(value.label, rawspf, p, TTL(value.ttl)));
    } else {
        r.push(TXT(value.label, rawspf, p));
    }

    return r;
}

// CAA_BUILDER takes an object:
// label: The DNS label for the CAA record. (default: '@')
// iodef: The contact mail address. (optional)
// iodef_critical: Boolean if sending report is required/critical. If not supported, certificate should be refused. (optional)
// issue: List of CAs which are allowed to issue certificates for the domain (creates one record for each), or the string 'none'.
// issuewild: List of allowed CAs which can issue wildcard certificates for this domain, or the string 'none'. (creates one record for each)
// issuevmc: List of allowed CAs which can issue VMC certificates for this domain, or the string 'none'. (creates one record for each)
// issuemail: List of allowed CAs which can issue email certificates for this domain, or the string 'none'. (creates one record for each)
// issue_critical: Boolean if issue entries are critical. If not supported, certificate should be refused. (optional)
// issuewild_critical: Boolean if issuewild entries are critical. If not supported, certificate should be refused. (optional)
// issuevmc_critical: Boolean if issuevmc entries are critical. If not supported, certificate should be refused. (optional)
// issuemail_critical: Boolean if issuemail entries are critical. If not supported, certificate should be refused. (optional)
// ttl: The time for TTL, integer or string. (default: not defined, using DefaultTTL)

function CAA_BUILDER(value) {
    if (!value.label) {
        value.label = '@';
    }

    if (value.issue && value.issue == 'none') value.issue = [';'];
    if (value.issuewild && value.issuewild == 'none') value.issuewild = [';'];
    if (value.issuevmc && value.issuevmc == 'none') value.issuevmc = [';'];
    if (value.issuemail && value.issuemail == 'none') value.issuemail = [';'];

    if (
        (!value.issue &&
            !value.issuewild &&
            !value.issuevmc &&
            !value.issuemail) ||
        (value.issue &&
            value.issue.length == 0 &&
            value.issuewild &&
            value.issuewild.length == 0 &&
            value.issuevmc &&
            value.issuevmc.length == 0 &&
            value.issuemail &&
            value.issuemail.length == 0)
    ) {
        throw 'CAA_BUILDER requires at least one entry at issue, issuewild, issuevmc or issuemail';
    }

    var CAA_TTL = function () {};
    if (value.ttl) {
        CAA_TTL = TTL(value.ttl);
    }
    r = []; // The list of records to return.

    if (value.iodef) {
        if (value.iodef_critical) {
            r.push(
                CAA(value.label, 'iodef', value.iodef, CAA_CRITICAL, CAA_TTL)
            );
        } else {
            r.push(CAA(value.label, 'iodef', value.iodef, CAA_TTL));
        }
    }

    if (value.issue) {
        var flag = function () {};
        if (value.issue_critical) {
            flag = CAA_CRITICAL;
        }
        for (var i = 0, len = value.issue.length; i < len; i++)
            r.push(CAA(value.label, 'issue', value.issue[i], flag, CAA_TTL));
    }

    if (value.issuewild) {
        var flag = function () {};
        if (value.issuewild_critical) {
            flag = CAA_CRITICAL;
        }
        for (var i = 0, len = value.issuewild.length; i < len; i++)
            r.push(
                CAA(value.label, 'issuewild', value.issuewild[i], flag, CAA_TTL)
            );
    }

    if (value.issuevmc) {
        var flag = function () {};
        if (value.issuevmc_critical) {
            flag = CAA_CRITICAL;
        }
        for (var i = 0, len = value.issuevmc.length; i < len; i++)
            r.push(
                CAA(value.label, 'issuevmc', value.issuevmc[i], flag, CAA_TTL)
            );
    }

    if (value.issuemail) {
        var flag = function () {};
        if (value.issuemail_critical) {
            flag = CAA_CRITICAL;
        }
        for (var i = 0, len = value.issuemail.length; i < len; i++)
            r.push(
                CAA(value.label, 'issuemail', value.issuemail[i], flag, CAA_TTL)
            );
    }

    return r;
}

/**
 * Encodes a string into DKIM-specific quoted-printable format.
 *
 * This function converts characters that are outside the range of printable ASCII
 * characters, semicolons, DEL, or above ASCII 127 into their quoted-printable
 * hex representation, prefixed by '='. This encoding is used in DKIM signatures
 * to handle characters safely.
 *
 * @param {string} str - The input string to encode.
 * @returns {string} The DKIM quoted-printable encoded string.
 */
function _encodeDKIMQuotedPrintable(str) {
    var hexChars = '0123456789ABCDEF'.split('');
    var result = '';

    for (var i = 0; i < str.length; i++) {
        var charCode = str.charCodeAt(i);
        if (
            charCode < 0x21 ||
            charCode === 0x3b ||
            charCode === 0x3d ||
            charCode === 0x7f ||
            charCode > 0x7f
        ) {
            result +=
                '=' + hexChars[(charCode >>> 4) & 15] + hexChars[charCode & 15];
        } else {
            result += str.charAt(i);
        }
    }
    return result;
}

/**
 * Builds a DKIM DNS TXT record according to RFC 6376 its updates
 * @param {Object} value - Configuration object for the DKIM record.
 * @param {string} value.selector - The selector subdividing the namespace for the domain. **(Required)**
 * @param {string} [value.pubkey] - The base64-encoded public key (RSA or Ed25519).
 *   May be empty for key revocation or non-sending domains.
 * @param {string} [value.label='@'] - The DNS label for the DKIM record (`[selector]._domainkey` prefix is added).
 * @param {string} [value.version='DKIM1'] - The DKIM version (`v=` tag). Currently, only `"DKIM1"` is supported.
 * @param {string|string[]} [value.hashtypes] - Acceptable hash algorithms for signing (`h=` tag).
 *   - Supported values for RSA: `'sha1'`, `'sha256'`
 *   - Supported values for Ed25519: `'sha256'`
 * @param {string} [value.keytype='rsa'] - Key algorithm type (`k=` tag).
 *   - Supported values: `'rsa'`, `'ed25519'`
 * @param {string|string[]} [value.servicetypes] - Service types using this key (`s=` tag).
 *   - Supported values: `'*'`, `'email'`
 *   - `'*'` allows all services; `'email'` restricts usage to email only.
 * @param {string|string[]} [value.flags] - Flags modifying selector interpretation (`t=` tag).
 *   - Supported values: `'y'` (testing mode), `'s'` (subdomain restriction)
 * @param {string} [value.note] - Human-readable note for the record (`n=` tag).
 * @param {number} [value.ttl] - DNS TTL value in seconds.
 *
 * @throws {Error} If a required field is missing or a value is invalid.
 * @returns {Object} DNS TXT record entries for DKIM
 */

function DKIM_BUILDER(value) {
    value = value || {};

    // ========================================
    // PHASE 1: NORMALIZATION
    // ========================================

    // Apply defaults using _.defaults()
    value = _.defaults(value, {
        version: 'DKIM1',
        pubkey: '',
        label: '@',
    });

    // Normalize string|array fields to always be arrays
    if (!_.isEmpty(value.hashtypes)) {
        value.hashtypes = _.isString(value.hashtypes)
            ? [value.hashtypes]
            : value.hashtypes;
    }

    if (!_.isEmpty(value.servicetypes)) {
        value.servicetypes = _.isString(value.servicetypes)
            ? [value.servicetypes]
            : value.servicetypes;
    }

    if (!_.isEmpty(value.flags)) {
        value.flags = _.isString(value.flags) ? [value.flags] : value.flags;
    }

    // ========================================
    // PHASE 2: VALIDATION (Fail Fast)
    // ========================================

    // Static allowed values
    var ALLOWED_VERSIONS = ['DKIM1'];
    var ALLOWED_KEYTYPES = ['rsa', 'ed25519'];
    var ALLOWED_HASHTYPES = {
        rsa: ['sha1', 'sha256'],
        ed25519: ['sha256'],
    };
    var ALLOWED_SERVICETYPES = ['*', 'email'];
    var ALLOWED_FLAGS = ['y', 's'];

    // Required fields
    if (_.isEmpty(value.selector)) {
        throw 'DKIM_BUILDER selector cannot be empty';
    }

    // Version validation
    if (!_.contains(ALLOWED_VERSIONS, value.version)) {
        throw (
            'DKIM_BUILDER version must be one of: ' +
            ALLOWED_VERSIONS.join(', ')
        );
    }

    // Keytype validation
    if (
        !_.isEmpty(value.keytype) &&
        !_.contains(ALLOWED_KEYTYPES, value.keytype)
    ) {
        throw (
            'DKIM_BUILDER keytype must be one of: ' +
            ALLOWED_KEYTYPES.join(', ') +
            ', ' +
            value.keytype +
            ' given'
        );
    }

    // Hashtypes validation (now always an array after normalization)
    if (!_.isEmpty(value.hashtypes)) {
        var allowedHashtypes = ALLOWED_HASHTYPES[value.keytype || 'rsa'];
        var invalidHashtypes = _.difference(value.hashtypes, allowedHashtypes);
        if (invalidHashtypes.length > 0) {
            throw (
                'DKIM_BUILDER hashtypes for ' +
                value.keytype +
                ' must be one of: ' +
                allowedHashtypes.join(', ')
            );
        }
    }

    // Servicetypes validation (now always an array after normalization)
    if (!_.isEmpty(value.servicetypes)) {
        var invalidServicetypes = _.difference(
            value.servicetypes,
            ALLOWED_SERVICETYPES
        );
        if (invalidServicetypes.length > 0) {
            throw (
                'DKIM_BUILDER servicetypes must be one of: ' +
                ALLOWED_SERVICETYPES.join(', ')
            );
        }
    }

    // Flags validation (now always an array after normalization)
    if (!_.isEmpty(value.flags)) {
        var invalidFlags = _.difference(value.flags, ALLOWED_FLAGS);
        if (invalidFlags.length > 0) {
            throw (
                'DKIM_BUILDER flags must be one of: ' + ALLOWED_FLAGS.join(', ')
            );
        }
    }

    // ========================================
    // PHASE 3: BUILD OUTPUT
    // ========================================

    // Build record RFC 6376 order: v=, h=, k=, n=, p=, s=, t=
    var record = [];

    record.push('v=' + value.version);

    if (value.hashtypes) {
        record.push('h=' + value.hashtypes.join(':'));
    }

    if (value.keytype) {
        record.push('k=' + value.keytype);
    }

    if (!_.isEmpty(value.note)) {
        record.push('n=' + _encodeDKIMQuotedPrintable(value.note));
    }

    record.push('p=' + value.pubkey);

    if (value.servicetypes) {
        record.push('s=' + value.servicetypes.join(':'));
    }

    if (value.flags) {
        record.push('t=' + value.flags.join(':'));
    }

    // Build label
    var fullLabel = value.selector + '._domainkey';
    if (value.label !== '@') {
        fullLabel += '.' + value.label;
    }

    // Handle TTL
    var DKIM_TTL = value.ttl ? TTL(value.ttl) : function () {};

    return TXT(fullLabel, record.join('; '), DKIM_TTL);
}

// DMARC_BUILDER takes an object:
// label: The DNS label for the DMARC record (_dmarc prefix is added; default: '@')
// version: The DMARC version, by default DMARC1 (optional)
// policy: The DMARC policy (p=), must be one of 'none', 'quarantine', 'reject'
// subdomainPolicy: The DMARC policy for subdomains (sp=), must be one of 'none', 'quarantine', 'reject' (optional)
// nonexistentSubdomainPolicy: The DMARC policy for nonexistent subdomains (np=), must be one of 'none', 'quarantine', 'reject' (optional)
// alignmentSPF: 'strict'/'s' or 'relaxed'/'r' alignment for SPF (aspf=, default: 'r')
// alignmentDKIM: 'strict'/'s' or 'relaxed'/'r' alignment for DKIM (adkim=, default: 'r')
// rua: Array of aggregate report targets (optional)
// ruf: Array of failure report targets (optional)
// publicSuffixDomain: 'y', 'n', or 'u' indicates whether the domain is a Public Suffix Domain (`psd=`, default='u')
// testMode: 'y' or 'n' DMARC policy test mode dictates whether p, sp, or np policy tags are applied (`t=`, default: 'n')
// failureOptions: Object or string; Object containing booleans SPF and DKIM, string is passed raw (fo=, default: '0')
// ttl: Input for TTL method
function DMARC_BUILDER(value) {
    if (!value) {
        value = {};
    }
    if (!value.label) {
        value.label = '@';
    }

    if (!value.version) {
        value.version = 'DMARC1';
    }

    var label = '_dmarc';
    if (value.label !== '@') {
        label += '.' + value.label;
    }

    if (!value.policy) {
        value.policy = 'none';
    }

    if (
        !value.policy === 'none' ||
        !value.policy === 'quarantine' ||
        !value.policy === 'reject'
    ) {
        throw 'Invalid DMARC policy';
    }

    var record = [];
    record.push('v=' + value.version);
    record.push('p=' + value.policy);

    // Subdomain policy
    if (
        !value.subdomainPolicy === 'none' ||
        !value.subdomainPolicy === 'quarantine' ||
        !value.subdomainPolicy === 'reject'
    ) {
        throw 'Invalid DMARC subdomain policy';
    }
    if (value.subdomainPolicy) {
        record.push('sp=' + value.subdomainPolicy);
    }

    // Nonexistent-Subdomain policy
    if (
        !value.nonexistentSubdomainPolicy === 'none' ||
        !value.nonexistentSubdomainPolicy === 'quarantine' ||
        !value.nonexistentSubdomainPolicy === 'reject'
    ) {
        throw 'Invalid DMARC nonexistent-subdomain policy';
    }
    if (value.nonexistentSubdomainPolicy) {
        record.push('np=' + value.nonexistentSubdomainPolicy);
    }

    // Alignment DKIM
    if (value.alignmentDKIM) {
        switch (value.alignmentDKIM) {
            case 'relaxed':
                value.alignmentDKIM = 'r';
                break;
            case 'strict':
                value.alignmentDKIM = 's';
                break;
            case 'r':
            case 's':
                break;
            default:
                throw 'Invalid DMARC DKIM alignment policy';
        }
        record.push('adkim=' + value.alignmentDKIM);
    }

    // Alignment SPF
    if (value.alignmentSPF) {
        switch (value.alignmentSPF) {
            case 'relaxed':
                value.alignmentSPF = 'r';
                break;
            case 'strict':
                value.alignmentSPF = 's';
                break;
            case 'r':
            case 's':
                break;
            default:
                throw 'Invalid DMARC DKIM alignment policy';
        }
        record.push('aspf=' + value.alignmentSPF);
    }

    // Percentage
    if (value.percent) {
        record.push('pct=' + value.percent);
        console.log('WARNING: DMARC pct tag depracated.');
    }

    // Aggregate reports
    if (value.rua && value.rua.length > 0) {
        record.push('rua=' + value.rua.join(','));
    }

    // Failure reports
    if (value.ruf && value.ruf.length > 0) {
        record.push('ruf=' + value.ruf.join(','));
    }

    // Failure reporting options
    if (value.ruf && value.failureOptions) {
        var fo = '0';
        if (_.isObject(value.failureOptions)) {
            if (value.failureOptions.DKIM) {
                fo = 'd';
            }
            if (value.failureOptions.SPF) {
                fo = 's';
            }
            if (value.failureOptions.DKIM && value.failureOptions.SPF) {
                fo = '1';
            }
        } else {
            fo = value.failureOptions;
        }

        if (fo !== '0') {
            record.push('fo=' + fo);
        }
    }

    // Failure report format
    if (value.ruf && value.failureFormat) {
        record.push('rf=' + value.failureFormat);
        console.log('WARNING: DMARC rf tag depracated.');
    }

    // Report interval
    if (value.reportInterval) {
        if (_.isString(value.reportInterval)) {
            value.reportInterval = stringToDuration(value.reportInterval);
        }

        record.push('ri=' + value.reportInterval);
        console.log('WARNING: DMARC ri tag depracated.');
    }

    // Public Suffix Domain
    if (value.publicSuffixDomain) {
        if (
            !value.publicSuffixDomain === 'u' ||
            !value.publicSuffixDomain === 'y' ||
            !value.publicSuffixDomain === 'n'
        ) {
            throw 'Invalid public-suffix-domain tag';
        }

        record.push('psd=' + value.publicSuffixDomain);
    }

    // Test mode

    if (value.testMode) {
        if (!value.testMode === 'y' || !value.testMode === 'n') {
            throw 'Invalid test-mode tag';
        }

        record.push('t=' + value.testMode);
    }

    if (value.ttl) {
        return TXT(label, record.join('; '), TTL(value.ttl));
    }
    return TXT(label, record.join('; '));
}

// Documentation of the records: https://learn.microsoft.com/en-us/microsoft-365/enterprise/external-domain-name-system-records?view=o365-worldwide
function M365_BUILDER(name, value) {
    // value is optional
    if (!value) {
        value = {};
    }

    if (value.mx !== false) {
        value.mx = true;
    }
    if (value.autodiscover !== false) {
        value.autodiscover = true;
    }
    if (value.dkim !== false) {
        value.dkim = true;
    }

    if (!value.label) {
        value.label = '@';
    }

    if (!value.domainGUID) {
        // Does not work with dashes in domain name.
        // Microsoft uses its own, (probably) deterministic algorithm to transform these domains.
        // Unfortunately, underlying algorithm is not known to us.
        if (name.indexOf('-') !== -1) {
            throw (
                'M365_BUILDER requires domainGUID for domains with dashes: ' +
                name
            );
        }

        value.domainGUID = name.replace(/\./g, '-');
    }

    if (value.dkim && !value.initialDomain) {
        throw (
            "M365_BUILDER requires your M365 account's initial domain to set up DKIM (default: enabled): " +
            name
        );
    }

    var r = [];

    // MX (default: true)
    if (value.mx) {
        r.push(
            MX(
                value.label,
                0,
                value.domainGUID + '.mail.protection.outlook.com.'
            )
        );
    }

    // Autodiscover (default: true)
    if (value.autodiscover) {
        if ((value.label = '@')) {
            r.push(CNAME('autodiscover', 'autodiscover.outlook.com.'));
        } else {
            r.push(
                CNAME(
                    'autodiscover.' + value.label,
                    'autodiscover.outlook.com.'
                )
            );
        }
    }

    // DKIM (default: true)
    if (value.dkim) {
        r.push(
            CNAME(
                'selector1._domainkey',
                'selector1-' +
                    value.domainGUID +
                    '._domainkey.' +
                    value.initialDomain +
                    '.'
            )
        );
        r.push(
            CNAME(
                'selector2._domainkey',
                'selector2-' +
                    value.domainGUID +
                    '._domainkey.' +
                    value.initialDomain +
                    '.'
            )
        );
    }

    // Skype for Business (default: false)
    if (value.skypeForBusiness) {
        r.push(CNAME('lyncdiscover', 'webdir.online.lync.com.'));
        r.push(CNAME('sip', 'sipdir.online.lync.com.'));
        r.push(SRV('_sip._tls', 100, 1, 443, 'sipdir.online.lync.com.'));
        r.push(
            SRV(
                '_sipfederationtls._tcp',
                100,
                1,
                5061,
                'sipfed.online.lync.com.'
            )
        );
    }

    // Mobile Device Management (default: false)
    if (value.mdm) {
        r.push(
            CNAME(
                'enterpriseregistration',
                'enterpriseregistration.windows.net.'
            )
        );
        r.push(
            CNAME(
                'enterpriseenrollment',
                'enterpriseenrollment.manage.microsoft.com.'
            )
        );
    }

    return r;
}

// This is a no-op.  Long TXT records are handled natively now.
function DKIM(arr) {
    return arr;
}

// Function wrapper for glob() for recursively loading files.
// As the main function (in Go) is in our control anyway, all the values here are already sanity-checked.
// Note: glob() is only an internal undocumented helper function. So use it on your own risk.
function require_glob() {
    arguments[2] = 'js'; // force to only include .js files.
    var files = glob.apply(null, arguments);
    for (var i = 0; i < files.length; i++) {
        require(files[i]);
    }
    return files;
}

// Set default values for CLI variables
function CLI_DEFAULTS(defaults) {
    for (var key in defaults) {
        if (typeof this[key] === 'undefined') {
            this[key] = defaults[key];
        }
    }
}

function FETCH() {
    return fetch.apply(null, arguments).catch(PANIC);
}

// DOMAIN_ELSEWHERE is a helper macro that delegates a domain to a
// static list of nameservers.  It updates the registrar (the parent)
// with a list of nameservers.  This is used when we own the domain (we
// control the registrar) but something else controls the DNS records
// (often a third-party of Azure).
// Usage: DOMAIN_ELSEWHERE("example.com", REG_NAMEDOTCOM, ["ns1.foo.com", "ns2.foo.com"]);
function DOMAIN_ELSEWHERE(domain, registrar, nslist) {
    D(domain, registrar, NO_PURGE);
    // NB(tlim): NO_PURGE is added as a precaution since something else
    // is maintaining the DNS records in that zone.  In theory this is
    // not needed since this domain won't have a DSP defined.
    for (var i = 0; i < nslist.length; i++) {
        D_EXTEND(domain, NAMESERVER(nslist[i]));
    }
}

// DOMAIN_ELSEWHERE_AUTO is similar to DOMAIN_ELSEWHERE but the list of
// nameservers is queried from a DNS Service Provider.
// Usage: DOMAIN_ELSEWHERE_AUTO("example.com", REG_NAMEDOTCOM, DNS_FOO)
function DOMAIN_ELSEWHERE_AUTO(domain, registrar, dsplist) {
    D(domain, registrar, NO_PURGE);
    // NB(tlim): NO_PURGE is required since something else
    // is maintaining the DNS records in that zone, and we have access
    // to updating it (but we don't want to use it.)
    for (var i = 2; i < arguments.length; i++) {
        D_EXTEND(domain, DnsProvider(arguments[i]));
    }
}

var END = {}; // This is null. It permits the last item to include a comma.
// D("foo.com", ...
//    A(...),
//    A(...),
//    A(...),
// END)

// Record modifiers:

// Permit labels like "foo.bar.com.bar.com" (normally an error):
var DISABLE_REPEATED_DOMAIN_CHECK = { skip_fqdn_check: 'true' };
// D("bar.com", ...
//     A("foo.bar.com", "10.1.1.1", DISABLE_REPEATED_DOMAIN_CHECK),
// )

// ============================================================

// RTYPES

// Background:
// Old-style commands: Commands built using recordBuild() are the original
// style.  They all validation and pre-processing here in helpers.js. This
// seemed like a good idea at the time, but in hindsight it puts a burden on the
// developer to know both Javascript and go.

// New-style commands: Command built using rawrecordBuilder() are the new style.
// They simply pack up the arguments listed in dnsconfig.js and store them in
// .rawrecords. This is passed to the Go code, which is responsible for all
// validation, pre-processing, etc.  The benefit is this minimizes the need for
// Javascript knowledge, and allows us to use the testing platform build into
// Go.

function rawrecordBuilder(type, noLabel, optionalsFn) {
    return function () {
        var rawArgs = [];
        if (noLabel !== undefined && noLabel) {
            rawArgs.push('@');
        }
        // Copy the raw args locally.
        for (var i = 0; i < arguments.length; i++) {
            rawArgs.push(arguments[i]);
        }

        // Record which line called this record type.
        // NB(tlim): Hopefully we can find a better way to do this in the
        // future. Right now we're faking that there was an error just to parse
        // out the line number. That's inefficient but I can't find anything better.
        // This will certainly break if we change to a different Javascript interpreter.
        // Hopefully any other interpreter will have a better way to do this.
        var positionLines = new Error().stack.split('\n');
        var position = positionLines[positionLines.length - 2];

        return function (d) {
            var record = {
                type: type,
                filepos: position,
                ttl: d.defaultTTL,
            };

            // Process the args: Functions are executed, objects are assumed to
            // be meta and stored, strings are assumed to be args and are
            // stored.
            // NB(tlim): Allowing for the intermixing of args and meta seems
            // bad.  It might be better to simply preserve the first n items as
            // args, then assume the rest are metas. That would be more similar
            // to the old style functions. However at this time I can't think of
            // a reason this isn't sufficient.
            var processedArgs = [];
            var processedMetas = [];
            for (var i = 0; i < rawArgs.length; i++) {
                var r = rawArgs[i];
                if (_.isFunction(r)) {
                    r(record);
                } else if (_.isArray(r)) {
                    // An array argument (e.g. TXT("n", ["a", "b"])) is a set of
                    // string segments; join them into a single argument,
                    // mirroring the legacy recordBuilder. Arrays are never
                    // metadata, so this must be checked before _.isObject()
                    // (which is true for arrays).
                    processedArgs.push(r.join(''));
                } else if (_.isObject(r)) {
                    // Convert a transform array to its encoded string form
                    // (see format_tt), mirroring the legacy recordBuilder.
                    // Otherwise the array crosses to Go as an unparseable
                    // string and IP() numbers are never converted to dotted
                    // decimal.
                    if (r.transform && _.isArray(r.transform)) {
                        r.transform = format_tt(r.transform);
                    }
                    processedMetas.push(r);
                } else {
                    processedArgs.push(r);
                }
            }

            // Grab meta data that is OptionalsFields
            if (optionalsFn !== undefined) {
                //console.debug("FOO", optionalsFn)
                processedArgs = optionalsFn(
                    record,
                    processedArgs,
                    processedMetas
                );
            }

            // Modifier functions may have written to record.meta. Capture that
            // so it propagates to Go as a meta; otherwise it would be silently
            // dropped.
            if (_.isObject(record.meta) && !_.isEmpty(record.meta)) {
                processedMetas.push(record.meta);
            }

            // Store the processed args.
            record.args = processedArgs;
            record.metas = processedMetas;

            // Capture the D_EXTEND() subdomain (per-record, at push time).
            // d.subdomain is domain-level mutable state that each D_EXTEND()
            // call overwrites, so it must be snapshotted here. The label
            // rewriting itself is done in Go (models.ImportRawRecords).
            record.subdomain = d.subdomain;

            // Add this raw record to the list of records.
            d.rawrecords.push(record);

            return record;
        };
    };
}

// PLEASE KEEP THIS LIST ALPHABETICAL!

var A = rawrecordBuilder('A');
var AAAA = rawrecordBuilder('AAAA');
var ADGUARDHOME_AAAA_PASSTHROUGH = rawrecordBuilder(
    'ADGUARDHOME_AAAA_PASSTHROUGH'
);
var ADGUARDHOME_A_PASSTHROUGH = rawrecordBuilder('ADGUARDHOME_A_PASSTHROUGH');
var AKAMAICDN = rawrecordBuilder('AKAMAICDN');
var AKAMAITLC = rawrecordBuilder('AKAMAITLC');
var ALIAS = rawrecordBuilder('ALIAS');
var AZURE_ALIAS = rawrecordBuilder('AZURE_ALIAS');
var BUNNY_DNS_PZ = rawrecordBuilder('BUNNY_DNS_PZ');
var BUNNY_DNS_RDR = rawrecordBuilder('BUNNY_DNS_RDR');
var CAA = rawrecordBuilder('CAA', false, caaOptions);
var CF_REDIRECT = rawrecordBuilder('CF_REDIRECT', true);
var CF_SINGLE_REDIRECT = rawrecordBuilder(
    'CLOUDFLAREAPI_SINGLE_REDIRECT',
    true
);
var CF_TEMP_REDIRECT = rawrecordBuilder('CF_TEMP_REDIRECT', true);
var CF_WORKER_ROUTE = rawrecordBuilder('CF_WORKER_ROUTE', true);
var CLOUDNS_WR = rawrecordBuilder('CLOUDNS_WR');
var CNAME = rawrecordBuilder('CNAME');
var DHCID = rawrecordBuilder('DHCID');
var DNAME = rawrecordBuilder('DNAME');
var DNSKEY = rawrecordBuilder('DNSKEY');
var DS = rawrecordBuilder('DS');
var FRAME = rawrecordBuilder('FRAME');
var HTTPS = rawrecordBuilder('HTTPS');
var LOC = rawrecordBuilder('LOC');
var LUA = rawrecordBuilder('LUA');
var MIKROTIK_FORWARDER = rawrecordBuilder('MIKROTIK_FORWARDER');
var MIKROTIK_FWD = rawrecordBuilder('MIKROTIK_FWD');
var MIKROTIK_NXDOMAIN = rawrecordBuilder('MIKROTIK_NXDOMAIN');
var MX = rawrecordBuilder('MX');
var NAPTR = rawrecordBuilder('NAPTR');
var NS = rawrecordBuilder('NS');
var OPENPGPKEY = rawrecordBuilder('OPENPGPKEY');
var PORKBUN_URLFWD = rawrecordBuilder('PORKBUN_URLFWD');
var PTR = rawrecordBuilder('PTR');
var R53_ALIAS = rawrecordBuilder('R53_ALIAS', false, r53AliasOptions);
var RP = rawrecordBuilder('RP');
var SMIMEA = rawrecordBuilder('SMIMEA');
var SOA = rawrecordBuilder('SOA');
var SRV = rawrecordBuilder('SRV');
var SSHFP = rawrecordBuilder('SSHFP');
var SVCB = rawrecordBuilder('SVCB');
var TLSA = rawrecordBuilder('TLSA');
var TXT = rawrecordBuilder('TXT');
var URL = rawrecordBuilder('URL');
var URL301 = rawrecordBuilder('URL301');
