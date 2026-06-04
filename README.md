# redline.nvim

A tiny Neovim proof-of-concept for viewing word-level changes from `jj diff` inside long prose paragraphs.

The plugin has three view modes:

1. `old` — parent/last-commit file content, with no redline effects.
2. `redline` — current buffer plus redline markup: added spans are bold+italic; deleted text is hidden until the cursor is on that line.
3. `new` — current edited version, with no redline effects.

## Install from this checkout

With `lazy.nvim`, add this to your custom Neovim config:

```lua
{
  dir = "{project-location}/redline", 
  name = "redline.nvim",
  config = function()
    require("redline").setup({
      vcs = "jj", -- or "git"
      mode = "redline",
      deletion_visibility = "cursor",
      highlights = {
        add = { fg = "#c8d3f5", bg = "#2a4556" },
        edit = { fg = "#c8d3f5", bg = "#394b70" },
        delete = { fg = "#e26a75", bg = "#4b2a3d", strikethrough = true },
        delete_marker = { fg = "#828bb8", bg = "#4b2a3d" },
      },
    })
  end,
}
```

Or with built-in packages:

```sh
mkdir -p ~/.config/nvim/pack/local/start
ln -s {project-location}/redline ~/.config/nvim/pack/local/start/redline.nvim
```

Then in Neovim:

```vim
:RedlineEnable
:RedlineMode old
:RedlineMode redline
:RedlineMode new
:RedlineNext
:RedlinePrev
:RedlineShow
:RedlineRefresh
:RedlineDisable
```

`:RedlineNext` and `:RedlinePrev` jump between changed spans in the current file and wrap at the ends. They accept an optional kind filter: `add`, `edit`, or `delete`.

Suggested mappings:

```lua
vim.keymap.set("n", "<leader>rt", "<cmd>RedlineToggle<cr>")
vim.keymap.set("n", "<leader>ro", "<cmd>RedlineMode old<cr>")
vim.keymap.set("n", "<leader>rr", "<cmd>RedlineMode redline<cr>")
vim.keymap.set("n", "<leader>rn", "<cmd>RedlineMode new<cr>")
vim.keymap.set("n", "]r", "<cmd>RedlineNext<cr>")
vim.keymap.set("n", "[r", "<cmd>RedlinePrev<cr>")
vim.keymap.set("n", "<leader>rs", "<cmd>RedlineShow<cr>")
```

## Display style

- Old mode: swaps the visible buffer to the parent revision and makes that view read-only.
- Added text: existing buffer text gets a muted teal background (`#2a4556`).
- Deleted text: virtual inline text is hidden by default, then revealed with muted wine background/red foreground when the cursor is on that line.
- Edited text: old deleted text appears virtually near the edit; new replacement text gets a muted slate-blue background (`#394b70`).
- New mode: restores the current edited buffer and clears all overlays.

## Notes

This is intentionally a PoC:

- It parses unified/git-style patches from `jj diff --git`.
- `old` mode reads `jj file show -r @-` by default; for `vcs = "git"`, it reads `git show HEAD:<path>`.
- Single-line replacements get word-level spans.
- Multi-line changes fall back to line-level spans.
- `old` mode is a non-destructive read-only preview; switch back to `new` or `redline` before writing.
