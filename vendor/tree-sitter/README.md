# Tree-sitter grammars

Put prebuilt Emacs tree-sitter grammar libraries here for air-gapped use.

This directory is added to `treesit-extra-load-path` by `modules/16-languages.el`.
Do not download or build grammars during Emacs startup. Build them on an online
machine, commit the resulting platform-specific libraries, then import the repo
into the closed network.
