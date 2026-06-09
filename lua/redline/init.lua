local diff = require("redline.diff")
local render = require("redline.render")
local vcs = require("redline.vcs")

local M = {}

local defaults = {
  mode = "redline",
  vcs = "jj",
  old_revision = "@-",
  deletion_visibility = "cursor",
  deletion_prefix = "[- ",
  deletion_suffix = " -]",
  highlights = {
    add = { fg = "#c8d3f5", bg = "#2a4556" },
    edit = { fg = "#c8d3f5", bg = "#394b70" },
    delete = { fg = "#e26a75", bg = "#4b2a3d", strikethrough = true },
    delete_marker = { fg = "#828bb8", bg = "#4b2a3d" },
  },
}

local state = {
  config = vim.deepcopy(defaults),
  enabled = false,
  by_buf = {},
  views = {},
}

local function notify(msg, level)
  vim.notify(msg, level or vim.log.levels.INFO, { title = "redline.nvim" })
end

function M.refresh(bufnr)
  bufnr = bufnr or vim.api.nvim_get_current_buf()
  local view = render.view_state(state.views, bufnr)
  if view.active_mode ~= "old" then
    render.save_new_view(state.views, bufnr)
  end

  local name = vim.api.nvim_buf_get_name(bufnr)
  if name == "" then
    state.by_buf[bufnr] = nil
    render.render(state.views, state.config, state.enabled, state.by_buf, bufnr)
    return
  end

  local lines, err = vcs.system_lines(vcs.diff_command(state.config.vcs, name))
  if not lines then
    notify(err or "failed to read diff", vim.log.levels.WARN)
    return
  end

  state.by_buf[bufnr] = diff.hunks_to_changes(diff.parse_diff(lines))
  render.render(state.views, state.config, state.enabled, state.by_buf, bufnr)
end

local function apply_mode(bufnr, mode)
  if mode == "old" then
    local ok, err = render.show_old_view(state.views, state.config, bufnr)
    if not ok then
      notify(err, vim.log.levels.WARN)
    end
    return ok
  end

  render.restore_new_view(state.views, bufnr)
  vim.api.nvim_buf_clear_namespace(bufnr, render.ns, 0, -1)

  if mode == "redline" then
    M.refresh(bufnr)
  end

  return true
end

local function kind_matches(change_kind, filter_kind)
  return filter_kind == nil or filter_kind == "" or change_kind == filter_kind
end

local function change_targets(changes, filter_kind)
  local targets = {}

  for _, addition in ipairs(changes.additions) do
    if kind_matches(addition.kind, filter_kind) then
      table.insert(targets, {
        lnum = addition.lnum,
        col = addition.start_col,
      })
    end
  end

  for _, deletion in ipairs(changes.deletions) do
    if kind_matches(deletion.kind, filter_kind) then
      table.insert(targets, {
        lnum = deletion.lnum,
        col = deletion.col,
      })
    end
  end

  table.sort(targets, function(a, b)
    if a.lnum == b.lnum then
      return a.col < b.col
    end
    return a.lnum < b.lnum
  end)

  local unique = {}
  local last
  for _, target in ipairs(targets) do
    if not last or target.lnum ~= last.lnum or target.col ~= last.col then
      table.insert(unique, target)
      last = target
    end
  end

  return unique
end

local function ensure_changes(bufnr)
  local view = state.views[bufnr]
  if view and view.active_mode == "old" then
    notify("change motions use the current file view; switch to redline or new mode first", vim.log.levels.WARN)
    return nil
  end

  if not state.by_buf[bufnr] then
    M.refresh(bufnr)
  end

  return state.by_buf[bufnr]
end

local function compare_position(a_lnum, a_col, b_lnum, b_col)
  if a_lnum == b_lnum then
    return a_col - b_col
  end
  return a_lnum - b_lnum
end

local function jump_to_target(bufnr, target)
  local line_count = vim.api.nvim_buf_line_count(bufnr)
  local lnum = math.min(math.max(target.lnum, 1), line_count)
  local line = vim.api.nvim_buf_get_lines(bufnr, lnum - 1, lnum, false)[1] or ""
  local col = math.min(math.max(target.col, 0), #line)

  vim.cmd("normal! m'")
  vim.api.nvim_win_set_cursor(0, { lnum, col })
  if state.enabled and state.config.mode == "redline" then
    render.render(state.views, state.config, state.enabled, state.by_buf, bufnr)
  end
end

local function jump_change(direction, filter_kind)
  if filter_kind ~= nil and filter_kind ~= "" and filter_kind ~= "add" and filter_kind ~= "edit" and filter_kind ~= "delete" then
    notify("change kind must be one of: add, edit, delete", vim.log.levels.ERROR)
    return
  end

  local bufnr = vim.api.nvim_get_current_buf()
  local changes = ensure_changes(bufnr)
  if not changes then
    notify("no redline data for this buffer")
    return
  end

  local targets = change_targets(changes, filter_kind)
  if #targets == 0 then
    local has_filter = filter_kind ~= nil and filter_kind ~= ""
    notify(has_filter and ("no " .. filter_kind .. " redline changes") or "no redline changes")
    return
  end

  local cursor = vim.api.nvim_win_get_cursor(0)
  local cursor_lnum = cursor[1]
  local cursor_col = cursor[2]
  local selected = direction == "previous" and targets[#targets] or targets[1]

  if direction == "previous" then
    for i = #targets, 1, -1 do
      local target = targets[i]
      if compare_position(target.lnum, target.col, cursor_lnum, cursor_col) < 0 then
        selected = target
        break
      end
    end
  else
    for _, target in ipairs(targets) do
      if compare_position(target.lnum, target.col, cursor_lnum, cursor_col) > 0 then
        selected = target
        break
      end
    end
  end

  jump_to_target(bufnr, selected)
end

function M.set_mode(mode)
  if mode ~= "old" and mode ~= "redline" and mode ~= "new" then
    notify("mode must be one of: old, redline, new", vim.log.levels.ERROR)
    return
  end

  state.config.mode = mode
  state.enabled = true
  apply_mode(vim.api.nvim_get_current_buf(), mode)
end

function M.enable()
  state.enabled = true
  apply_mode(vim.api.nvim_get_current_buf(), state.config.mode)
end

function M.disable()
  local bufnr = vim.api.nvim_get_current_buf()
  render.restore_new_view(state.views, bufnr)
  state.enabled = false
  vim.api.nvim_buf_clear_namespace(bufnr, render.ns, 0, -1)
end

function M.toggle()
  if state.enabled then
    M.disable()
  else
    M.enable()
  end
end

function M.next_change(kind)
  jump_change("next", kind)
end

function M.prev_change(kind)
  jump_change("previous", kind)
end

function M.show_change()
  local bufnr = vim.api.nvim_get_current_buf()
  local changes = state.by_buf[bufnr]
  if not changes then
    notify("no redline data for this buffer")
    return
  end

  local lnum = render.current_cursor_lnum()
  local lines = {}

  for _, deletion in ipairs(changes.deletions) do
    if deletion.lnum == lnum then
      table.insert(lines, "- " .. deletion.text)
    end
  end
  for _, addition in ipairs(changes.additions) do
    if addition.lnum == lnum then
      local line = vim.api.nvim_buf_get_lines(bufnr, addition.lnum - 1, addition.lnum, false)[1] or ""
      table.insert(lines, "+ " .. line:sub(addition.start_col + 1, addition.end_col))
    end
  end

  if #lines == 0 then
    notify("no redline change on this line")
    return
  end

  local width = 0
  for _, line in ipairs(lines) do
    width = math.max(width, vim.fn.strdisplaywidth(line))
  end

  local popup = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_buf_set_lines(popup, 0, -1, false, lines)
  vim.api.nvim_open_win(popup, false, {
    relative = "cursor",
    row = 1,
    col = 0,
    width = math.min(math.max(width, 20), math.floor(vim.o.columns * 0.8)),
    height = #lines,
    border = "rounded",
    style = "minimal",
  })
end

function M.setup(opts)
  state.config = vim.tbl_deep_extend("force", vim.deepcopy(defaults), opts or {})
  render.set_highlights(state.config)

  local group = vim.api.nvim_create_augroup("redline_nvim", { clear = true })

  vim.api.nvim_create_autocmd({ "CursorMoved", "CursorMovedI" }, {
    group = group,
    callback = function(args)
      if state.enabled and state.config.mode == "redline" and state.config.deletion_visibility == "cursor" then
        render.render(state.views, state.config, state.enabled, state.by_buf, args.buf)
      end
    end,
  })

  vim.api.nvim_create_autocmd({ "BufWritePre" }, {
    group = group,
    callback = function(args)
      local view = state.views[args.buf]
      if view and view.active_mode == "old" then
        error("Redline old mode is read-only; switch to :RedlineMode new or :RedlineMode redline before writing")
      end
    end,
  })

  vim.api.nvim_create_autocmd({ "BufWritePost" }, {
    group = group,
    callback = function(args)
      local view = state.views[args.buf]
      if state.enabled and not (view and view.active_mode == "old") then
        M.refresh(args.buf)
      end
    end,
  })
end

return M
