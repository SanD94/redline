local M = {}

M.ns = vim.api.nvim_create_namespace("redline")

function M.set_highlights(config)
  vim.api.nvim_set_hl(0, "RedlineAdd", config.highlights.add)
  vim.api.nvim_set_hl(0, "RedlineEdit", config.highlights.edit)
  vim.api.nvim_set_hl(0, "RedlineDelete", config.highlights.delete)
  vim.api.nvim_set_hl(0, "RedlineDeleteMarker", config.highlights.delete_marker)
end

function M.current_cursor_lnum()
  return vim.api.nvim_win_get_cursor(0)[1]
end

function M.should_show_deletion(mode, deletion_visibility, deletion)
  if mode ~= "redline" then
    return false
  end
  if deletion_visibility == "always" then
    return true
  end
  return deletion.lnum == M.current_cursor_lnum()
end

function M.view_state(views, bufnr)
  views[bufnr] = views[bufnr] or {}
  return views[bufnr]
end

function M.set_buffer_lines(bufnr, lines)
  local was_modifiable = vim.bo[bufnr].modifiable
  if not was_modifiable then
    vim.bo[bufnr].modifiable = true
  end

  vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)

  if not was_modifiable then
    vim.bo[bufnr].modifiable = false
  end
end

function M.save_new_view(views, bufnr)
  local view = M.view_state(views, bufnr)
  if view.active_mode == "old" then
    return view
  end

  view.new_lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
  view.modified = vim.bo[bufnr].modified
  view.modifiable = vim.bo[bufnr].modifiable
  view.readonly = vim.bo[bufnr].readonly
  return view
end

function M.restore_new_view(views, bufnr)
  local view = M.view_state(views, bufnr)
  if view.active_mode ~= "old" then
    return
  end

  vim.bo[bufnr].readonly = false
  vim.bo[bufnr].modifiable = true
  M.set_buffer_lines(bufnr, view.new_lines or {})
  vim.bo[bufnr].modified = view.modified or false
  vim.bo[bufnr].readonly = view.readonly or false
  vim.bo[bufnr].modifiable = view.modifiable ~= false
  view.active_mode = nil
end

local vcs_mod = nil
local function vcs()
  if not vcs_mod then
    vcs_mod = require("redline.vcs")
  end
  return vcs_mod
end

function M.show_old_view(views, config, bufnr)
  local name = vim.api.nvim_buf_get_name(bufnr)
  if name == "" then
    return false, "old mode needs a file-backed buffer"
  end

  M.save_new_view(views, bufnr)
  local lines, err = vcs().old_file_lines(config.vcs, name, config.old_revision)
  if not lines then
    return false, err or "failed to read old file version"
  end

  vim.api.nvim_buf_clear_namespace(bufnr, M.ns, 0, -1)
  vim.bo[bufnr].readonly = false
  vim.bo[bufnr].modifiable = true
  M.set_buffer_lines(bufnr, lines)
  vim.bo[bufnr].modified = views[bufnr].modified or false
  vim.bo[bufnr].readonly = true
  vim.bo[bufnr].modifiable = false
  views[bufnr].active_mode = "old"
  return true, nil
end

function M.render(views, config, enabled, by_buf, bufnr)
  if not vim.api.nvim_buf_is_valid(bufnr) then
    return
  end

  vim.api.nvim_buf_clear_namespace(bufnr, M.ns, 0, -1)

  if not enabled then
    return
  end

  local mode = config.mode

  if mode ~= "redline" then
    return
  end

  local changes = by_buf[bufnr]
  if not changes then
    return
  end

  for _, addition in ipairs(changes.additions) do
    pcall(vim.api.nvim_buf_set_extmark, bufnr, M.ns, addition.lnum - 1, addition.start_col, {
      end_row = addition.lnum - 1,
      end_col = addition.end_col,
      hl_group = addition.kind == "edit" and "RedlineEdit" or "RedlineAdd",
      priority = 200,
    })
  end

  for _, deletion in ipairs(changes.deletions) do
    if M.should_show_deletion(mode, config.deletion_visibility, deletion) then
      pcall(vim.api.nvim_buf_set_extmark, bufnr, M.ns, deletion.lnum - 1, deletion.col, {
        virt_text = {
          { config.deletion_prefix, "RedlineDeleteMarker" },
          { deletion.text, "RedlineDelete" },
          { config.deletion_suffix, "RedlineDeleteMarker" },
        },
        virt_text_pos = "inline",
        right_gravity = false,
        priority = 210,
      })
    end
  end
end

return M
