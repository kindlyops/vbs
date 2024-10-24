[![Gitpod ready-to-code](https://img.shields.io/badge/Gitpod-ready--to--code-blue?logo=gitpod)](https://gitpod.io/#https://github.com/kindlyops/vbs)

# Tools for video broadcast production

## IVS - OSC bridge

An Open Sound Control bridge for integrating Companion, QLab, and more with
the IVS PutMetadata API

## Lighting bridge

An bridge from a mobile web page to an instance of Companion for Streamdeck
running on your local network. Now remote people can press the same buttons
you have on your streamdeck!

## media Chapters

Generate OBS scenes from chapter markers for easier setup of run lists.

### Example of listing chapters from video file

```bash
vbs chapterlist file.mp4
```

### Example of splitting video file by chapters

```bash
vbs chaptersplit file.mp4
```

## installation for homebrew (MacOS/Linux)

    brew install kindlyops/tap/vbs

once installed, you can upgrade to a newer version using this command:

    brew upgrade kindlyops/tap/vbs

## installation for scoop (Windows Powershell)

To enable the bucket for your scoop installation

    scoop bucket add kindlyops https://github.com/kindlyops/kindlyops-scoop
    
To install deleterious

    scoop install vbs

once installed, you can upgrade to a newer version using this command:

    scoop status
    scoop update vbs

## installation from source

    go get github.com/kindlyops/vbs
    vbs help

## Developer instructions

Want to help add features or fix bugs? Awesome! vbs is build using bazel.

    `brew install bazelisk`
    grab the source code from github
    `bazel run vbs` to compile and run the locally compiled version

### Testing release process

To run goreleaser locally to test changes to the release process configuration:

    goreleaser release --snapshot --skip-publish --rm-dist

### Running container image locally

To run the container image locally to test changes to the container:

    bazel run vbs_image -- --publish=8090:8090 -- serve

To inspect the contents of the container using the dive utility

    bazel run vbs_image -- --norun
    dive bazel:vbs_image

### pushing an experimental version of the image to github container registry

To push a locally built image to github container registry:

    bazel run --define=IMAGE_TAG=experimental --stamp //:push_image
