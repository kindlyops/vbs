"""
Bazel macro for building an embedded NextJS app into a go library.
"""
load("@npm//next:index.bzl", "next")
load("@build_bazel_rules_nodejs//:index.bzl", "copy_to_bin")

def _static_site_embedder_impl(ctx):
    #tree = ctx.actions.declare_directory(ctx.attr.name + ".artifacts")
    args = [str(ctx.outputs.embedder.path)] + [f.path for f in ctx.files.srcs]

    ctx.actions.run(
        inputs = ctx.files.srcs,
        arguments = args,
        outputs = [ctx.outputs.embedder],
        progress_message = "Generating %s from %s" % (str(ctx.outputs.embedder.path), str(ctx.files.srcs)),
        executable = ctx.executable._manifester,
    )
    return [DefaultInfo(files = depset([ctx.outputs.embedder]))]

static_site_embedder = rule(
    implementation = _static_site_embedder_impl,
    doc = """
Builds a static website with manifest file.
This is useful for collecting together the generated files from a static
site build and embedding or publishing.
""",
    attrs = {
        "srcs": attr.label_list(allow_files = True),
        "_manifester": attr.label(
            default = Label("//sitemanifest:manifester"),
            allow_single_file = True,
            executable = True,
            cfg = "exec",
        ),
    },
    outputs = {
        "embedder": "embedder.go",
    },
)

def embed_nextjs(name, srcs = [], visibility=None, **kwargs):
    """
    Embeds a static site into a go library.

    This is useful for collecting together the generated files from a static
    site build and embedding or publishing.

    Args:
        name: Name of the embedder.
        srcs: List of files to embed.
        visibility: Visibility of the embedder.
        **kwargs: Additional arguments to pass to the embedder rule.

    Returns:
        A label pointing to the embedder.
    """
    copy_to_bin(
        name = "copy_source_files",
        srcs = srcs,
        visibility = ["//visibility:private"],
    )

    next(
        name = "next_build",
        outs = [".next/build-manifest.json"],
        args = ["build $(RULEDIR)"],
        data = [":copy_source_files"],  # + NPM_DEPENDENCIES,
        # tags = ["no-sandbox"],
        visibility = ["//visibility:private"],
    )

    next(
        name = "next_export",
        outs = ["dist"],
        args = [
            "export $(RULEDIR)",
            "-o $(@)",
        ],
        data = [":next_build"],
        visibility = ["//visibility:private"],
    )

    return static_site_embedder(
        name = name,
        srcs = [":dist"],
        visibility = visibility,
        **kwargs
    )
