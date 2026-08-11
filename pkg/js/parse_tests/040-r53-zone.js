D('foo.com', 'none', R53_ZONE('Z2FTEDLFRTZ'));
D(
    'foo.com!internal',
    'none',
    R53_ZONE('Z2FTEDLFRTF'),
    R53_ALIAS('atest', 'A', 'foo.com.', R53_ZONE('Z2FTEDLFRTZ'))
);
