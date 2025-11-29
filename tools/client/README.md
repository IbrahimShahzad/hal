# Client


In order to send messages to the server, run the following

```bash
cd tools/client
go build . -o hal-client
./hal-cient --help
./hal-cleint -addr localhost:8080 -token myuperduperstrongandmightypassword -m "Hello, HAL!" -t tag1,tag2
```

If the token is not provided, the client will look for the `X-Auth-Token` environment variable.

In order to run from any location

```bash
mv hal-client /usr/local/bin/hal-client
```

> Note: Will try to upgrade the client to be a TUI...
