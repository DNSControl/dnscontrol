D_EXTEND('dnscontrol.org',
  SPF_BUILDER({
    label: '@',
    parts: [
      'v=spf1',
      '-all',
    ],
  }),
);
