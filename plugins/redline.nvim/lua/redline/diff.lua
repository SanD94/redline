local M = {}

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

function M.parse_diff(lines)
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
      end
    end
  end

  return hunks
end

local function add_addition_span(changes, lnum, new_text, first_token, last_token, kind)
  if not first_token or not last_token then
    return
  end

  local start_col = first_token.start_col
  local end_col = last_token.end_col

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

function M.hunks_to_changes(hunks)
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

return M
