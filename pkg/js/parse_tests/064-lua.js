D(
    'example.com',
    'reg',
    LUA('app', 'a', ['return_', '127_0_0_1'], TTL(60)),
    LUA('_diag', 'txt', "; return 'ok'", TTL(30))
);
