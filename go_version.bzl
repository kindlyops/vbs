"""Repository rule to read the Go version from go.mod."""

def _go_version_impl(repository_ctx):
    go_mod_content = repository_ctx.read(repository_ctx.attr.go_mod)
    go_version = ""
    for line in go_mod_content.split("\n"):
        line = line.strip()
        if line.startswith("go "):
            parts = line.split()
            if len(parts) >= 2:
                go_version = parts[1]
                break

    if not go_version:
        fail("Could not find 'go' directive in go.mod")

    # go_register_toolchains requires a full major.minor.patch version
    if go_version.count(".") < 2:
        fail("go.mod version '{}' must include a patch version (e.g. 1.21.13) for Bazel toolchain registration".format(go_version))

    repository_ctx.file("BUILD.bazel", "")
    repository_ctx.file(
        "def.bzl",
        'GO_VERSION = "{}"\n'.format(go_version),
    )

go_version = repository_rule(
    implementation = _go_version_impl,
    attrs = {
        "go_mod": attr.label(
            default = "//:go.mod",
            allow_single_file = True,
        ),
    },
)
