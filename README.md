# Monitor web sites and services

## Start

This service need only be run from the same directory as the sources.

    go build -v
    ./sitecheck

Sitecheck serves a webpage on the host at port 8080.  Use
http://localhost:8080 to access.

## Configuration File

Add services to **sitecheck.yml**.

    - name: "service name"
      type: "website" or "etcd" or "docker" or "registry"
      description: <optional> "descriptive text for hover"
      timeout: <optional> 70 <in seconds>
      url:
        - "http://fumble.foo.bar.com:666/root"
