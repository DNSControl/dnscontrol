D("foo.com", "none",
    // LOC "subdomain", d1, m1, s1, "[NnSs]", d2, m2, s2, "[EeWw]", alt, siz, hp, vp)
    LOC("@", 42, 21, 54, "N", 71, 6, 18, "W", -24, 30, 0, 0), //42 21 54     N  71 06  18     W -24m 30m
    LOC("a", 42, 21, 43.952, "N", 71, 5, 6.344, "W", -24.01, 1, 200, 10), //42 21 43.952 N  71 5   6.344  W -24.01m 1m 200m
    LOC("b", 52, 14, 5, "N", 0, 8, 50, "E", 10.33, 0, 0, 0), //52 14 05     N  00 08  50     E 10.33m
    LOC("c", 32, 7, 19, "S", 116, 2, 25, "E", 10, 0, 0, 0), //32  7 19     S 116  2  25     E 10m
    LOC("d", 42, 21, 28.764, "N", 71, 0, 51.617, "W", -44, 2000, 0, 0), //42 21 28.764 N  71 00  51.617 W -44m 2000m
    LOC("d-alt-highest", 42, 21, 28.764, "N", 71, 0, 51.617, "W", 42849672.95, 2000, 0, 0), //42 21 28.764 N  71 00  51.617 W 42849672.95m 2000m
    LOC("d-alt-lowest", 42, 21, 28.764, "N", 71, 0, 51.617, "W", -100000.00, 2000, 0, 0), //42 21 28.764 N  71 00  51.617 W -100000.00m 2000m
    LOC("d-alt-toohigh", 42, 21, 28.764, "N", 71, 0, 51.617, "W", 42849672.96, 2000, 0, 0), //42 21 28.764 N  71 00  51.617 W 42849672.95m 2000m
    LOC("d-alt-toolow", 42, 21, 28.764, "N", 71, 0, 51.617, "W", -100000.01, 2000, 0, 0), //42 21 28.764 N  71 00  51.617 W -100000m 2000m
    LOC("d-horizprecision-hi", 42, 21, 28.764, "N", 71, 0, 51.617, "W", 0, 1, 90000000, 0), //42 21 28.764 N 71 00 51.617 W 0m 1m 90000000m 0.00m
    LOC("d-horizprecision-toohi", 42, 21, 28.764, "N", 71, 0, 51.617, "W", 0, 1, 98765432, 0), //42 21 28.764 N 71 00 51.617 W 0m 1m 90000000m 0.00m
    LOC("d-horizprecision-toolow", 42, 21, 28.764, "N", 71, 0, 51.617, "W", 0, 1, -1, 0), //42 21 28.764 N 71 00 51.617 W 0m 1m 0m 0.00m
    LOC("d-size-toohi", 42, 21, 28.764, "N", 71, 0, 51.617, "W", 0, 98765432, 0, 0), //42 21 28.764 N  71 00  51.617 W -44m 2000m
    LOC("d-size-toolow", 42, 21, 28.764, "N", 71, 0, 51.617, "W", 0, -1, 0, 0), //42 21 28.764 N  71 00  51.617 W -44m 0m
    LOC("d-size-hi", 42, 21, 28.764, "N", 71, 0, 51.617, "W", 0, 90000000, 0, 0), //42 21 28.764 N  71 00  51.617 W -44m 90000000m
    LOC("d-vertprecision-hi", 42, 21, 28.764, "N", 71, 0, 51.617, "W", 0, 1, 0, 90000000), //42 21 28.764 N 71 00 51.617 W 0m 1m 0.00m 90000000m
    LOC("d-vertprecision-toohi", 42, 21, 28.764, "N", 71, 0, 51.617, "W", 0, 1, 0, 98765432), //42 21 28.764 N 71 00 51.617 W 0m 1m 0.00m 90000000m
    LOC("d-vertprecision-toolow", 42, 21, 28.764, "N", 71, 0, 51.617, "W", 0, 1, 0, -1), //42 21 28.764 N 71 00 51.617 W 0m 1m 0.00m 0m

    // via the Decimal degrees to LOC builder.
    LOC_BUILDER_DD({
        label: "big-ben",
        x: 51.50084265331501,
        y: -0.12462541415599787,
        alt: 6,
    }),
    LOC_BUILDER_DD({
        label: "white-house",
        x: 38.89775977858357,
        y: -77.03655125982903,
        alt: 19,
        ttl: "5m",
    }),
    LOC_BUILDER_DMS_STR({
        label: "opera-house",
        str: '33°51′31″S 151°12′51″E',
        alt: 4,
        ttl: "5m",
    }),
    LOC_BUILDER_DMS_STR({
        label: "opera-house2",
        str: '33°51\'31"S 151°12\'51"E',
        alt: 4,
        ttl: "5m",
    }),
    LOC_BUILDER_DMS_STR({
        label: "opera-house3",
        str: '33d51m31sS 151d12m51sE',
        alt: 4,
        ttl: "5m",
    }),
    LOC_BUILDER_DMS_STR({
        label: "opera-house4",
        str: '33d51m31s S 151d12m51s E',
        alt: 4,
        ttl: "5m",
    }),
    LOC_BUILDER_DMM_STR({
        label: "fraser-island",
        str: '25.24°S 153.15°E',
        alt: 3,
    }),
    LOC_BUILDER_STR({
        label: "tasmania",
        str: '42°S 147°E',
        alt: 3,
    }),
    LOC_BUILDER_STR({
        label: "hawaii",
        str: '21.5°N 158.0°W',
        alt: 920,
    }),
    LOC_BUILDER_STR({
        label: "old-faithful",
        str: '44.46046°N 110.82815°W',
        alt: 2240,
    }),
    LOC_BUILDER_STR({
        label: "ribblehead-viaduct",
        str: '54.210436°N 2.370231°W',
        alt: 300,
    }),
    LOC_BUILDER_STR({
        label: "guinness-brewery",
        str: '53°20′40″N 6°17′20″W',
        alt: 300,
    }),
);
