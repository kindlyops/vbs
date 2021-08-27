
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
            cfg = "host",
        ),
    },
    outputs = {
        "embedder": "embedder.go",
    },
)
