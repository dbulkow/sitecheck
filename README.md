# Monitor web sites and services

## Configuration File

Add services to **sitecheck.conf**.

    [[service]]
    name = "service name"
    type = "website" or "etcd"
    url  = "http://fumble.foo.bar.com:666/root"
