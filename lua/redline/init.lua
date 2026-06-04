local M = {}

local ns = vim.api.nvim_create_namespace("redline")

local defaults = {
  mode = "redline", -- "old", "redline", "new"
  vcs = "jj", -- "jj" or "git"
  old_revision = "@-", -- jj revision for old mode; git uses HEAD
  deletion_visibility = "cursor", -- "cursor" or "always" in redline mode
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

local function set_highlights()
  vim.api.nvim_set_hl(0, "RedlineAdd", state.config.highlights.add)
  vim.api.nvim_set_hl(0, "RedlineEdit", state.config.highlights.edit)
  vim.api.nvim_set_hl(0, "RedlineDelete", state.config.highlights.delete)
  vim.api.nvim_set_hl(0, "RedlineDeleteMarker", state.config.highlights.delete_marker)
end

local function system_lines(cmd)
  local out = vim.fn.systemlist(cmd)
  if vim.v.shell_error ~= 0 then
    return nil, table.concat(out, "\n")
  end
  return out, nil
end

local function project_root(vcs, path)
  local path_dir = vim.fs.dirname(path)
  local commands
  if vcs == "git" then
    commands = {
      { "git", "-C", path_dir, "rev-parse", "--show-toplevel" },
      { "jj", "-R", path_dir, "root" },
    }
  else
    commands = {
      { "jj", "-R", path_dir, "root" },
      { "git", "-C", path_dir, "rev-parse", "--show-toplevel" },
    }
  end

  for _, command in ipairs(commands) do
    local out = system_lines(command)
    if out and out[1] and out[1] ~= "" then
      return out[1]
    end
  end

  return vim.loop.cwd()
end

local function relpath(path, root)
  local normalized_root = vim.fs.normalize(root)
  local normalized_path = vim.fs.normalize(path)
  if vim.startswith(normalized_path, normalized_root .. "/") then
    return normalized_path:sub(#normalized_root + 2)
  end
  return normalized_path
end

local function diff_command(path)
  local root = project_root(state.config.vcs, path)
  local relative = relpath(path, root)

  if state.config.vcs == "git" then
    return { "git", "-C", root, "diff", "--", relative }
  end

  return { "jj", "-R", root, "diff", "--git", "--", "root:" .. vim.fn.fnameescape(relative) }
end

local function old_file_command(path)
  local root = project_root(state.config.vcs, path)
  local relative = relpath(path, root)

  if state.config.vcs == "git" then
    return { "git", "-C", root, "show", "HEAD:" .. relative }
  end

  return { "jj", "-R", root, "file", "show", "-r", state.config.old_revision, "root:" .. vim.fn.fnameescape(relative) }
end

local function old_file_lines(path)
  return system_lines(old_file_command(path))
end

local function parse_hunk_header(line)
  local old_start, old_count, new_start, new_count = line:match("^@@ %-(%d+),?(%d*) %+(%d+),?(%d*) @@")
  if not old_start then
    return nil
  end
  return {
    old_start = tonumber(old_start),
    old_count = tonumber(old_count ~= "" and old_count or "1"),
    new_start = tonumber(new_start),
    new_count = tonumber(new_count ~= "" and new_count or "1"),
    lines = {},
  }
end

local function parse_diff(lines)
  local hunks = {}
  local current = nil

  for _, line in ipairs(lines) do
    local hunk = parse_hunk_header(line)
    if hunk then
      current = hunk
      table.insert(hunks, current)
    elseif current then
      local prefix = line:sub(1, 1)
      if prefix == " " or prefix == "+" or prefix == "-" then
        table.insert(current.lines, { kind = prefix, text = line:sub(2) })
      elseif line:sub(1, 1) == "\\" then
        -- "No newline at end of file" marker: ignore.
      end
    end
  end

  return hunks
end

local function tokenize_words(text)
  local tokens = {}
  local pos = 1
  while pos <= #text do
    local start_col, end_col = text:find("%S+", pos)
    if not start_col then
      break
    end
    table.insert(tokens, {
      text = text:sub(start_col, end_col),
      start_col = start_col - 1,
      end_col = end_col,
    })
    pos = end_col + 1
  end
  return tokens
end

local function lcs_matches(old_tokens, new_tokens)
  local n = #old_tokens
  local m = #new_tokens
  local dp = {}

  for i = 0, n do
    dp[i] = {}
    for j = 0, m do
      dp[i][j] = 0
    end
  end

  for i = n - 1, 0, -1 do
    for j = m - 1, 0, -1 do
      if old_tokens[i + 1].text == new_tokens[j + 1].text then
        dp[i][j] = dp[i + 1][j + 1] + 1
      else
        dp[i][j] = math.max(dp[i + 1][j], dp[i][j + 1])
      end
    end
  end

  local matches = {}
  local i, j = 1, 1
  while i <= n and j <= m do
    if old_tokens[i].text == new_tokens[j].text then
      table.insert(matches, { old = i, new = j })
      i = i + 1
      j = j + 1
    elseif dp[i][j - 1] >= dp[i - 1][j] then
      i = i + 1
    else
      j = j + 1
    end
  end

  return matches
end

local function add_addition_span(changes, lnum, new_text, first_token, last_token, kind)
  if not first_token or not last_token then
    return
  end

  local start_col = first_token.start_col
  local end_col = last_token.end_col

  -- Include nearby spaces for pure insertions without swallowing unchanged words.
  if kind ~= "edit" then
    while start_col > 0 and new_text:sub(start_col, start_col):match("%s") do
      start_col = start_col - 1
    end
    while end_col < #new_text and new_text:sub(end_col + 1, end_col + 1):match("%s") do
      end_col = end_col + 1
    end
  end

  table.insert(changes.additions, {
    lnum = lnum,
    start_col = start_col,
    end_col = end_col,
    kind = kind or "add",
  })
end

local function add_deletion(changes, lnum, col, text, kind)
  if text == nil or text == "" then
    return
  end
  table.insert(changes.deletions, {
    lnum = math.max(lnum, 1),
    col = math.max(col or 0, 0),
    text = text,
    kind = kind or "delete",
  })
end

local function word_diff(changes, lnum, old_text, new_text, kind)
  local old_tokens = tokenize_words(old_text)
  local new_tokens = tokenize_words(new_text)
  kind = kind or "edit"

  if #old_tokens == 0 and #new_tokens == 0 then
    return
  end
  if #old_tokens == 0 then
    table.insert(changes.additions, { lnum = lnum, start_col = 0, end_col = #new_text, kind = "add" })
    return
  end
  if #new_tokens == 0 then
    add_deletion(changes, lnum, 0, old_text, "delete")
    return
  end

  local matches = lcs_matches(old_tokens, new_tokens)
  table.insert(matches, { old = #old_tokens + 1, new = #new_tokens + 1 })

  local old_i, new_i = 1, 1
  for _, match in ipairs(matches) do
    if new_i < match.new then
      add_addition_span(changes, lnum, new_text, new_tokens[new_i], new_tokens[match.new - 1], kind)
    end

    if old_i < match.old then
      local first_old = old_tokens[old_i]
      local last_old = old_tokens[match.old - 1]
      local deleted = old_text:sub(first_old.start_col + 1, last_old.end_col)
      local anchor_col = 0
      if new_i <= #new_tokens then
        anchor_col = new_tokens[new_i].start_col
      elseif match.new <= #new_tokens then
        anchor_col = new_tokens[match.new].start_col
      elseif #new_tokens > 0 then
        anchor_col = new_tokens[#new_tokens].end_col
      end
      add_deletion(changes, lnum, anchor_col, deleted, kind)
    end

    old_i = match.old + 1
    new_i = match.new + 1
  end
end

local function deletion_text(text)
  return text ~= "" and text or "⏎"
end

local function collapse_lines(lines)
  local text = {}
  for _, line in ipairs(lines) do
    table.insert(text, deletion_text(line.text))
  end
  return table.concat(text, " / ")
end

local function line_similarity(old_text, new_text)
  local old_tokens = tokenize_words(old_text)
  local new_tokens = tokenize_words(new_text)
  local longest = math.max(#old_tokens, #new_tokens)
  if longest == 0 then
    return 1
  end
  return #lcs_matches(old_tokens, new_tokens) / longest
end

local function add_line_span(changes, lnum, text, kind)
  table.insert(changes.additions, {
    lnum = lnum,
    start_col = 0,
    end_col = #text,
    kind = kind,
  })
end

local function pair_similar_lines(removed, added)
  local pairs = {}
  local next_added = 1
  local minimum_similarity = 0.25

  for removed_index, removed_line in ipairs(removed) do
    local best_added = nil
    local best_score = 0

    for added_index = next_added, #added do
      local score = line_similarity(removed_line.text, added[added_index].text)
      if score > best_score then
        best_score = score
        best_added = added_index
      end
    end

    if best_added and best_score >= minimum_similarity then
      pairs[removed_index] = best_added
      next_added = best_added + 1
    end
  end

  return pairs
end

local function process_paired_change_group(changes, new_lnum, removed, added, line_pairs)
  local removed_by_added = {}
  local paired_removed = {}

  for removed_index, added_index in pairs(line_pairs) do
    removed_by_added[added_index] = removed_index
    paired_removed[removed_index] = true
  end

  for added_index, line in ipairs(added) do
    local removed_index = removed_by_added[added_index]
    if removed_index then
      word_diff(changes, new_lnum + added_index - 1, removed[removed_index].text, line.text, "edit")
    else
      add_line_span(changes, new_lnum + added_index - 1, line.text, "add")
    end
  end

  for removed_index, line in ipairs(removed) do
    if not paired_removed[removed_index] then
      local anchor_lnum = new_lnum

      for paired_removed_index = removed_index - 1, 1, -1 do
        local added_index = line_pairs[paired_removed_index]
        if added_index then
          anchor_lnum = new_lnum + added_index
          break
        end
      end

      if anchor_lnum == new_lnum then
        for paired_removed_index = removed_index + 1, #removed do
          local added_index = line_pairs[paired_removed_index]
          if added_index then
            anchor_lnum = new_lnum + added_index - 1
            break
          end
        end
      end

      add_deletion(changes, anchor_lnum, 0, deletion_text(line.text), #added > 0 and "edit" or "delete")
    end
  end
end

local function process_change_group(changes, new_lnum, removed, added)
  if #removed == 1 and #added == 1 then
    word_diff(changes, new_lnum, removed[1].text, added[1].text, "edit")
    return
  end

  if #removed > 0 and #added > 0 then
    local pairs = pair_similar_lines(removed, added)
    if next(pairs) ~= nil then
      process_paired_change_group(changes, new_lnum, removed, added, pairs)
      return
    end
  end

  if #removed > 0 then
    add_deletion(changes, new_lnum, 0, collapse_lines(removed), #added > 0 and "edit" or "delete")
  end

  for offset, line in ipairs(added) do
    add_line_span(changes, new_lnum + offset - 1, line.text, #removed > 0 and "edit" or "add")
  end
end

local function hunks_to_changes(hunks)
  local changes = { additions = {}, deletions = {} }

  for _, hunk in ipairs(hunks) do
    local new_lnum = hunk.new_start
    local i = 1

    while i <= #hunk.lines do
      local line = hunk.lines[i]
      if line.kind == " " then
        new_lnum = new_lnum + 1
        i = i + 1
      elseif line.kind == "-" or line.kind == "+" then
        local removed = {}
        local added = {}
        local group_new_lnum = new_lnum

        while i <= #hunk.lines and hunk.lines[i].kind ~= " " do
          if hunk.lines[i].kind == "-" then
            table.insert(removed, hunk.lines[i])
          elseif hunk.lines[i].kind == "+" then
            table.insert(added, hunk.lines[i])
            new_lnum = new_lnum + 1
          end
          i = i + 1
        end

        process_change_group(changes, group_new_lnum, removed, added)
      else
        i = i + 1
      end
    end
  end

  return changes
end

local function current_cursor_lnum()
  return vim.api.nvim_win_get_cursor(0)[1]
end

local function should_show_deletion(mode, deletion)
  if mode ~= "redline" then
    return false
  end
  if state.config.deletion_visibility == "always" then
    return true
  end
  return deletion.lnum == current_cursor_lnum()
end

local function view_state(bufnr)
  state.views[bufnr] = state.views[bufnr] or {}
  return state.views[bufnr]
end

local function set_buffer_lines(bufnr, lines)
  local was_modifiable = vim.bo[bufnr].modifiable
  if not was_modifiable then
    vim.bo[bufnr].modifiable = true
  end

  vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)

  if not was_modifiable then
    vim.bo[bufnr].modifiable = false
  end
end

local function save_new_view(bufnr)
  local view = view_state(bufnr)
  if view.active_mode == "old" then
    return view
  end

  view.new_lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
  view.modified = vim.bo[bufnr].modified
  view.modifiable = vim.bo[bufnr].modifiable
  view.readonly = vim.bo[bufnr].readonly
  return view
end

local function restore_new_view(bufnr)
  local view = view_state(bufnr)
  if view.active_mode ~= "old" then
    return
  end

  vim.bo[bufnr].readonly = false
  vim.bo[bufnr].modifiable = true
  set_buffer_lines(bufnr, view.new_lines or {})
  vim.bo[bufnr].modified = view.modified or false
  vim.bo[bufnr].readonly = view.readonly or false
  vim.bo[bufnr].modifiable = view.modifiable ~= false
  view.active_mode = nil
end

local function show_old_view(bufnr)
  local name = vim.api.nvim_buf_get_name(bufnr)
  if name == "" then
    notify("old mode needs a file-backed buffer", vim.log.levels.WARN)
    return false
  end

  local view = save_new_view(bufnr)
  local lines, err = old_file_lines(name)
  if not lines then
    notify(err or "failed to read old file version", vim.log.levels.WARN)
    return false
  end

  vim.api.nvim_buf_clear_namespace(bufnr, ns, 0, -1)
  vim.bo[bufnr].readonly = false
  vim.bo[bufnr].modifiable = true
  set_buffer_lines(bufnr, lines)
  vim.bo[bufnr].modified = view.modified or false
  vim.bo[bufnr].readonly = true
  vim.bo[bufnr].modifiable = false
  view.active_mode = "old"
  return true
end

local function render(bufnr)
  if not vim.api.nvim_buf_is_valid(bufnr) then
    return
  end

  vim.api.nvim_buf_clear_namespace(bufnr, ns, 0, -1)

  if not state.enabled then
    return
  end

  local mode = state.config.mode

  if mode ~= "redline" then
    return
  end

  local changes = state.by_buf[bufnr]
  if not changes then
    return
  end

  for _, addition in ipairs(changes.additions) do
    pcall(vim.api.nvim_buf_set_extmark, bufnr, ns, addition.lnum - 1, addition.start_col, {
      end_row = addition.lnum - 1,
      end_col = addition.end_col,
      hl_group = addition.kind == "edit" and "RedlineEdit" or "RedlineAdd",
      priority = 200,
    })
  end

  for _, deletion in ipairs(changes.deletions) do
    if should_show_deletion(mode, deletion) then
      pcall(vim.api.nvim_buf_set_extmark, bufnr, ns, deletion.lnum - 1, deletion.col, {
        virt_text = {
          { state.config.deletion_prefix, "RedlineDeleteMarker" },
          { deletion.text, "RedlineDelete" },
          { state.config.deletion_suffix, "RedlineDeleteMarker" },
        },
        virt_text_pos = "inline",
        right_gravity = false,
        priority = 210,
      })
    end
  end
end

function M.refresh(bufnr)
  bufnr = bufnr or vim.api.nvim_get_current_buf()
  local view = view_state(bufnr)
  if view.active_mode ~= "old" then
    save_new_view(bufnr)
  end

  local name = vim.api.nvim_buf_get_name(bufnr)
  if name == "" then
    state.by_buf[bufnr] = nil
    render(bufnr)
    return
  end

  local lines, err = system_lines(diff_command(name))
  if not lines then
    notify(err or "failed to read diff", vim.log.levels.WARN)
    return
  end

  state.by_buf[bufnr] = hunks_to_changes(parse_diff(lines))
  render(bufnr)
end

local function apply_mode(bufnr, mode)
  if mode == "old" then
    return show_old_view(bufnr)
  end

  restore_new_view(bufnr)
  vim.api.nvim_buf_clear_namespace(bufnr, ns, 0, -1)

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
    render(bufnr)
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
  restore_new_view(bufnr)
  state.enabled = false
  vim.api.nvim_buf_clear_namespace(bufnr, ns, 0, -1)
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

  local lnum = current_cursor_lnum()
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
  set_highlights()

  local group = vim.api.nvim_create_augroup("redline_nvim", { clear = true })

  vim.api.nvim_create_autocmd({ "CursorMoved", "CursorMovedI" }, {
    group = group,
    callback = function(args)
      if state.enabled and state.config.mode == "redline" and state.config.deletion_visibility == "cursor" then
        render(args.buf)
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
