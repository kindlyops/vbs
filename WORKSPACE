
workspace(
    # How this workspace would be referenced with absolute labels from another workspace
    name = "com_kindlyops_vbs",
)

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# Keep the version of rules_go in sync with go.mod
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "0936c9bc3c4321ee372cb8f66dd972d368cb940ed01a9ba9fd7debcf0093f09b",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.51.0/rules_go-v0.51.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.51.0/rules_go-v0.51.0.zip",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(version = "1.22.8")

http_archive(
    name = "bazel_gazelle",
    sha256 = "e467b801046b6598c657309b45d2426dc03513777bd1092af2c62eebf990aca5",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.45.0/bazel-gazelle-v0.45.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.45.0/bazel-gazelle-v0.45.0.tar.gz",
    ],
)

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()


buildtools_version = "4.2.3"

http_archive(
    name = "io_bazel_buildtools",
    sha256 = "5ec71602e9b458b01717fab1d37492154c1c12ea83f881c745dbd88e9b2098d8",
    strip_prefix = "buildtools-{0}".format(buildtools_version),
    urls = ["https://github.com/bazelbuild/buildtools/archive/{0}.tar.gz".format(buildtools_version)],
)

http_archive(
    name = "bazel_skylib",
    urls = [
        "https://github.com/bazelbuild/bazel-skylib/releases/download/1.8.2/bazel-skylib-1.8.2.tar.gz",
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.8.2/bazel-skylib-1.8.2.tar.gz",
    ],
    sha256 = "6e78f0e57de26801f6f564fa7c4a48dc8b36873e416257a92bbb0937eeac8446",
)
load("@bazel_skylib//:workspace.bzl", "bazel_skylib_workspace")
bazel_skylib_workspace()

http_archive(
    name = "rules_pkg",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_pkg/releases/download/0.10.1/rules_pkg-0.10.1.tar.gz",
        "https://github.com/bazelbuild/rules_pkg/releases/download/0.10.1/rules_pkg-0.10.1.tar.gz",
    ],
    sha256 = "d250924a2ecc5176808fc4c25d5cf5e9e79e6346d79d5ab1c493e289e722d1d0",
)
load("@rules_pkg//:deps.bzl", "rules_pkg_dependencies")
rules_pkg_dependencies()


http_archive(
    name = "rules_oci",
    sha256 = "46ce9edcff4d3d7b3a550774b82396c0fa619cc9ce9da00c1b09a08b45ea5a14",
    strip_prefix = "rules_oci-1.8.0",
    url = "https://github.com/bazel-contrib/rules_oci/releases/download/v1.8.0/rules_oci-v1.8.0.tar.gz",
)

load("@rules_oci//oci:dependencies.bzl", "rules_oci_dependencies")

rules_oci_dependencies()

load("@rules_oci//oci:repositories.bzl", "LATEST_CRANE_VERSION", "oci_register_toolchains")

oci_register_toolchains(
    name = "oci",
    crane_version = LATEST_CRANE_VERSION,
)

load("@rules_oci//oci:pull.bzl", "oci_pull")

oci_pull(
    name = "distroless_static",
    digest = "sha256:87bce11be0af225e4ca761c40babb06d6d559f5767fbf7dc3c47f0f1a466b92c",
    image = "gcr.io/distroless/static",
    platforms = [
        "linux/amd64",
        "linux/arm64/v8",
    ],
)
