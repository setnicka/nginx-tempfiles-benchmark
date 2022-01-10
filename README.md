# Nginx tempfiles benchmark

## How to run

1. Obtain two nginx binaries and place them into this folder:
   * `nginx-vanilla` - nginx binary builded without any patches
   * `nginx` - nginx binary builded with tempfiles patch
2. Run benchmark by executing `./run.sh`

## How tests works

Nginx is set up to proxy_pass all requests to the origin with caching enabled.

Origin serves each request with continuous stream of random bytes for given
amount of seconds (3 seconds by default, 10MiB/s rate).

Client runs N paralel workers, each of the workers sends request for file-0,
file-1, file-2, ... until time limit is reached (default: 30 seconds and 100
workers).

## Test cases

There are three test suites:

1. Full speed origin - Origin returns files with its full speed. Typical
   situation with quick origin server with static files.
2. Slow speed origin (each file streamed for 1s) - Origin sends file with
   constant speed in 10 chunks. This could be a slow origin server on
   slow network or a live video streaming service with short chunks.
3. Slow speed origin (each file streamed for 3s) - Origin sends file with
   constant speed in 30 chunks (10 chunks per second). This is typical example
   of live video streaming service.

In each test suites there were four instances of the nginx compared:

1. Vanilla nginx without cache lock and tempfiles - This is the most basic
   caching setup when every MISS is proxy passed to the origin. Already cached
   files are server from the cache.
2. Vanilla nginx with cache lock - This nginx does only one request for the same
   file to the origin, other requests for the same file are paused and are served
   when the file is fully cached. This is common technique to lower load to the
   origin in high-demand scenarios.
3. Patched nginx with cache lock - This is same scenario as 2. but with patched
   binary. This is just an check that tempfiles patch does not degraded
   performance.
4. Patched nginx with cache tempfiles - As in 2. this nginx does only one request
   for the same file to the origin, but other request are not paused. They are
   served on-the-fly from the same cache tempfile.

## Results

TODO
