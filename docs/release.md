# Release flow

This is a note for repo owner.

```bach
$ git checkout -b release/vX.Y.Z
# edit `kingpin.Version("X.Y.Z")` at speedtest.go
$ git commit -am 'Release vX.Y.Z'
$ git push origin release/vX.Y.Z
```

merge PR

```bach
$ git checkout master
$ git pull origin master
$ git tag vX.Y.Z
$ git push origin vX.Y.Z
# run GitHub Action to build packages and make release.
```
