# Release flow

This is a note for repo owner.

```bach
$ git checkout -b release/vX.Y.Z
# edit `var version` at speedtest/speedtest.go
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

update brew formula
https://github.com/showwin/homebrew-speedtest/blob/master/speedtest.rb#L3
