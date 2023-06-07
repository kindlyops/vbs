
workspace(
    # How this workspace would be referenced with absolute labels from another workspace
    name = "com_kindlyops_vbs",
)

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# Keep the version of rules_go in sync with go.mod
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "6b65cb7917b4d1709f9410ffe00ecf3e160edf674b78c54a894471320862184f",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.39.0/rules_go-v0.39.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.39.1/rules_go-v0.39.0.zip",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(version = "1.18.4")

http_archive(
    name = "bazel_gazelle",
    sha256 = "ecba0f04f96b4960a5b250c8e8eeec42281035970aa8852dda73098274d14a1d",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.29.0/bazel-gazelle-v0.29.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.29.0/bazel-gazelle-v0.29.0.tar.gz",
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
        "https://github.com/bazelbuild/bazel-skylib/releases/download/1.4.2/bazel-skylib-1.4.2.tar.gz",
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.4.2/bazel-skylib-1.4.2.tar.gz",
    ],
    sha256 = "66ffd9315665bfaafc96b52278f57c7e2dd09f5ede279ea6d39b2be471e7e3aa",
)
load("@bazel_skylib//:workspace.bzl", "bazel_skylib_workspace")
bazel_skylib_workspace()

http_archive(
    name = "build_bazel_rules_nodejs",
    sha256 = "c911b5bd8aee8b0498cc387cacdb5f917098ce477fb4182db07b0ef8a9e045c0",
    urls = ["https://github.com/bazelbuild/rules_nodejs/releases/download/4.7.1/rules_nodejs-4.7.1.tar.gz"],
)


load("@build_bazel_rules_nodejs//:index.bzl", "node_repositories", "yarn_install")
# NOTE: this rule installs nodejs, npm, and yarn, but does NOT install
# your npm dependencies into your node_modules folder.
# You must still run the package manager to do this.
# M1 Macs require Node 16+
node_repositories(
    package_json = ["//embeddy:package.json"],
    node_version = "16.13.0",
)

# Setup Bazel managed npm dependencies with the `yarn_install` rule.
# The name of this rule should be set to `npm` so that `ts_library` and `ts_web_test_suite`
# can find your npm dependencies by default in the `@npm` workspace. You may
# also use the `npm_install` rule with a `package-lock.json` file if you prefer.
# See https://github.com/bazelbuild/rules_nodejs#dependencies for more info.
yarn_install(
  name = "npm",
  package_json = "//embeddy:package.json",
  quiet = False,
  yarn_lock = "//embeddy:yarn.lock",
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "b1e80761a8a8243d03ebca8845e9cc1ba6c82ce7c5179ce2b295cd36f7e394bf",
    urls = ["https://github.com/bazelbuild/rules_docker/releases/download/v0.25.0/rules_docker-v0.25.0.tar.gz"],
)

# OPTIONAL: Call this to override the default docker toolchain configuration.
# This call should be placed BEFORE the call to "container_repositories" below
# to actually override the default toolchain configuration.
# Note this is only required if you actually want to call
# docker_toolchain_configure with a custom attr; please read the toolchains
# docs in /toolchains/docker/ before blindly adding this to your WORKSPACE.
# BEGIN OPTIONAL segment:
#load("@io_bazel_rules_docker//toolchains/docker:toolchain.bzl",
#    docker_toolchain_configure="toolchain_configure"
#)
# docker_toolchain_configure(
#   name = "docker_config",
#   # OPTIONAL: Bazel target for the build_tar tool, must be compatible with build_tar.py
#   build_tar_target="<enter absolute path (i.e., must start with repo name @...//:...) to an executable build_tar target>",
#   # OPTIONAL: Path to a directory which has a custom docker client config.json.
#   # See https://docs.docker.com/engine/reference/commandline/cli/#configuration-files
#   # for more details.
#   client_config="<enter Bazel label to your docker config.json here>",
#   # OPTIONAL: Path to the docker binary.
#   # Should be set explicitly for remote execution.
#   docker_path="<enter absolute path to the docker binary (in the remote exec env) here>",
#   # OPTIONAL: Path to the gzip binary.
#   gzip_path="<enter absolute path to the gzip binary (in the remote exec env) here>",
#   # OPTIONAL: Bazel target for the gzip tool.
#   gzip_target="<enter absolute path (i.e., must start with repo name @...//:...) to an executable gzip target>",
#   # OPTIONAL: Path to the xz binary.
#   # Should be set explicitly for remote execution.
#   xz_path="<enter absolute path to the xz binary (in the remote exec env) here>",
#   # OPTIONAL: Bazel target for the xz tool.
#   # Either xz_path or xz_target should be set explicitly for remote execution.
#   xz_target="<enter absolute path (i.e., must start with repo name @...//:...) to an executable xz target>",
#   # OPTIONAL: List of additional flags to pass to the docker command.
#   docker_flags = [
#     "--tls",
#     "--log-level=info",
#   ],

# )
# End of OPTIONAL segment.

load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)
container_repositories()

load("@io_bazel_rules_docker//repositories:deps.bzl", container_deps = "deps")

container_deps()

load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_pull",
)
load(
    "@io_bazel_rules_docker//go:image.bzl",
    _go_image_repos = "repositories",
)

_go_image_repos()

container_pull(
  name = "static_base",
  registry = "gcr.io",
  repository = "distroless/static",
  # 'tag' is also supported, but digest is encouraged for reproducibility.
  digest = "sha256:d1d4a57d06e3c59f71cd1d72d894ab2a3c17973684d42348fbe84c1396fb4b41",
)
