# Lennut

A _very_ simple reverse tunnel (get it?) that allows TCP proxying where the
"server" end of the connection is not accessible.

It has 0% code coverage. Who needs tests anyway?

## What?

This is a little like ngrok but way simpler. Imagine for example that you have a
server on your Mac listening at `localhost:8000` and you want to connect to it
from a Docker container, but make it appear to be running at `localhost:8000`
INSIDE the container. On linux you could just use `--network=host` but on macOS
that doesn't work.

So instead you can use `lennut`.

The "server" and "client" refer to the direction in which the tunnel is
established. But it's a _reverse_ tunnel so the server end is the one trying to
connect out.

In our docker example, you can make this work by running `lennut -server` in a
docker container that shared the target container's network namespace and
exposing it's _client_ port to the host.

Then `lennut` client can run on your host, can establish a connection to the
server container. Once established the roles reverse: any inbound TCP connection
to the server's inbound port will be proxied to the established client and the
client will proxy them on to the configured backend address on the host.

## Example

Run a server on your docker host. We'll make it a simple tcp echo:

```
$ socat -v tcp-l:8080,fork exec:"/bin/cat"
```

Now run a container that needs to connect to the host service in Docker. I'll
use a sleep container as a place holder for now to demonstrate.

```
$ docker run --rm -d --name sleepy -p 3001:3001 alpine sleep 3600
4203dbab943f8db20355fc7640f5783cc41f1706ab93874b9e5b834cda15e8d9
```

Note that we had to expose port 3001 here. More on that later...

Now we want to be able to exec into the `sleepy` container and talk to socat over
`localhost:8080` as if it were... local.

To do that we need to run a lennut _server_ inside the same network namespace.

```
$ docker run --rm -d --name lennut \
  --network=container:sleepy \
  pbanks/lennut -bind-proxy localhost:8080
```

This will proxy connections to `localhost:8080` to any waiting clients which may
connect on port 3001 which is exposed to the host via the original sleepy
container whose network namespace we are sharing. It's possible to reverse the
roles or even not share a namespace and use Docker links for the container ->
server connection.

Now run the client on the host:

```
$ lennut -server-addr localhost:3001 -proxy-to localhost:8080
```

And finally we can test it be execing into our `sleepy` container and netcatting
back to the echo server.

```
$ docker exec -it sleepy nc localhost 8080
asd
asd
hello
hello
lennut
lennut
```

Clean up:

```
$ docker kill sleepy lennut
```

## Why?

The main use-case for this was integration testing a Go server that needed to
communicate with an external process which for reasons is easiest to run in a
docker container. However we also need the external process to be able to dial
back to the Go process under test (the test binary itself which runs an embedded
server that we are testing).

To add some fun, we need this integration test to run reliably and without lots
of complex setup on both linux and macOS dev environments assuming they have
docker installed.

Docker for Mac exposes `host.docker.internal` DNS name to easily call back
however for reasons it was not possible to use DNS to resolve this and we need a
portable IP. It's also advantageous to have the system under test "appear" to be
on localhost as that is a realistic real-world setup.

Possible alternatives: Unix sockets are out on docker-for-mac. SSH tunnels too
complex to setup. Similar proxies like localtunnel don't _quite_ work because
you don't know which port it will open on the "server" end (in docker) so can't
expose that to the host. We didn't want to setup the e2e tests such that the
system under test is cross compiled into a Docker container and then run and the
tests driven externally because it's too complicated to orchestrate.