# Beltane

A [mayday](https://github.com/coreos/mayday) celebration.

Beltane is the ingestion service behind `mayday --upload`.

```
$ go build
$ ./beltane
```

In another terminal:

```
$ curl -X POST -F targz=@/tmp/mayday-tschuy-201609061411.617201601.tar.gz -F machine=`cat /etc/machine-id`  http://localhost:8080/upload

{
  "sha": "b7d4a48938d215a99e6c703a9b3d198c834db2f0",
  "access_url": "http://localhost:8080/dump/b7d4a48938d215a99e6c703a9b3d198c834db2f0"
}
```
